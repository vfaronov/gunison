package main

import (
	"fmt"
	"log"
	"reflect"
	"regexp"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func mustf(err error, format string, args ...interface{}) {
	if false { // enable govet printf checking
		log.Panicf(format, args...)
	}
	if err != nil {
		log.Panicf("failed to "+format+": %s", append(args, err)...)
	}
}

func shouldf(err error, format string, args ...interface{}) bool {
	if false { // enable govet printf checking
		log.Printf(format, args...)
	}
	if err != nil {
		log.Printf("failed to "+format+": %s", append(args, err)...)
		return false
	}
	return true
}

// SliceString returns slices of data between pairs of indices given in idx, ignoring -1.
func SliceString(data []byte, idx []int) []string {
	ss := make([]string, len(idx)/2)
	for i := range ss {
		if idx[2*i] == -1 {
			continue
		}
		ss[i] = string(data[idx[2*i]:idx[2*i+1]])
	}
	return ss
}

type Connector interface {
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)
}

func shouldConnect(obj Connector, detailedSignal string, f interface{}, userData ...interface{}) bool { //nolint:unparam
	_, err := obj.Connect(detailedSignal, f, userData...)
	return shouldf(err, "Connect(%#v, %#v)", detailedSignal, f)
}

func shouldIdleAdd(f interface{}, args ...interface{}) bool { //nolint:unparam
	handle, err := glib.IdleAdd(f, args...)
	log.Println("IdleAdd:", handle, err)
	return shouldf(err, "IdleAdd(%#v)", f)
}

func mustGetObject(b *gtk.Builder, name string) glib.IObject {
	obj, err := b.GetObject(name)
	mustf(err, "GetObject(%#v)", name)
	return obj
}

func Dialog(mType gtk.MessageType, msg string, options ...DialogOption) gtk.ResponseType {
	dlg := gtk.MessageDialogNew(window, gtk.DIALOG_DESTROY_WITH_PARENT, mType, gtk.BUTTONS_NONE, "%s", msg)
	defer dlg.Destroy()
	for _, opt := range options {
		_, err := dlg.AddButton(opt.Text, opt.Response)
		if !shouldf(err, "add button %q", opt.Text) {
			return options[0].Response
		}
		if opt.IsDefault {
			dlg.SetDefaultResponse(opt.Response)
		}
	}
	return dlg.Run()
}

type DialogOption struct {
	Text      string
	Response  gtk.ResponseType
	IsDefault bool
	// TODO: also mark button with suggested-action/destructive-action
}

const (
	// For GTK signal handlers, more readable than "return false" / "return true".
	handleDefault = false
	blockDefault  = true
)

func MustGetColumn(store *gtk.TreeStore, iter *gtk.TreeIter, column int) interface{} {
	gv, err := store.GetValue(iter, column)
	if err != nil {
		panic(fmt.Sprintf("failed to get value from column %v: %s", column, err))
	}
	v, err := gv.GoValue()
	if err != nil {
		panic(fmt.Sprintf("failed to get Go value from column %v value %v: %s", column, v, err))
	}
	return v
}

// ClearCursor removes the cursor indicating focus on treeview, while keeping the focus itself.
func ClearCursor(treeview *gtk.TreeView) error {
	treesel, err := treeview.GetSelection()
	if err != nil {
		return fmt.Errorf("failed to get tree selection: %w", err)
	}
	for {
		path, _ := treeview.GetCursor()
		if path == nil {
			return nil
		}
		path.Down()
		// As we keep descending, eventually the path becomes invalid, and
		// gtk_tree_view_set_cursor's documented "unset" behavior kicks in.
		treeview.SetCursor(path, nil, false)
		treesel.UnselectAll()
	}
}

// AnyOf returns a regexp pattern that matches and captures any of the keys in m,
// which must be a map with string keys.
func AnyOf(m interface{}) string {
	pat := "("
	first := true
	iter := reflect.ValueOf(m).MapRange()
	for iter.Next() {
		if !first {
			pat += "|"
		}
		first = false
		pat += regexp.QuoteMeta(iter.Key().Interface().(string))
	}
	pat += ")"
	return pat
}
