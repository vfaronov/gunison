package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/skratchdot/open-golang/open"
)

var (
	core = NewCore()

	unison      *exec.Cmd
	unisonR     io.ReadCloser
	unisonW     io.WriteCloser
	sysProcAttr *syscall.SysProcAttr

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
	columns             []*gtk.TreeViewColumn
	itemMenu            *gtk.Menu
	leftToRightMenuItem *gtk.MenuItem
	rightToLeftMenuItem *gtk.MenuItem
	mergeMenuItem       *gtk.MenuItem
	skipMenuItem        *gtk.MenuItem
	revertMenuItem      *gtk.MenuItem
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

	collapsed = map[string]bool{} // TODO: use a more efficient structure for this, like a trie?
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
	loadUIState()
	window.Show()
	startUnison(os.Args[1:]...)
	log.Println("starting main loop")
	gtk.Main()
	// saveUIState is not called here (unlike loadUIState), because it needs the current window size,
	// which is not available when the window has already been destroyed.
}

func startUnison(args ...string) {
	var err error

	args = append(args, "-dumbtty")
	unison = exec.Command("unison", args...)
	unison.SysProcAttr = sysProcAttr

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
	var buf [65536]byte // has to be rather large due to https://github.com/vfaronov/gunison/issues/1
	for {
		n, err := unisonR.Read(buf[:])
		log.Printf("Unison output: %d bytes: %q %v", n, buf[:n], err)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			// TODO: use a different mechanism for communicating with the main thread:
			// see https://discourse.gnome.org/t/g-idle-add-ordering/6088
			glib.IdleAdd(func() { recvOutput(data) })
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				glib.IdleAdd(func() { recvError(err) })
			}
			break
		}
	}
	shouldf(unisonR.Close(), "close unisonR pipe")

	// TODO: This is incorrect. This normally works because unisonR automatically EOFs when all
	// descriptors for its write end are closed, i.e. when Unison and its children (that inherit
	// stdout/stderr, such as ssh) exit. This is how pipes work on Linux, at least. But if Unison
	// leaks its end of the pipe to some process that doesn't exit, or if we run on a platform
	// where pipes work differently, we might never get to this line.
	e := unison.Wait()
	log.Println("Unison exit:", e)
	glib.IdleAdd(func() { recvExit(e) })
}

func setupWidgets() {
	builder, err := gtk.BuilderNewFromString(ui)
	mustf(err, "load GtkBuilder")

	window = mustGetObject(builder, "window").(*gtk.Window)
	window.Connect("delete-event", onWindowDeleteEvent)
	window.Connect("destroy", gtk.MainQuit)

	infobar = mustGetObject(builder, "infobar").(*gtk.InfoBar)
	infobar.Connect("response", onInfobarResponse)

	infobarLabel = mustGetObject(builder, "infobar-label").(*gtk.Label)

	headerbar = mustGetObject(builder, "headerbar").(*gtk.HeaderBar)

	treeview = mustGetObject(builder, "treeview").(*gtk.TreeView)
	treeview.Connect("popup-menu", onTreeviewPopupMenu)
	treeview.Connect("button-press-event", onTreeviewButtonPressEvent)
	treeview.Connect("query-tooltip", onTreeviewQueryTooltip)
	treeview.Connect("row-expanded", onTreeviewRowExpanded)
	treeview.Connect("row-collapsed", onTreeviewRowCollapsed)

	treeSelection = mustGetObject(builder, "tree-selection").(*gtk.TreeSelection)
	treeSelection.Connect("changed", onTreeSelectionChanged)

	treestore = mustGetObject(builder, "treestore").(*gtk.TreeStore)
	pathColumn = mustGetObject(builder, "path-column").(*gtk.TreeViewColumn)
	pathColumn.Connect("clicked", onPathColumnClicked)
	leftColumn = mustGetObject(builder, "left-column").(*gtk.TreeViewColumn)
	actionColumn = mustGetObject(builder, "action-column").(*gtk.TreeViewColumn)
	actionColumn.Connect("clicked", onActionColumnClicked)
	rightColumn = mustGetObject(builder, "right-column").(*gtk.TreeViewColumn)
	// Pin down the original order of columns (before the user reorders them) for loadUIState/saveUIState.
	for li, next := Iter(treeview.GetColumns()); li != nil; li = next() {
		columns = append(columns, li.Data().(*gtk.TreeViewColumn))
	}

	itemMenu = mustGetObject(builder, "item-menu").(*gtk.Menu)
	leftToRightMenuItem = mustGetObject(builder, "left-to-right-menuitem").(*gtk.MenuItem)
	leftToRightMenuItem.Connect("activate", onLeftToRightMenuItemActivate)
	rightToLeftMenuItem = mustGetObject(builder, "right-to-left-menuitem").(*gtk.MenuItem)
	rightToLeftMenuItem.Connect("activate", onRightToLeftMenuItemActivate)
	mergeMenuItem = mustGetObject(builder, "merge-menuitem").(*gtk.MenuItem)
	mergeMenuItem.Connect("activate", onMergeMenuItemActivate)
	skipMenuItem = mustGetObject(builder, "skip-menuitem").(*gtk.MenuItem)
	skipMenuItem.Connect("activate", onSkipMenuItemActivate)
	revertMenuItem = mustGetObject(builder, "revert-menuitem").(*gtk.MenuItem)
	revertMenuItem.Connect("activate", onRevertMenuItemActivate)
	diffMenuItem = mustGetObject(builder, "diff-menuitem").(*gtk.MenuItem)
	diffMenuItem.Connect("activate", onDiffMenuItemActivate)

	// For some reason GTK/Glade think xalign has a default of 0.5, so Glade optimizes it away from
	// the XML file upon saving.
	mustf(mustGetObject(builder, "left-renderer").(*gtk.CellRendererText).Set("xalign", 0.5), "set xalign")
	mustf(mustGetObject(builder, "action-renderer").(*gtk.CellRendererText).Set("xalign", 0.5), "set xalign")
	mustf(mustGetObject(builder, "right-renderer").(*gtk.CellRendererText).Set("xalign", 0.5), "set xalign")

	statusLabel = mustGetObject(builder, "status-label").(*gtk.Label)

	spinner = mustGetObject(builder, "spinner").(*gtk.Spinner)

	progressbar = mustGetObject(builder, "progressbar").(*gtk.ProgressBar)

	syncButton = mustGetObject(builder, "sync-button").(*gtk.Button)
	syncButton.Connect("clicked", onSyncButtonClicked)

	abortButton = mustGetObject(builder, "abort-button").(*gtk.Button)
	abortButton.Connect("clicked", onAbortButtonClicked)

	killButton = mustGetObject(builder, "kill-button").(*gtk.Button)
	killButton.Connect("clicked", onKillButtonClicked)

	closeButton = mustGetObject(builder, "close-button").(*gtk.Button)
	closeButton.Connect("clicked", exit)

	update(Update{})
}

