// +build !coremock

package main

import (
	"bytes"
	"strings"
	"time"
)

type Core struct {
	Running          bool
	Busy             bool
	Status           string
	Progress         string  // empty string iff not progressing
	ProgressFraction float64 // 0 to 1; or -1 for unknown

	Left, Right string
	Items       []Item
	Plan        map[string]Action

	procBuffer func([]byte) Update
	ProcExit   func(int, error) Update
	ProcError  func(error) Update

	Diff      func(string) Update
	Sync      func() Update
	Quit      func() Update
	Abort     func() Update
	Interrupt func() Update
	Kill      func() Update

	buf bytes.Buffer
}

type Update struct {
	Progressed bool
	PlanReady  bool
	Diff       []byte
	Input      []byte
	Interrupt  bool
	Kill       bool
	Messages   []Message
	Alert      Alert
}

func (upd Update) join(other Update) Update {
	return Update{
		Progressed: upd.Progressed || other.Progressed,
		PlanReady:  upd.PlanReady || other.PlanReady,
		Diff:       append(upd.Diff, other.Diff...), // FIXME
		Input:      append(upd.Input, other.Input...),
		Interrupt:  upd.Interrupt || other.Interrupt,
		Kill:       upd.Kill || other.Kill,
		Messages:   append(upd.Messages, other.Messages...),
		Alert:      other.Alert, // FIXME
	}
}

type Item struct {
	Path        string
	Left, Right Content
	Action      Action
}

type Content struct {
	Type     Type
	Status   Status
	Props    string
	Modified time.Time
	Size     int64
}

type Type byte

const (
	Absent Type = 1 + iota
	File
	Directory
	Symlink
)

type Status byte

const (
	Unchanged Status = 1 + iota
	Created
	Modified
	PropsChanged
	Deleted
)

type Action byte

const (
	Skip Action = 1 + iota
	LeftToRight
	MaybeLeftToRight
	RightToLeft
	MaybeRightToLeft
	Merge
)

type Message struct {
	Text       string
	Importance Importance
}

type Importance byte

const (
	Info Importance = 1 + iota
	Warning
	Error
)

type Alert struct {
	Message
	Proceed func() Update
	Abort   func() Update
}

func NewCore() *Core {
	c := &Core{
		Busy:   true,
		Status: "Starting Unison",
	}
	c.ProcError = c.procStartFailed
	return c
}

func (c *Core) ProcStart() Update {
	*c = Core{
		Running: true,
		Busy:    true,
		Status:  "Starting Unison",

		ProcExit:  c.procExitBeforeSync,
		ProcError: c.procErrorBeforeSync,

		Interrupt: c.interrupt,
		Kill:      c.kill,
	}
	return Update{}
}

func (c *Core) ProcOutput(data []byte) Update {
	_, _ = c.buf.Write(data)
	var upd Update
	prev := c.buf.Len()
	for c.procBuffer != nil {
		upd = upd.join(c.procBuffer(c.buf.Bytes()))
		pos := c.buf.Len()
		if pos == prev { // unable to make any more progress
			break
		}
		prev = pos
	}
	return upd
}

func (c *Core) procStartFailed(err error) Update {
	*c = Core{
		Busy:   false,
		Status: "Failed to start Unison",

		ProcError: echoError,
	}
	return echoError(err)
}

func (c *Core) interrupt() Update {
	*c = Core{
		Running: true,
		Busy:    true,
		Status:  "Interrupting Unison",

		Left:  c.Left,
		Right: c.Right,
		Items: c.Items,
		Plan:  c.Plan,

		ProcExit:  c.ProcExit,
		ProcError: echoError,

		Kill: c.kill,

		buf: c.buf,
	}
	return Update{Interrupt: true}
}

func (c *Core) kill() Update {
	*c = Core{
		Running: true,
		Busy:    true,
		Status:  "Forcing Unison to stop",

		Left:  c.Left,
		Right: c.Right,
		Items: c.Items,
		Plan:  c.Plan,

		ProcExit:  c.ProcExit,
		ProcError: echoError,

		buf: c.buf,
	}
	return Update{Kill: true}
}

func (c *Core) procExitBeforeSync(code int, err error) Update {
	output := c.buf.String()

	*c = Core{
		Left:  c.Left,
		Right: c.Right,
		Items: c.Items,
		Plan:  c.Plan,

		ProcError: echoError,
	}
	if code == 0 {
		c.Status = "Finished successfully"
	} else {
		c.Status = "Unison exited" // TODO: ..." with code %d"?
	}

	upd := Update{}
	if output != "" {
		upd.Messages = []Message{{strings.TrimSpace(output), Info}}
	}
	return upd.join(echoError(err))
}

func (c *Core) procErrorBeforeSync(err error) Update {
	msg := Message{
		Text:       strings.Title(err.Error()) + "\nThis is a fatal error. Unison will be stopped now.",
		Importance: Error,
	}
	return Update{Messages: []Message{msg}}.join(c.interrupt())
}

func echoError(err error) Update {
	if err == nil {
		return Update{}
	}
	return Update{Messages: []Message{
		{err.Error(), Error},
	}}
}
