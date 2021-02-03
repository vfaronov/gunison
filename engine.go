package main

import (
	"bytes"
	"encoding/json"
	"strings"
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

func (e *Engine) RecvOutput(d []byte) Update {
	if e.finished {
		return Update{}
	}
	e.buf.Write(d)
	output := e.buf.String()
	var update Update
	if i := strings.IndexByte(output, '\n'); i >= 0 {
		line := output[:i]
		e.buf.Next(i)
		update.Progressed = true
		e.Current = line
		_ = json.Unmarshal([]byte(line), e)
		_ = json.Unmarshal([]byte(line), &update)
	}
	return update
}

func (e *Engine) RecvExit(err error) Update {
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

func (e *Engine) RecvError(err error) Update {
	if e.finished {
		return Update{}
	}
	e.Status = err.Error()
	return Update{Progressed: true}
}
