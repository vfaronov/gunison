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
	Left, Right      string

	Quit      func() Update
	Interrupt func() Update
	Kill      func() Update

	buf bytes.Buffer
}

type Update struct {
	Progressed bool
	Input      []byte
	Interrupt  bool
	Kill       bool
}

func NewEngine() *Engine {
	e := &Engine{
		Busy:             true,
		Status:           "Starting Unison...",
		ProgressFraction: -1,
	}
	e.Quit = e.doQuit
	e.Interrupt = e.doInterrupt
	e.Kill = e.doKill
	return e
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
	e.Quit = nil
	e.Interrupt = nil
	e.Kill = nil
	return Update{}
}

func (e *Engine) ProcError(err error) Update {
	if e.Finished {
		return Update{}
	}
	e.Status = err.Error()
	return Update{}
}

func (e *Engine) doQuit() Update {
	e.Status = "Quitting Unison..."
	e.Busy = true
	e.Progress = ""
	e.Quit = nil
	return Update{Input: []byte("q\n")}
}

func (e *Engine) doInterrupt() Update {
	e.Status = "Interrupting Unison..."
	e.Busy = true
	e.Progress = ""
	e.Quit = nil
	e.Interrupt = nil
	return Update{Interrupt: true}
}

func (e *Engine) doKill() Update {
	e.Status = "Killing Unison..."
	e.Busy = true
	e.Progress = ""
	e.Quit = nil
	e.Interrupt = nil
	e.Kill = nil
	return Update{Kill: true}
}
