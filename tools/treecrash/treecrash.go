// Program treecrash reproduces a gotk3 crash that seems to also affect Gunison under some circumstances.
//	go run treecrash.go 10000 5
// Select all items (e.g. with Ctrl+A) and click "Modify". This reliably crashes on my system.
// With fewer rows (first argument) and/or columns (second argument), sometimes it doesn't crash
// on the first click, but instead sometimes it fails to process all nodes on every click
// (just silently stops in the middle), and with repeated clicks, eventually crashes anyway.
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func main() {
	nitems, err := strconv.Atoi(os.Args[1])
	check(err)
	ncols, err := strconv.Atoi(os.Args[2])
	check(err)

	gtk.Init(nil)

	window, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	check(err)
	window.SetDefaultSize(500, 500)
	_, err = window.Connect("destroy", func() { gtk.MainQuit() })
	check(err)

	hbox, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	check(err)
	window.Add(hbox)

	scrollwin, err := gtk.ScrolledWindowNew(nil, nil)
	check(err)
	hbox.PackStart(scrollwin, true, true, 0)

	view, err := gtk.TreeViewNew()
	check(err)
	scrollwin.Add(view)
	renderer, err := gtk.CellRendererTextNew()
	check(err)
	for i := 0; i < ncols; i++ {
		column, err := gtk.TreeViewColumnNewWithAttribute("Name", renderer, "text", i)
		check(err)
		view.AppendColumn(column)
	}

	var colTypes []glib.Type
	for i := 0; i < ncols; i++ {
		colTypes = append(colTypes, glib.TYPE_STRING)
	}
	store, err := gtk.ListStoreNew(colTypes...)
	check(err)
	for i := 0; i < nitems; i++ {
		iter := store.Append()
		for j := 0; j < ncols; j++ {
			check(store.SetValue(iter, j, fmt.Sprintf("item%d", i)))
		}
	}
	view.SetModel(store)

	selection, err := view.GetSelection()
	check(err)
	selection.SetMode(gtk.SELECTION_MULTIPLE)

	button, err := gtk.ButtonNew()
	check(err)
	button.SetLabel("Modify")
	_, err = button.Connect("clicked", func() {
		for li := selection.GetSelectedRows(store); li != nil; li = li.Next() {
			iter, err := store.GetIter(li.Data().(*gtk.TreePath))
			check(err)
			gv, err := store.GetValue(iter, 0)
			check(err)
			name, err := gv.GoValue()
			check(err)
			fmt.Println(name)
			check(store.SetValue(iter, 0, name.(string)+"'"))
		}
	})
	check(err)
	hbox.PackEnd(button, false, false, 0)

	window.ShowAll()
	gtk.Main()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
