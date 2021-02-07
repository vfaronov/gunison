package main

import (
	"bytes"
)

type Core struct {
	Finished         bool
	Busy             bool
	Status           string
	Progress         string  // empty string iff not progressing
	ProgressFraction float64 // 0 to 1; or -1 for unknown

	Left, Right string

	Sync      func(Plan) Update
	Quit      func() Update
	Abort     func() Update
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

type Plan interface{}

func NewCore() *Core {
	e := &Core{
		Busy:             true,
		Status:           "Starting Unison...",
		ProgressFraction: -1,
	}
	e.Quit = e.doQuit
	e.Interrupt = e.doInterrupt
	e.Kill = e.doKill
	return e
}

func (c *Core) ProcOutput(d []byte) Update {
	if c.Finished {
		return Update{}
	}
	c.buf.Write(d)
	c.Progress = string(d)
	return Update{Progressed: true}
}

func (c *Core) ProcExit(err error) Update {
	c.Finished = true
	c.Busy = false
	if err == nil {
		c.Status = "Finished successfully"
	} else {
		c.Status = "Unison finished with error: " + err.Error()
	}
	c.Progress = ""
	c.Quit = nil
	c.Interrupt = nil
	c.Kill = nil
	return Update{}
}

func (c *Core) ProcError(err error) Update {
	if c.Finished {
		return Update{}
	}
	c.Status = err.Error()
	return Update{}
}

func (c *Core) doSync(Plan) Update {
	c.Busy = true
	c.Status = "Synchronizing..."
	c.Progress = ""
	c.ProgressFraction = -1
	c.Sync = nil
	c.Quit = nil
	c.Abort = c.Interrupt
	return Update{}
}

func (c *Core) doQuit() Update {
	c.Status = "Quitting Unison..."
	c.Busy = true
	c.Progress = ""
	c.Quit = nil
	c.Abort = nil
	return Update{Input: []byte("q\n")}
}

func (c *Core) doInterrupt() Update {
	c.Status = "Interrupting Unison..."
	c.Busy = true
	c.Progress = ""
	c.Quit = nil
	c.Abort = nil
	c.Interrupt = nil
	return Update{Interrupt: true}
}

func (c *Core) doKill() Update {
	c.Status = "Killing Unison..."
	c.Busy = true
	c.Progress = ""
	c.Quit = nil
	c.Abort = nil
	c.Interrupt = nil
	c.Kill = nil
	return Update{Kill: true}
}
