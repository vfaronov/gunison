package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/gotk3/gotk3/gtk"
)

var (
	core = NewCore()

	unison  *exec.Cmd
	unisonR io.ReadCloser
	unisonW io.WriteCloser

	plan = make(map[string]Action)

	window      *gtk.Window
	headerbar   *gtk.HeaderBar
	treeview    *gtk.TreeView
	treestore   *gtk.TreeStore
	statusLabel *gtk.Label
	spinner     *gtk.Spinner
	progressbar *gtk.ProgressBar
	syncButton  *gtk.Button
	abortButton *gtk.Button
	killButton  *gtk.Button
	closeButton *gtk.Button

	wantQuit bool

	success = errors.New("success")
)

func main() {
	gtk.Init(nil)
	setupWidgets()
	startUnison(os.Args[1:]...)
	log.Print("starting main loop")
	gtk.Main()
}

func startUnison(args ...string) {
	var err error

	args = append(args, "-dumbtty")
	unison = exec.Command("unison", args...)
	unison.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	unisonW, err = unison.StdinPipe()
	if err != nil {
		recvError(fmt.Errorf("failed to create input pipe: %w", err))
		return
	}

	var pipeW *os.File
	unisonR, pipeW, err = os.Pipe()
	if err != nil {
		recvError(fmt.Errorf("failed to create output pipe: %w", err))
		return
	}
	unison.Stdout = pipeW
	unison.Stderr = pipeW

	log.Printf("starting %v", unison)
	if err := unison.Start(); err != nil {
		recvError(fmt.Errorf("failed to start unison: %w", err))
		return
	}
	shouldf(pipeW.Close(), "close pipeW")
	go watchOutput()
	go watchExit()
	if core.ProcStart != nil {
		update(core.ProcStart())
	} else {
		log.Println("core is not ready to process unison start")
	}
}

func watchOutput() {
	var buf [4096]byte
	for {
		n, err := unisonR.Read(buf[:])
		log.Printf("unison output: %q %v", buf[:n], err)
		if n > 0 {
			shouldIdleAdd(recvOutput, buf[:n])
		}
		if err != nil {
			shouldIdleAdd(recvError, err)
			break
		}
	}
	shouldf(unisonR.Close(), "close unisonR pipe")
}

func watchExit() {
	e := unison.Wait()
	log.Println("unison exit:", e)
	if e == nil {
		e = success // nil doesn't seem to work with IdleAdd
	}
	shouldIdleAdd(recvExit, e)
}

func setupWidgets() {
	builder, err := gtk.BuilderNewFromFile("/home/vasiliy/cur/gunison/gunison/gunison.glade") // +FIXME
	mustf(err, "load GtkBuilder")

	window = mustGetObject(builder, "window").(*gtk.Window)
	shouldConnect(window, "delete-event", onWindowDeleteEvent)
	shouldConnect(window, "destroy", gtk.MainQuit)

	headerbar = mustGetObject(builder, "headerbar").(*gtk.HeaderBar)

	treeview = mustGetObject(builder, "treeview").(*gtk.TreeView)
	treestore = mustGetObject(builder, "treestore").(*gtk.TreeStore)

	statusLabel = mustGetObject(builder, "status-label").(*gtk.Label)

	spinner = mustGetObject(builder, "spinner").(*gtk.Spinner)

	progressbar = mustGetObject(builder, "progressbar").(*gtk.ProgressBar)

	syncButton = mustGetObject(builder, "sync-button").(*gtk.Button)
	shouldConnect(syncButton, "clicked", onSyncButtonClicked)

	abortButton = mustGetObject(builder, "abort-button").(*gtk.Button)
	shouldConnect(abortButton, "clicked", onAbortButtonClicked)

	killButton = mustGetObject(builder, "kill-button").(*gtk.Button)
	shouldConnect(killButton, "clicked", onKillButtonClicked)

	closeButton = mustGetObject(builder, "close-button").(*gtk.Button)
	shouldConnect(closeButton, "clicked", onCloseButtonClicked)

	update(Update{})

	window.ShowAll()
}