func recvOutput(d []byte) {
	log.Printf("processing Unison output: %d bytes", len(d))
	update(core.ProcOutput(d))
}

func recvError(err error) {
	log.Println("processing Unison I/O error:", err)
	update(core.ProcError(err))
}

func recvExit(e error) {
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
		exit()
		return
	}

	if upd.Diff != nil {
		displayDiff(upd.Diff)
	}

	if core.Left != "" && core.Right != "" {
		setReplicaNames(core.Left, core.Right)
	}

	if core.Items != nil && !treeview.GetVisible() {
		displayItems()
		treeview.SetVisible(true)
		treeview.GrabFocus()
		shouldf(ClearCursor(treeview), "clear treeview cursor")
	}

	updateMenuItems()

	spinner.SetVisible(core.Busy)
	statusLabel.SetText(core.Status)
	progressbar.SetVisible(core.Progress != "")
	progressbar.SetText(core.Progress)
	if core.ProgressFraction >= 0 {
		progressbar.SetFraction(core.ProgressFraction)
	} else if upd.Progressed {
		progressbar.Pulse()
	}

	messages = append(messages, upd.Messages...)
	updateInfobar()

	syncButton.SetVisible(core.Sync != nil)
	abortButton.SetVisible(core.Abort != nil)
	closeButton.SetVisible(!core.Running)
	if closeButton.GetVisible() {
		closeButton.GrabFocus()
	}

	// If we just show the kill button right away, like the other buttons above, it flashes menacingly
	// during normal quit, due to the brief delay between sending "q" to Unison and receiving its exit.
	if offerKill := func() bool { return wantQuit && core.Kill != nil }; offerKill() {
		if !killButton.GetVisible() {
			glib.TimeoutAdd(1000, func() { killButton.SetVisible(offerKill()) })
		}
	} else {
		killButton.SetVisible(false)
	}

	if len(upd.Input) > 0 {
		log.Printf("Unison input: %#v", upd.Input)
		if _, err := unisonW.Write(upd.Input); err != nil {
			recvError(fmt.Errorf("Failed to write to Unison: %w", err))
		}
	}

	if upd.Interrupt {
		log.Println("interrupting Unison")
		if err := SignalGroup(unison.Process, os.Interrupt); err != nil {
			recvError(fmt.Errorf("Failed to interrupt Unison: %w", err))
		}
	}

	if upd.Kill {
		log.Println("killing Unison")
		if err := SignalGroup(unison.Process, os.Kill); err != nil {
			recvError(fmt.Errorf("Failed to kill Unison: %w", err))
		}
	}

	// This goes last because we better update everything before showing the dialog
	// (which itself will, moreover, trigger another update).
	if upd.Alert.Text != "" {
		showAlert(upd.Alert)
	}
}

func invokeUpdate(f func() Update) {
	// Methods on core may have become nil while we were interacting with the user.
	if f == nil {
		log.Println("cannot invoke method because it is already nil")
		update(Update{})
		return
	}
	update(f())
}

