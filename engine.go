package main

import (
	"bytes"
)

type Engine struct {
	Finished bool
	Busy     bool
	Status   string
	Progress float64 // fraction of 1, or -1 for "not applicable"
	Current  string

	buf bytes.Buffer
}

type Update struct {
	Progressed bool
	Input      []byte
}

func NewEngine() *Engine {
	return &Engine{
		Busy:     true,
		Status:   "Starting Unison...",
		Progress: -1,
	}
}

func (e *Engine) ProcOutput(d []byte) Update {
	if e.Finished {
		return Update{}
	}
	e.buf.Write(d)
	e.Current = string(d)
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
	e.Progress = -1
	return Update{Progressed: true}
}

func (e *Engine) ProcError(err error) Update {
	if e.Finished {
		return Update{}
	}
	e.Status = err.Error()
	return Update{Progressed: true}
}