func recvOutput(d []byte) {
	if core.ProcOutput == nil {
		return
	}
	log.Printf("processing %d bytes of output", len(d))
	update(core.ProcOutput(d))
}

func recvError(err error) {
	if core.ProcError == nil {
		return
	}
	log.Println("processing unison I/O error:", err)
	update(core.ProcError(err))
}

func recvExit(e error) {
	if e == success {
		e = nil
	}
	if core.ProcExit == nil {
		return
	}
	log.Println("processing unison exit:", e)
	code := -1
	if ee, ok := e.(*exec.ExitError); ok {
		code = ee.ExitCode()
	}
	update(core.ProcExit(code, e))
}

func update(upd Update) {
	if wantQuit && !core.Running {
		window.Destroy()
		return
	}

	if core.Left != "" && core.Right != "" {
		headerbar.SetSubtitle(core.Left + " → " + core.Right) // TODO: is it always '→'?
	}

	if upd.Items != nil {
		displayItems(upd.Items)
		treeview.SetVisible(true)
	}

	spinner.SetVisible(core.Busy)
	statusLabel.SetText(core.Status)
	progressbar.SetVisible(core.Progress != "")
	FitText(progressbar, core.Progress)
	if core.ProgressFraction >= 0 {
		progressbar.SetFraction(core.ProgressFraction)
	} else if upd.Progressed {
		progressbar.Pulse()
	}

	syncButton.SetVisible(core.Sync != nil)
	abortButton.SetVisible(core.Abort != nil)
	killButton.SetVisible(wantQuit && core.Kill != nil)
	closeButton.SetVisible(!core.Running)

	if len(upd.Input) > 0 {
		log.Printf("unison input: %#v", upd.Input)
		if _, err := unisonW.Write(upd.Input); err != nil {
			log.Printf("failed to write to unison: %v", err)
			recvError(err)
		}
	}

	if upd.Interrupt {
		log.Print("interrupting unison")
		if err := unison.Process.Signal(os.Interrupt); err != nil {
			log.Printf("failed to interrupt unison: %v", err)
			recvError(err)
		}
	}

	if upd.Kill {
		log.Printf("killing unison")
		if err := unison.Process.Kill(); err != nil {
			log.Printf("failed to kill unison: %v", err)
			recvError(err)
		}
	}
}

func onWindowDeleteEvent() bool {
	switch {
	case !core.Running:
		return handleDefault

	case core.Quit != nil:
		wantQuit = true
		update(core.Quit())

	case core.Interrupt != nil:
		resp := Dialog(gtk.MESSAGE_QUESTION, "Interrupt Unison?",
			DialogOption{Text: "_Keep running", Response: gtk.RESPONSE_NO},
			DialogOption{Text: "_Interrupt", Response: gtk.RESPONSE_YES},
		)
		if resp == gtk.RESPONSE_YES {
			wantQuit = true
			update(core.Interrupt())
		}

	case core.Kill != nil:
		resp := Dialog(gtk.MESSAGE_QUESTION, "Unison is still running. Force it to stop?",
			DialogOption{Text: "_Keep running", Response: gtk.RESPONSE_NO},
			DialogOption{Text: "_Force stop", Response: gtk.RESPONSE_YES},
		)
		if resp == gtk.RESPONSE_YES {
			wantQuit = true
			update(core.Kill())
		}
	}

	return blockDefault
}

func onSyncButtonClicked() {
	update(core.Sync())
}

func onAbortButtonClicked() {
	resp := Dialog(gtk.MESSAGE_QUESTION, "Abort the operation?",
		DialogOption{Text: "_Keep running", Response: gtk.RESPONSE_NO},
		DialogOption{Text: "_Abort", Response: gtk.RESPONSE_YES},
	)
	if resp == gtk.RESPONSE_YES {
		update(core.Abort())
	}
}

func onKillButtonClicked() {
	update(core.Kill())
}

func onCloseButtonClicked() {
	window.Destroy()
}
