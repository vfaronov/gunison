package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var (
	engine = NewEngine()

	unison  *exec.Cmd
	unisonR io.ReadCloser
	unisonW io.WriteCloser

	statusbar   *gtk.Statusbar
	spinner     *gtk.Spinner
	progressbar *gtk.ProgressBar

	statusbarContextID uint

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
		engine.RecvExit(fmt.Errorf("failed to create input pipe: %w", err))
		return
	}

	var pipeW *os.File
	unisonR, pipeW, err = os.Pipe()
	if err != nil {
		engine.RecvExit(fmt.Errorf("failed to create output pipe: %w", err))
		return
	}
	unison.Stdout = pipeW
	unison.Stderr = pipeW

	log.Printf("starting %v", unison)
	if err := unison.Start(); err != nil {
		engine.RecvExit(fmt.Errorf("failed to start unison: %w", err))
		return
	}
	shouldf(pipeW.Close(), "close pipeW")
	go consumeOutput()
	go consumeExit()
}

func consumeOutput() {
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

func consumeExit() {
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
	statusbar = mustGetObject(builder, "statusbar").(*gtk.Statusbar)
	statusbarContextID = statusbar.GetContextId("main")
	spinner = mustGetObject(builder, "spinner").(*gtk.Spinner)
	progressbar = mustGetObject(builder, "progressbar").(*gtk.ProgressBar)
	update(Update{})
	window.ShowAll()
}

func recvOutput(d []byte) {
	log.Printf("receiving %d bytes of output", len(d))
	update(engine.RecvOutput(d))
}

func recvError(err error) {
	log.Println("receiving unison I/O error:", err)
	update(engine.RecvError(err))
}

func recvExit(e error) {
	if e == success {
		e = nil
	}
	log.Println("receiving unison exit:", e)
	update(engine.RecvExit(e))
}

func update(upd Update) {
	statusbar.RemoveAll(statusbarContextID)
	if engine.Status != "" {
		statusbar.Push(statusbarContextID, engine.Status)
	}

	progressbar.SetText(engine.Current)
	if upd.Progressed {
		progressbar.SetVisible(true)
		progressbar.Pulse()
	}
	if engine.Progress >= 0 {
		progressbar.SetVisible(true)
		progressbar.SetFraction(engine.Progress)
	}

	if engine.Working {
		spinner.Start()
	} else {
		spinner.Stop()
		progressbar.SetVisible(false)
	}

	if len(upd.Send) > 0 {
		if _, err := unisonW.Write(upd.Send); err != nil {
			recvError(err)
		}
	}
}

func mustf(err error, format string, args ...interface{}) {
	if err != nil {
		log.Panicf("failed to "+format+": %s", append(args, err))
	}
}

func shouldf(err error, format string, args ...interface{}) {
	if err != nil {
		log.Printf("failed to "+format+": %s", append(args, err))
	}
}

type Connector interface {
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)
}

func shouldConnect(obj Connector, detailedSignal string, f interface{}, userData ...interface{}) {
	_, err := obj.Connect(detailedSignal, f, userData...)
	shouldf(err, "Connect(%#v, %#v)", detailedSignal, f)
}

func shouldIdleAdd(f interface{}, args ...interface{}) {
	_, err := glib.IdleAdd(f, args...)
	shouldf(err, "IdleAdd(%#v)", f)
}

func mustGetObject(b *gtk.Builder, name string) glib.IObject {
	obj, err := b.GetObject(name)
	mustf(err, "GetObject(%#v)", name)
	return obj
}
