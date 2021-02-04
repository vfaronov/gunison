package main

import (
	"bytes"
)

type Engine struct {
	Finished         bool
	Busy             bool
	Status           string
	Progress         string  // empty string iff not progressing
	ProgressFraction float64 // 0 to 1; or -1 for unknown
	CanQuit          bool
	OfferKill        bool

	buf bytes.Buffer
}

type Update struct {
	Progressed bool
	Input      []byte
	Interrupt  bool
	Kill       bool
}

func NewEngine() *Engine {
	return &Engine{
		Busy:             true,
		Status:           "Starting Unison...",
		ProgressFraction: -1,
	}
}

func (e *Engine) ProcOutput(d []byte) Update {
	if e.Finished {
		return Update{}
	}
	e.buf.Write(d)
	e.Progress = string(d)
	return Update{Progressed: true}
}

func (e *Engine) ProcExit(err error) Update {
	e.Finished = true
	e.Busy = false
	if err == nil {
		e.Status = "Finished successfully"
	} else {
		e.Status = "Unison finished with error: " + err.Error()
	}
	e.Progress = ""
	e.CanQuit = false
	e.OfferKill = false
	return Update{}
}

func (e *Engine) ProcError(err error) Update {
	if e.Finished {
		return Update{}
	}
	e.Status = err.Error()
	return Update{}
}

func (e *Engine) Quit() Update {
	if e.Finished {
		return Update{}
	}
	if !e.CanQuit {
		return Update{} // FIXME
	}
	e.Status = "Quitting Unison..."
	e.Busy = true
	e.Progress = ""
	e.CanQuit = false
	return Update{Input: []byte("q\n")}
}

func (e *Engine) Interrupt() Update {
	if e.Finished {
		return Update{}
	}
	e.Status = "Interrupting Unison..."
	e.Busy = true
	e.Progress = ""
	e.CanQuit = false
	e.OfferKill = true
	return Update{Interrupt: true}
}

func (e *Engine) Kill() Update {
	if e.Finished {
		return Update{}
	}
	e.Status = "Killing Unison..."
	e.Busy = true
	e.Progress = ""
	e.CanQuit = false
	e.OfferKill = false
	return Update{Kill: true}
}
