package main

import (
	"log"

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