func setReplicaNames(left, right string) {
	if headerbar.GetSubtitle() != "" {
		return // already set
	}
	headerbar.SetSubtitle(left + " — " + right)
	leftColumn.SetTitle(left)
	rightColumn.SetTitle(right)
	replaceIn := func(s string) string {
		s = strings.ReplaceAll(s, "left", left)
		s = strings.ReplaceAll(s, "right", right)
		return s
	}
	for k, s := range actionDescriptions {
		actionDescriptions[k] = replaceIn(s)
	}
	leftToRightMenuItem.SetLabel(replaceIn(leftToRightMenuItem.GetLabel()))
	rightToLeftMenuItem.SetLabel(replaceIn(rightToLeftMenuItem.GetLabel()))
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
	mustf(infobar.Set("revealed", true), "reveal infobar")
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

func exit() {
	saveUIState()
	window.Destroy()
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
		exit()

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
			invokeUpdate(core.Interrupt)
		}

	case core.Kill != nil:
		resp := Dialog(gtk.MESSAGE_QUESTION, "Unison is still running. Force it to stop?",
			DialogOption{Text: "_Keep running", Response: gtk.RESPONSE_NO},
			DialogOption{Text: "_Force stop", Response: gtk.RESPONSE_YES},
		)
		if resp == gtk.RESPONSE_YES {
			wantQuit = true
			invokeUpdate(core.Kill)
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
	invokeUpdate(core.Sync)
}

func onAbortButtonClicked() {
	resp := Dialog(gtk.MESSAGE_QUESTION, "Abort the operation?",
		DialogOption{Text: "_Keep running", Response: gtk.RESPONSE_NO},
		DialogOption{Text: "_Abort", Response: gtk.RESPONSE_YES},
	)
	if resp == gtk.RESPONSE_YES {
		invokeUpdate(core.Abort)
	}
}

func onKillButtonClicked() {
	invokeUpdate(core.Kill)
}

type uiState struct {
	Width, Height int
	Maximized     bool
	ColumnOrder   []int // indices match var columns
	ColumnWidth   []int // indices match var columns
	Collapsed     []string
}

func uiStatePath() string {
	base, err := os.UserConfigDir()
	mustf(err, "get user config dir")
	return filepath.Join(base, "gunison", "state.json")
}

func loadUIState() {
	statePath := uiStatePath()
	log.Println("loading UI state from", statePath)
	f, err := os.Open(statePath)
	if !shouldf(err, "open UI state file") {
		return
	}
	defer f.Close()
	var state uiState
	if err := json.NewDecoder(f).Decode(&state); !shouldf(err, "decode UI state JSON") {
		return
	}

	log.Printf("state: Width:%v Height:%v Maximized:%v ColumnOrder:%v ColumnWidth:%v",
		state.Width, state.Height, state.Maximized, state.ColumnOrder, state.ColumnWidth)

	window.SetDefaultSize(state.Width, state.Height)
	if state.Maximized {
		window.Maximize()
	}

	var prev *gtk.TreeViewColumn
	for ord := range state.ColumnOrder { // For each position in the order of columns,
		// find the column that should be at this position, and move it there.
		for i, x := range state.ColumnOrder {
			if ord == x {
				treeview.MoveColumnAfter(columns[i], prev)
				prev = columns[i]
				break
			}
		}
	}

	for i, column := range columns {
		column.SetFixedWidth(state.ColumnWidth[i])
	}

	collapsed = make(map[string]bool, len(state.Collapsed))
	for _, path := range state.Collapsed {
		collapsed[path] = true
	}
}

func saveUIState() {
	statePath := uiStatePath()
	log.Println("saving UI state to", statePath)
	if !shouldf(os.MkdirAll(filepath.Dir(statePath), 0755), "create UI state directory") {
		return
	}
	f, err := os.Create(statePath)
	if !shouldf(err, "create UI state file") {
		return
	}
	defer f.Close()
	var state uiState

	if window.IsMaximized() {
		state.Maximized = true
		state.Width, state.Height = window.GetDefaultSize()
	} else {
		state.Width, state.Height = window.GetSize()
	}

	state.ColumnOrder = make([]int, len(columns))
	ord := 0
	for li, next := Iter(treeview.GetColumns()); li != nil; li = next() {
		for i, column := range columns {
			if column.Native() == li.Data().(*gtk.TreeViewColumn).Native() {
				state.ColumnOrder[i] = ord
				break
			}
		}
		ord++
	}

	for _, column := range columns {
		width := column.GetWidth()
		if width == 0 { // treeview is not shown
			width = column.GetFixedWidth()
		}
		state.ColumnWidth = append(state.ColumnWidth, width)
	}

	state.Collapsed = make([]string, 0, len(collapsed))
	for path := range collapsed {
		state.Collapsed = append(state.Collapsed, path)
	}
	sort.Strings(state.Collapsed)

	log.Printf("state: Width:%v Height:%v Maximized:%v ColumnOrder:%v ColumnWidth:%v",
		state.Width, state.Height, state.Maximized, state.ColumnOrder, state.ColumnWidth)

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	shouldf(enc.Encode(state), "encode UI state JSON")
}

func checkf(err error, format string, args ...interface{}) bool {
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
