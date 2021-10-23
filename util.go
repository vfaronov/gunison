package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"

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

// SignalGroup tries to send sig to the process group whose leader is p,
// falling back on just p if this fails.
func SignalGroup(p *os.Process, sig os.Signal) error {
	if pg, err := os.FindProcess(-p.Pid); err == nil {
		if pg.Signal(sig) == nil {
			return nil
		}
	}
	return p.Signal(sig)
}

// Prefix iterates over successively longer prefixes of path consisting of whole numbers of segments.
// Prefix(path, 0) returns the first prefix and an index to be passed back for the next iteration.
// At the end, Prefix returns "", -1.
func Prefix(path string, i int) (string, int) {
	if i > len(path) {
		return "", -1
	}
	prefix := path[:i]
	var next int
	if i < len(path) {
		if j := strings.IndexByte(path[i+1:], '/'); j != -1 {
			next = i + 1 + j
		} else {
			next = len(path)
		}
	} else {
		next = len(path) + 1
	}
	return prefix, next
}

// Seen returns true if k has already been passed to it with the same m.
func Seen(m map[string]bool, k string) bool {
	if m[k] {
		return true
	}
	m[k] = true
	return false
}

// PathIsAncestor returns true if p1 is an ancestor of p2.
func PathIsAncestor(p1, p2 string) bool {
	return strings.HasPrefix(p2, p1+"/") || (p2 != "" && p1 == "")
}

// DeleteEnv returns vars ("key=value" strings) without the given keys. It does not modify vars.
func DeleteEnv(vars []string, keys ...string) []string {
	result := vars
	copied := false
varsLoop:
	for i := 0; i < len(result); {
		v := result[i]
		for _, key := range keys {
			if len(v) >= len(key)+1 && v[:len(key)] == key && v[len(key)] == '=' { // starts with "key="
				if !copied {
					result = make([]string, len(vars))
					copy(result, vars)
					copied = true
				}
				result[i] = result[len(result)-1]
				result = result[:len(result)-1]
				continue varsLoop
			}
		}
		i++
	}
	return result
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
		mustf(err, "add button %q", opt.Text)
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
	mustf(err, "get value from column %v", column)
	v, err := gv.GoValue()
	mustf(err, "get Go value from column %v value %v", column, v)
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

// DetachModel temporarily detaches the model from treeview for faster batch updates,
// and returns a function to restore it.
func DetachModel(treeview *gtk.TreeView) func() {
	model, err := treeview.GetModel()
	if !shouldf(err, "get treeview model") {
		return func() {}
	}
	searchColumn := treeview.GetSearchColumn() // is reset to -1 when the model is changed
	treeview.SetModel(nil)
	return func() {
		treeview.SetModel(model)
		treeview.SetSearchColumn(searchColumn)
	}
}

// DisplaySort makes view display a sort indicator (only) on column for order.
func DisplaySort(view *gtk.TreeView, column *gtk.TreeViewColumn, order gtk.SortType) {
	for li, next := Iter(view.GetColumns()); li != nil; li = next() {
		col := li.Data().(*gtk.TreeViewColumn)
		col.SetSortIndicator(column != nil && col.Native() == column.Native())
		col.SetSortOrder(order)
	}
}

// Iter helps iterating over a glib.List while keeping its head alive, preventing the finalizer
// from firing prematurely. See the comment on gtk.(*TreeSelection).GetSelectedRows.
func Iter(head *glib.List) (li *glib.List, next func() *glib.List) {
	li = head
	return li, func() *glib.List {
		runtime.KeepAlive(head)
		li = li.Next()
		return li
	}
}

// PreserveScroll computes the relative (percent) value of adj and
// schedules it to be restored on the next iteration of the GLib main loop.
// (It can't be restored on the same iteration because, at least in case of
// a treeview scroll, it will only be updated at the end.)
func PreserveScroll(adj *gtk.Adjustment) {
	relvalue := (adj.GetValue() - adj.GetLower()) / (adj.GetUpper() - adj.GetLower())
	if math.IsNaN(relvalue) { // all zero, nothing to restore
		return
	}
	glib.IdleAdd(func() { adj.SetValue(relvalue * (adj.GetUpper() - adj.GetLower())) })
}
