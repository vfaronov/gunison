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
	engine = NewEngine()

	unison  *exec.Cmd
	unisonR io.ReadCloser
	unisonW io.WriteCloser

	window      *gtk.Window
	headerbar   *gtk.HeaderBar
	statusLabel *gtk.Label
	spinner     *gtk.Spinner
	progressbar *gtk.ProgressBar
	syncButton  *gtk.Button
	killButton  *gtk.Button

	wantQuit bool

	success = errors.New("success")
)

func main() {
	gtk.Init(nil)
	startUnison(os.Args[1:]...)
	setupWidgets()
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
		engine.ProcExit(fmt.Errorf("failed to create input pipe: %w", err))
		return
	}

	var pipeW *os.File
	unisonR, pipeW, err = os.Pipe()
	if err != nil {
		engine.ProcExit(fmt.Errorf("failed to create output pipe: %w", err))
		return
	}
	unison.Stdout = pipeW
	unison.Stderr = pipeW

	log.Printf("starting %v", unison)
	if err := unison.Start(); err != nil {
		engine.ProcExit(fmt.Errorf("failed to start unison: %w", err))
		return
	}
	shouldf(pipeW.Close(), "close pipeW")
	go watchOutput()
	go watchExit()
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

	statusLabel = mustGetObject(builder, "status-label").(*gtk.Label)

	spinner = mustGetObject(builder, "spinner").(*gtk.Spinner)

	progressbar = mustGetObject(builder, "progressbar").(*gtk.ProgressBar)

	syncButton = mustGetObject(builder, "sync-button").(*gtk.Button)
	shouldConnect(syncButton, "clicked", onSyncButtonClicked)

	killButton = mustGetObject(builder, "kill-button").(*gtk.Button)
	shouldConnect(killButton, "clicked", onKillButtonClicked)

	update(Update{})

	window.ShowAll()
}

func recvOutput(d []byte) {
	log.Printf("receiving %d bytes of output", len(d))
	update(engine.ProcOutput(d))
}

func recvError(err error) {
	log.Println("receiving unison I/O error:", err)
	update(engine.ProcError(err))
}

func recvExit(e error) {
	if e == success {
		e = nil
	}
	log.Println("receiving unison exit:", e)
	update(engine.ProcExit(e))
}

func update(upd Update) {
	if wantQuit && engine.Finished {
		window.Destroy()
		return
	}

	if engine.Left != "" && engine.Right != "" {
		headerbar.SetSubtitle(engine.Left + " → " + engine.Right) // TODO: is it always '→'?
	}

	statusLabel.SetText(engine.Status)

	progressbar.SetVisible(engine.Progress != "")
	FitText(progressbar, engine.Progress)
	if engine.ProgressFraction >= 0 {
		progressbar.SetFraction(engine.ProgressFraction)
	} else if upd.Progressed {
		progressbar.Pulse()
	}

	if engine.Busy {
		spinner.Start()
	} else {
		spinner.Stop()
	}

	syncButton.SetVisible(engine.Sync != nil)
	killButton.SetVisible(wantQuit && engine.Kill != nil)

	if len(upd.Input) > 0 {
		log.Printf("unison input: %#v", upd.Input)
		if _, err := unisonW.Write(upd.Input); err != nil {
			recvError(err)
		}
	}

	if upd.Interrupt {
		log.Print("interrupting unison")
		if err := unison.Process.Signal(os.Interrupt); err != nil {
			recvError(err)
		}
	}

	if upd.Kill {
		log.Printf("killing unison")
		if err := unison.Process.Kill(); err != nil {
			recvError(err)
		}
	}
}

func onWindowDeleteEvent() bool {
	switch {
	case engine.Finished:
		return handleDefault

	case engine.Quit != nil:
		wantQuit = true
		update(engine.Quit())

	case engine.Interrupt != nil:
		resp := Dialog(gtk.MESSAGE_QUESTION, "Interrupt Unison?",
			DialogOption{Text: "_Keep running", Response: gtk.RESPONSE_NO},
			DialogOption{Text: "_Interrupt", Response: gtk.RESPONSE_YES},
		)
		if resp == gtk.RESPONSE_YES {
			wantQuit = true
			update(engine.Interrupt())
		}

	case engine.Kill != nil:
		resp := Dialog(gtk.MESSAGE_QUESTION, "Unison is still running. Force it to stop?",
			DialogOption{Text: "_Keep running", Response: gtk.RESPONSE_NO},
			DialogOption{Text: "_Force stop", Response: gtk.RESPONSE_YES},
		)
		if resp == gtk.RESPONSE_YES {
			wantQuit = true
			update(engine.Kill())
		}
	}

	return blockDefault
}

func onKillButtonClicked() {
	update(engine.Kill())
}

func onSyncButtonClicked() {
	update(engine.Sync(0))
}
