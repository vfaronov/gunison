package main

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/gotk3/gotk3/gtk"
	"github.com/skratchdot/open-golang/open"
)

var (
	core = NewCore()

	unison  *exec.Cmd
	unisonR io.ReadCloser
	unisonW io.WriteCloser

	//go:embed gunison.glade
	ui string

	window              *gtk.Window
	headerbar           *gtk.HeaderBar
	infobar             *gtk.InfoBar
	infobarLabel        *gtk.Label
	treeview            *gtk.TreeView
	treeSelection       *gtk.TreeSelection
	treestore           *gtk.TreeStore
	pathColumn          *gtk.TreeViewColumn
	leftColumn          *gtk.TreeViewColumn
	actionColumn        *gtk.TreeViewColumn
	rightColumn         *gtk.TreeViewColumn
	itemMenu            *gtk.Menu
	leftToRightMenuItem *gtk.MenuItem
	rightToLeftMenuItem *gtk.MenuItem
	mergeMenuItem       *gtk.MenuItem
	skipMenuItem        *gtk.MenuItem
	diffMenuItem        *gtk.MenuItem
	statusLabel         *gtk.Label
	spinner             *gtk.Spinner
	progressbar         *gtk.ProgressBar
	syncButton          *gtk.Button
	abortButton         *gtk.Button
	killButton          *gtk.Button
	closeButton         *gtk.Button

	messages = []Message{}
	wantQuit bool

	success = errors.New("success")
)

func init() {
	// From https://developer.gnome.org/gdk3/stable/gdk3-Threads.html:
	//
	//	GTK+ [...] is not thread safe. You should only use GTK+ and GDK from the thread gtk_init()
	//	and gtk_main() were called on. This is usually referred to as the “main thread”.
	//
	// Calling LockOSThread in init guarantees that gtk.Init() and gtk.Main() below will run
	// only in our main thread.
	runtime.LockOSThread()
}

func main() {
	log.SetFlags(0)
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
		recvError(fmt.Errorf("Failed to create input pipe: %w", err))
		return
	}

	var pipeW *os.File
	unisonR, pipeW, err = os.Pipe()
	if err != nil {
		recvError(fmt.Errorf("Failed to create output pipe: %w", err))
		return
	}
	unison.Stdout = pipeW
	unison.Stderr = pipeW

	log.Printf("starting %v", unison)
	if err := unison.Start(); err != nil {
		recvError(err)
		return
	}
	shouldf(pipeW.Close(), "close pipeW")
	go watchUnison()
	update(core.ProcStart())
}

func watchUnison() {
	// When a large diff is being pumped through this buffer, after each Read the core will run its
	// regexp on the *entire* diff received so far (until eventually it finds the diff's end).
	// The smaller the buffer, the smaller the diffs we can handle until drowning in these regexp runs.
	// TODO: This could be solved by postponing core.ProcOutput until we get a short read or a timeout
	// (i.e. probably no more data is coming from Unison), but I'm not sure this is worthwhile,
	// because it's more convenient to use a visual -diff (such as Meld) anyway.
	var buf [65536]byte
	for {
		n, err := unisonR.Read(buf[:])
		log.Printf("Unison output: %q %v", buf[:n], err)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			shouldIdleAdd(recvOutput, data)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				shouldIdleAdd(recvError, err)
			}
			break
		}
	}
	shouldf(unisonR.Close(), "close unisonR pipe")

	e := unison.Wait()
	log.Println("Unison exit:", e)
	if e == nil {
		e = success // nil doesn't seem to work with IdleAdd
	}
	shouldIdleAdd(recvExit, e)
}

