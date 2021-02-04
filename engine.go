package main

import (
	"bytes"
)

type Engine struct {
	Status   string
	Current  string
	Working  bool
	Progress float64 // fraction of 1, or -1 for "not applicable"

	buf      bytes.Buffer
	finished bool
}

type Update struct {
	Progressed bool
	Send       []byte
}

func NewEngine() *Engine {
	return &Engine{
		Status:   "Starting up",
		Working:  true,
		Progress: -1,
	}
}

func (e *Engine) ProcOutput(d []byte) Update {
	if e.finished {
		return Update{}
	}
	e.buf.Write(d)
	e.Current = string(d)
	return Update{Progressed: true}
}

func (e *Engine) ProcExit(err error) Update {
	if e.finished {
		return Update{}
	}
	if err == nil {
		e.Status = "Completed"
	} else {
		e.Status = err.Error()
	}
	e.Working = false
	e.Progress = -1
	e.finished = true
	return Update{Progressed: true}
}

func (e *Engine) ProcError(err error) Update {
	if e.finished {
		return Update{}
	}
	e.Status = err.Error()
	return Update{Progressed: true}
}
