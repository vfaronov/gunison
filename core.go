// +build !coremock

package main

import (
	"bytes"
	"log"
	"regexp"
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

	procBuffer func() Update
	ProcExit   func(int, error) Update
	ProcError  func(error) Update
	Diff       func(string) Update
	Sync       func() Update
	Quit       func() Update
	Abort      func() Update
	Interrupt  func() Update
	Kill       func() Update

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
	LeftToRightPartial
	RightToLeft
	RightToLeftPartial
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
	var upd Update
	if c.buf.Len() > 0 && c.procBuffer != nil {
		upd = c.procBuffer()
	}
	if c.buf.Len() > 0 {
		upd = upd.join(c.procBufCommon())
	}
	return upd
}

func (c *Core) ProcStart() Update {
	return c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Starting Unison",

		procBuffer: c.procBufStartup,
		ProcExit:   c.procExitBeforeSync,
		ProcError:  c.procErrorBeforeSync,
		Interrupt:  c.interrupt,
		Kill:       c.kill,
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

func (c *Core) quit() Update {
	return Update{Input: []byte("q\n")}.join(c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Quitting Unison",

		ProcExit:  c.ProcExit,
		Interrupt: c.interrupt,
		Kill:      c.kill,
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
	output := c.buf.Bytes()
	c.buf.Reset()
	status := "Unison exited"
	if code == 0 {
		status = "Finished successfully"
	}
	return echo(output).
		join(echoError(err)).
		join(c.transition(Core{Status: status}))
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

var (
	patEraseLine          = "^\r *\r"
	patPrompt             = "\\s*\\[[^\\]]*\\] $"
	patReallyProceed      = "Do you really want to proceed\\?" + patPrompt
	patPressReturn        = "Press return to continue\\." + patPrompt
	patContactingServer   = "(?m:^Unison [^:\n]+: (Contacting server)\\.\\.\\.$)"
	patConnected          = "(?m:^Connected \\[[^\\]]+\\]$)"
	patLookingForChanges  = "(?m:^(Looking for changes)$)"
	patFileProgress       = "(?m:^[-/|\\\\] ([^\r\n]+))"
	patFileProgressCont   = "^[^\r\n]+"
	patWaitingForChanges  = "(?m:^\\s*(Waiting for changes from server)$)"
	patReconcilingChanges = "(?m:^(Reconciling changes)$)"

	patItem            = patShortTypeStatus + " " + anyOf(parseAction) + " " + patShortTypeStatus + "   (.+)  "
	patShortTypeStatus = "(?:        |deleted |new file|file    |changed |props   |new link|link    |chgd lnk|new dir |dir     |chgd dir|props   )"
	patItemPrompt      = "(?m:^)" + patItem + patPrompt

	patPlanBeginning  = patReplicasHeader + "\n" + patItemPrompt
	patReplicasHeader = "(?m:^(.{12})   (.{12}) +$)"
)

var parseAction = map[string]Action{
	// FIXME: "error"
	"<-?->": Skip,
	"---->": LeftToRight,
	"--?->": LeftToRightPartial,
	"<----": RightToLeft,
	"<-?--": RightToLeftPartial,
	"<-M->": Merge,
}

var expCommon = makeExpecter(true, patReallyProceed, patPressReturn)

func (c *Core) procBufCommon() Update {
	switch pat, _, upd, extra := expCommon(&c.buf); pat {
	case patReallyProceed:
		upd.Alert = Alert{
			Message: Message{extra + "Do you really want to proceed?", Warning},
			Proceed: func() Update { return Update{Input: []byte("y\n")}.join(c.next()) },
			Abort:   c.quit,
		}
		return upd

	case patPressReturn:
		upd.Alert = Alert{
			Message: Message{strings.TrimSpace(extra), Warning},
			Proceed: func() Update { return Update{Input: []byte("\n")}.join(c.next()) },
			Abort:   c.quit,
		}
		return upd

	default:
		return upd
	}
}

var expStartup = makeExpecter(false, patContactingServer, patConnected, patLookingForChanges,
	patFileProgress, patWaitingForChanges, patReconcilingChanges, patPlanBeginning)

func (c *Core) procBufStartup() Update {
	switch pat, m, upd, _ := expStartup(&c.buf); pat {
	case patContactingServer, patLookingForChanges, patWaitingForChanges, patReconcilingChanges:
		c.Status = string(m[1])
		c.Progress = ""
		c.ProgressFraction = 0
		return upd.join(c.next())

	case patFileProgress:
		upd.Progressed = true
		c.Progress = string(m[1])
		c.ProgressFraction = -1
		c.procBuffer = c.procBufFileProgress
		return upd.join(c.next())

	case patPlanBeginning:
		c.Left = strings.TrimSpace(string(m[1]))
		c.Right = strings.TrimSpace(string(m[2]))
		upd.Input = []byte("l\n")
		return upd.join(c.transition(Core{
			Running: true,
			Busy:    true,
			Status:  "Assembling plan",

			ProcExit:  c.procExitBeforeSync,
			ProcError: c.procErrorBeforeSync,
			Interrupt: c.interrupt,
			Kill:      c.kill,
		}))

	default:
		return upd
	}
}

var expFileProgress = makeExpecter(false, patFileProgressCont, patEraseLine)

func (c *Core) procBufFileProgress() Update {
	// We're here when Unison has printed something like "- path/to/file". Because there is
	// no newline or other delimiter, we can't know if "path/to/file" is the entire path or just
	// the chunk that happened to fit into some buffer.
	switch pat, m, upd, _ := expFileProgress(&c.buf); pat {
	case patFileProgressCont: // So, if the line continues, it's more of the same path.
		c.Progress += string(m[0])
		return upd.join(c.next())

	default: // But if there's anything else, we revert to the previous state.
		// (There has to be something else, because procBuffer is only called on a non-empty buffer.)
		c.procBuffer = c.procBufStartup
		return upd.join(c.next())
	}
}

func makeExpecter(raw bool, patterns ...string) func(*bytes.Buffer) (string, [][]byte, Update, string) {
	start := make([]int, len(patterns))
	start[0] = 1
	combined := ""
	for i, pat := range patterns {
		if i > 0 {
			start[i] = start[i-1] + regexp.MustCompile(patterns[i-1]).NumSubexp() + 1
			combined += "|"
		}
		combined += "(" + pat + ")"
	}
	exp := regexp.MustCompile(combined)

	return func(buf *bytes.Buffer) (pattern string, match [][]byte, upd Update, extra string) {
		data := buf.Bytes()
		m := exp.FindSubmatch(data)
		if m == nil {
			return
		}
		offset := bytes.Index(data, m[0])
		buf.Next(offset + len(m[0]))
		if raw {
			extra = string(data[:offset])
		} else {
			upd = echo(data[:offset])
		}
		for i, pat := range patterns {
			if len(m[start[i]]) > 0 {
				pattern = pat
				if i < len(patterns)-1 {
					match = m[start[i]:start[i+1]]
				} else {
					match = m[start[i]:]
				}
				break
			}
		}
		log.Printf("match: %q %q", pattern, match)
		return
	}
}

var (
	expWarning = regexp.MustCompile(`(?i)^warning`)
	expError   = regexp.MustCompile(`(?i)^((?:fatal )?error|can't |failed)`)
)

func echo(output []byte) Update {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return Update{}
	}
	msg := Message{text, Info}
	if expWarning.MatchString(text) {
		msg.Importance = Warning
	} else if expError.MatchString(text) {
		msg.Importance = Error
	}
	return Update{Messages: []Message{msg}}
}