func setupWidgets() {
	builder, err := gtk.BuilderNewFromString(ui)
	mustf(err, "load GtkBuilder")

	window = mustGetObject(builder, "window").(*gtk.Window)
	shouldConnect(window, "delete-event", onWindowDeleteEvent)
	shouldConnect(window, "destroy", gtk.MainQuit)

	infobar = mustGetObject(builder, "infobar").(*gtk.InfoBar)
	shouldConnect(infobar, "response", onInfobarResponse)

	infobarLabel = mustGetObject(builder, "infobar-label").(*gtk.Label)

	headerbar = mustGetObject(builder, "headerbar").(*gtk.HeaderBar)

	treeview = mustGetObject(builder, "treeview").(*gtk.TreeView)
	shouldConnect(treeview, "popup-menu", onTreeviewPopupMenu)
	shouldConnect(treeview, "button-press-event", onTreeviewButtonPressEvent)
	shouldConnect(treeview, "query-tooltip", onTreeviewQueryTooltip)

	treeSelection = mustGetObject(builder, "tree-selection").(*gtk.TreeSelection)
	shouldConnect(treeSelection, "changed", onTreeSelectionChanged)

	treestore = mustGetObject(builder, "treestore").(*gtk.TreeStore)
	pathColumn = mustGetObject(builder, "path-column").(*gtk.TreeViewColumn)
	leftColumn = mustGetObject(builder, "left-column").(*gtk.TreeViewColumn)
	actionColumn = mustGetObject(builder, "action-column").(*gtk.TreeViewColumn)
	rightColumn = mustGetObject(builder, "right-column").(*gtk.TreeViewColumn)

	itemMenu = mustGetObject(builder, "item-menu").(*gtk.Menu)
	leftToRightMenuItem = mustGetObject(builder, "left-to-right-menuitem").(*gtk.MenuItem)
	shouldConnect(leftToRightMenuItem, "activate", onLeftToRightMenuItemActivate)
	rightToLeftMenuItem = mustGetObject(builder, "right-to-left-menuitem").(*gtk.MenuItem)
	shouldConnect(rightToLeftMenuItem, "activate", onRightToLeftMenuItemActivate)
	mergeMenuItem = mustGetObject(builder, "merge-menuitem").(*gtk.MenuItem)
	shouldConnect(mergeMenuItem, "activate", onMergeMenuItemActivate)
	skipMenuItem = mustGetObject(builder, "skip-menuitem").(*gtk.MenuItem)
	shouldConnect(skipMenuItem, "activate", onSkipMenuItemActivate)
	diffMenuItem = mustGetObject(builder, "diff-menuitem").(*gtk.MenuItem)
	shouldConnect(diffMenuItem, "activate", onDiffMenuItemActivate)

	// For some reason GTK/Glade think xalign has a default of 0.5, so Glade optimizes it away from
	// the XML file upon saving.
	mustf(mustGetObject(builder, "action-renderer").(*gtk.CellRendererText).Set("xalign", 0.5), "set xalign")

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

	window.Show()
}

func recvOutput(d []byte) {
	log.Printf("processing %d bytes of output", len(d))
	update(core.ProcOutput(d))
}

func recvError(err error) {
	log.Println("processing Unison I/O error:", err)
	update(core.ProcError(err))
}

func recvExit(e error) {
	if e == success {
		e = nil
	}
	code := 0
	if ee, ok := e.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if e != nil {
		code = -1
	}
	log.Println("processing Unison exit:", code, e)
	update(core.ProcExit(code, e))
}

