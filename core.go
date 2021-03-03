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

	procBuffered func([]byte) (int, Update)
	ProcExit     func(int, error) Update
	ProcError    func(error) Update
	Diff         func(string) Update
	Sync         func() Update
	Quit         func() Update
	Abort        func() Update
	Interrupt    func() Update
	Kill         func() Update

	buf bytes.Buffer
}

func (c *Core) transition(newc Core) Update {
	// Some pieces of state have to be preserved in all transitions.
	// For example, even after Unison exits and there's nothing more to do, the UI is
	// still displaying the tree, for which it still needs c.Items and c.Plan.
	if newc.Left == "" {
		newc.Left = c.Left
	}
	if newc.Right == "" {
		newc.Right = c.Right
	}
	if newc.Items == nil {
		newc.Items = c.Items
	}
	if newc.Plan == nil {
		newc.Plan = c.Plan
	}
	newc.buf = c.buf

	if newc.ProcError == nil {
		newc.ProcError = echoError
	}

	*c = newc
	return c.next()
}

type Update struct {
	Progressed bool
	Diff       []byte
	Input      []byte
	Interrupt  bool
	Kill       bool
	Messages   []Message
	Alert      Alert
}

func (upd Update) join(other Update) Update {
	upd = Update{
		Progressed: upd.Progressed || other.Progressed,
		Diff:       upd.Diff,
		Input:      append(upd.Input, other.Input...),
		Interrupt:  upd.Interrupt || other.Interrupt,
		Kill:       upd.Kill || other.Kill,
		Messages:   append(upd.Messages, other.Messages...),
		Alert:      upd.Alert,
	}
	if other.Diff != nil {
		upd.Diff = other.Diff
	}
	if other.Alert.Text != "" {
		if upd.Alert.Text != "" {
			panic("cannot join two Updates with non-zero Alert")
		}
		upd.Alert = other.Alert
	}
	return upd
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
	c.ProcError = c.procErrorBeforeStart
	return c
}

func (c *Core) next() Update {
	if data := c.buf.Bytes(); len(data) > 0 && c.procBuffered != nil {
		if n, upd := c.procBuffered(data); n > 0 {
			c.buf.Next(n)
			return upd.join(c.next())
		}
	}
	return Update{}
}

func (c *Core) ProcStart() Update {
	return c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Starting Unison",

		ProcExit:  c.procExitBeforeSync,
		ProcError: c.procErrorBeforeSync,
		Interrupt: c.interrupt,
		Kill:      c.kill,
	})
}

func (c *Core) ProcOutput(data []byte) Update {
	_, _ = c.buf.Write(data)
	return c.next()
}

func (c *Core) procErrorBeforeStart(err error) Update {
	return echoError(err).join(c.transition(Core{
		Status: "Failed to start Unison",
	}))
}

func (c *Core) interrupt() Update {
	return Update{Interrupt: true}.join(c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Interrupting Unison",

		ProcExit: c.ProcExit,
		Kill:     c.kill,
	}))
}

func (c *Core) kill() Update {
	return Update{Kill: true}.join(c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Killing Unison",

		ProcExit: c.ProcExit,
	}))
}

func (c *Core) procExitBeforeSync(code int, err error) Update {
	output := strings.TrimSpace(c.buf.String())
	c.buf.Reset()

	status := "Unison exited"
	if code == 0 {
		status = "Finished successfully"
	}

	var upd Update
	if output != "" {
		upd.Messages = append(upd.Messages, Message{output, Info})
	}

	return upd.join(echoError(err)).join(c.transition(Core{
		Status: status,
	}))
}

func (c *Core) procErrorBeforeSync(err error) Update {
	upd := Update{Messages: []Message{
		{
			Text:       err.Error() + "\nThis is a fatal error. Unison will be stopped now.",
			Importance: Error,
		},
	}}
	return upd.join(c.interrupt())
}

func echoError(err error) Update {
	var upd Update
	if err != nil {
		upd.Messages = append(upd.Messages, Message{err.Error(), Error})
	}
	return upd
}
