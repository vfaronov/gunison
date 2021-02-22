package main

import (
	"fmt"
	"log"
	"unicode/utf8"

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

type Connector interface {
	Connect(string, interface{}, ...interface{}) (glib.SignalHandle, error)
}

func shouldConnect(obj Connector, detailedSignal string, f interface{}, userData ...interface{}) bool { //nolint:unparam
	_, err := obj.Connect(detailedSignal, f, userData...)
	return shouldf(err, "Connect(%#v, %#v)", detailedSignal, f)
}

func shouldIdleAdd(f interface{}, args ...interface{}) bool { //nolint:unparam
	_, err := glib.IdleAdd(f, args...)
	return shouldf(err, "IdleAdd(%#v)", f)
}

func mustGetObject(b *gtk.Builder, name string) glib.IObject {
	obj, err := b.GetObject(name)
	mustf(err, "GetObject(%#v)", name)
	return obj
}

// FitText sets text on obj, proactively ellipsizing it so that it fits into obj's size request
// and thus doesn't cause it to grow.
func FitText(obj TextFitter, text string) {
	w, _ := obj.GetSizeRequest()
	// TODO: measure actual size of text with Pango instead of this arbitrary and crude approximation
	// (better yet, find a way to achieve the desired look without this crutch).
	maxChars := w / 12
	if chars := utf8.RuneCountInString(text); chars > maxChars {
		runes := []rune(text)
		text = string(runes[:maxChars/2]) + "…" + string(runes[chars-maxChars/2:])
	}
	obj.SetText(text)
}

type TextFitter interface {
	GetSizeRequest() (int, int)
	SetText(string)
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

// More readable than "return true" / "return false".
const (
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