func update(upd Update) {
	log.Printf("applying update: %+v (wantQuit = %v)", upd, wantQuit)

	if wantQuit && !core.Running {
		window.Destroy()
		return
	}

	if upd.Diff != nil {
		displayDiff(upd.Diff)
	}

	if core.Left != "" && core.Right != "" {
		headerbar.SetSubtitle(core.Left + " — " + core.Right)
		leftColumn.SetTitle(core.Left)
		rightColumn.SetTitle(core.Right)
	}

	if core.Items != nil && !treeview.GetVisible() {
		displayItems()
		treeview.SetVisible(true)
		treeview.GrabFocus()
		ClearCursor(treeview)
	}

	updateMenuItems()

	spinner.SetVisible(core.Busy)
	statusLabel.SetText(core.Status)
	progressbar.SetVisible(core.Progress != "")
	FitText(progressbar, core.Progress)
	if core.ProgressFraction >= 0 {
		progressbar.SetFraction(core.ProgressFraction)
	} else if upd.Progressed {
		progressbar.Pulse()
	}

	messages = append(messages, upd.Messages...)
	updateInfobar()

	syncButton.SetVisible(core.Sync != nil)
	abortButton.SetVisible(core.Abort != nil)
	killButton.SetVisible(wantQuit && core.Kill != nil)
	closeButton.SetVisible(!core.Running)
	if closeButton.GetVisible() {
		closeButton.GrabFocus()
	}

	if len(upd.Input) > 0 {
		log.Printf("Unison input: %#v", upd.Input)
		if _, err := unisonW.Write(upd.Input); err != nil {
			recvError(fmt.Errorf("Failed to write to Unison: %w", err))
		}
	}

	if upd.Interrupt {
		log.Print("interrupting Unison")
		if err := unison.Process.Signal(os.Interrupt); err != nil {
			recvError(fmt.Errorf("Failed to interrupt Unison: %w", err))
		}
	}

	if upd.Kill {
		log.Print("killing Unison")
		if err := unison.Process.Kill(); err != nil {
			recvError(fmt.Errorf("Failed to kill Unison: %w", err))
		}
	}

	// This goes last because we better update everything before showing the dialog
	// (which itself will, moreover, trigger another update).
	if upd.Alert.Text != "" {
		showAlert(upd.Alert)
	}
}

func updateInfobar() {
	if len(messages) == 0 {
		shouldf(infobar.Set("revealed", false), "occlude infobar")
		return
	}
	var text strings.Builder
	importance := Info
	for i, msg := range messages {
		if i > 0 {
			text.WriteByte('\n')
		}
		text.WriteString(msg.Text)
		if msg.Importance > importance {
			importance = msg.Importance
		}
	}
	infobarLabel.SetText(text.String())
	infobar.SetMessageType(importanceToMessageType[importance])
	shouldf(infobar.Set("revealed", true), "reveal infobar")
}

func showAlert(a Alert) {
	resp := Dialog(importanceToMessageType[a.Importance], a.Text,
		DialogOption{Text: "Abort", Response: gtk.RESPONSE_REJECT},
		DialogOption{Text: "Proceed", Response: gtk.RESPONSE_ACCEPT},
	)
	if resp == gtk.RESPONSE_ACCEPT {
		update(a.Proceed())
	} else { // including RESPONSE_NONE (Esc), etc.
		update(a.Abort())
	}
}

var importanceToMessageType = map[Importance]gtk.MessageType{
	Info:    gtk.MESSAGE_INFO,
	Warning: gtk.MESSAGE_WARNING,
	Error:   gtk.MESSAGE_ERROR,
}

func displayDiff(diff []byte) {
	f, err := os.CreateTemp("", "gunison-*.diff")
	if !checkf(err, "write diff to temporary file") {
		return
	}
	_, err = f.Write(diff)
	if !checkf(err, "write diff to temporary file") {
		return
	}
	name := f.Name()
	checkf(open.Start(name), "display diff file: %v", name)
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

func onInfobarResponse() {
	messages = messages[:0]
	updateInfobar()
}

func onSyncButtonClicked() {
	treeSelection.UnselectAll() // looks better
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

func checkf(err error, format string, args ...interface{}) bool { // TODO: vs. shouldf
	if false { // enable govet printf checking
		log.Printf(format, args...)
	}
	if err != nil {
		messages = append(messages, Message{
			Text:       fmt.Sprintf("Failed to "+format+": %s", append(args, err)...),
			Importance: Error,
		})
		updateInfobar()
		return false
	}
	return true
}
