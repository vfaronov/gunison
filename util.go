package main

import (
	"log"
	"unicode/utf8"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

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

// fitText sets text on obj, proactively ellipsizing it so that it fits into obj's size request
// and thus doesn't cause it to grow.
func fitText(obj TextFitter, text string) {
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
