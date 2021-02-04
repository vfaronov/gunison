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

	statusLabel *gtk.Label
	spinner     *gtk.Spinner
	progressbar *gtk.ProgressBar

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
	window := mustGetObject(builder, "window").(*gtk.Window)
	shouldConnect(window, "destroy", gtk.MainQuit)
	statusLabel = mustGetObject(builder, "status-label").(*gtk.Label)
	spinner = mustGetObject(builder, "spinner").(*gtk.Spinner)
	progressbar = mustGetObject(builder, "progressbar").(*gtk.ProgressBar)
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
	statusLabel.SetText(engine.Status)

	FitText(progressbar, engine.Current)
	if upd.Progressed {
		progressbar.SetVisible(true)
		progressbar.Pulse()
	}
	if engine.Progress >= 0 {
		progressbar.SetVisible(true)
		progressbar.SetFraction(engine.Progress)
	}

	if engine.Busy {
		spinner.Start()
	} else {
		spinner.Stop()
		progressbar.Hide()
	}

	if len(upd.Input) > 0 {
		log.Printf("unison input: %#v", upd.Input)
		if _, err := unisonW.Write(upd.Input); err != nil {
			recvError(err)
		}
	}
}
