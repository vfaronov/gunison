// +build !coremock

package main

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
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
		ProcError:  c.procErrorUnrecoverable,
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

func (c *Core) handleExit(code int, err error, codeStatus map[int]string) Update {
	output := c.buf.String()
	c.buf.Reset()
	status := "Unison exited"
	if s, ok := codeStatus[code]; ok {
		status = s
	}
	return echo(output).
		join(echoError(err)).
		join(c.transition(Core{Status: status}))
}

func (c *Core) procExitBeforeSync(code int, err error) Update {
	return c.handleExit(code, err, map[int]string{
		0: "Finished successfully",
	})
}

func (c *Core) procErrorUnrecoverable(err error) Update {
	upd := Update{Messages: []Message{
		{
			Text:       err.Error() + "\nThis is a fatal error. Unison will be stopped now.",
			Importance: Error,
		},
	}}
	return upd.join(c.interrupt())
}

func (c *Core) fatalf(clearBuf bool, format string, args ...interface{}) Update {
	if clearBuf {
		c.buf.Reset()
	}
	return c.procErrorUnrecoverable(fmt.Errorf(format, args...))
}

func echoError(err error) Update {
	var upd Update
	if err != nil {
		upd.Messages = append(upd.Messages, Message{err.Error(), Error})
	}
	return upd
}

const (
	lineBgn = "(?:^|\r|\n)"
	lineEnd = "(?:\r\n?|\n)"
)

func line(pattern string) string {
	return lineBgn + pattern + lineEnd
}

var (
	patSomeLine = line("(.*?)")

	patPrompt             = "\\s*\\[.*\\] $"
	patReallyProceed      = "Do you really want to proceed\\?" + patPrompt
	patPressReturn        = "Press return to continue\\." + patPrompt
	patContactingServer   = line("Unison [^:\n]+: (Contacting server)\\.\\.\\.")
	patPermissionDenied   = line("Permission denied, please try again\\.")
	patConnected          = line("Connected \\[[^\\]]+\\]")
	patLookingForChanges  = line("(Looking for changes)")
	patFileProgress       = lineBgn + "[-/|\\\\] ([^\r\n]+)"
	patFileProgressCont   = "^[^\r\n]+"
	patWaitingForChanges  = line("\\s*(Waiting for changes from server)")
	patReconcilingChanges = line("(Reconciling changes)")

	patPlanBeginning   = lineBgn + "(.{12})   (.{12}) +\r?" + patItemPrompt
	patItemPrompt      = lineBgn + patItem + patPrompt
	patItem            = patShortTypeStatus + " " + anyOf(parseAction) + " " + patShortTypeStatus + "   (.*?)  "
	patShortTypeStatus = "(?:        |deleted |new file|file    |changed |props   |new link|link    |chgd lnk|new dir |dir     |chgd dir|props   )"
	patItemHeader      = line("\\s*" + patItem)
	patItemSideInfo    = " : (?:(absent|deleted)|" + anyOf(parseTypeStatus) + "  (modified on ([0-9-]{10} at [ 0-9:]{8})  size ([0-9]+) .*?))"

	patProceedUpdates             = lineBgn + "Proceed with propagating updates\\?" + patPrompt
	patPropagatingUpdates         = line("(Propagating updates)")
	patStartedFinishedPropagating = line("UNISON [0-9.]+ \\(OCAML [0-9.]+\\) (?:started|finished) propagating changes at .*?")
	patSyncThreadStatus           = line("\\[(?:BGN|END|CONFLICT)\\] .*?")
	patSyncProgress               = lineBgn + "\\s*([0-9]+)%  (?:[0-9]+:[0-9]{2}|--:--) ETA"
	patWhySkipped                 = line("\\s*(?:conflicting updates|skip requested|contents changed on both sides)")
	patShortcut                   = line("Shortcut: .+")
	patSavingState                = line("(Saving synchronizer state)")
)

var parseAction = map[string]Action{
	// FIXME: "error"
	"<-?->": Skip,
	"<=?=>": Skip,
	"---->": LeftToRight,
	"====>": LeftToRight,
	"--?->": LeftToRightPartial,
	"==?=>": LeftToRightPartial,
	"<----": RightToLeft,
	"<====": RightToLeft,
	"<-?--": RightToLeftPartial,
	"<=?==": RightToLeftPartial,
	"<-M->": Merge,
	"<=M=>": Merge,
}

var sendAction = map[Action][]byte{
	Skip:        []byte("/\n"),
	LeftToRight: []byte(">\n"),
	RightToLeft: []byte("<\n"),
	Merge:       []byte("m\n"),

	// Gunison doesn't generate these actions, so if they are in the plan,
	// they are Unison's recommendations and we just accept them.
	LeftToRightPartial: []byte("\n"),
	RightToLeftPartial: []byte("\n"),
}

var parseTypeStatus = map[string]struct {
	Status
	Type
}{
	"unchanged file   ": {Unchanged, File},
	"unchanged symlink": {Unchanged, Symlink},
	"unchanged dir    ": {Unchanged, Directory},
	"new file         ": {Created, File},
	"file             ": {Created, File},
	"changed file     ": {Modified, File},
	"changed props    ": {PropsChanged, File},
	"new symlink      ": {Created, Symlink},
	"symlink          ": {Created, Symlink},
	"changed symlink  ": {Modified, Symlink},
	"new dir          ": {Created, Directory},
	"dir              ": {Created, Directory},
	"changed dir      ": {Modified, Directory},
	"dir props changed": {PropsChanged, Directory},
}

var expCommon = makeExpecter(true, patReallyProceed, patPressReturn)

func (c *Core) procBufCommon() Update {
	switch pat, _, upd, extra := expCommon(&c.buf); pat {
	case patReallyProceed:
		upd.Alert = Alert{
			Message: Message{strings.TrimSpace(extra) + "\n\nDo you really want to proceed?", Warning},
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

var expStartup = makeExpecter(false, patContactingServer, patPermissionDenied, patConnected,
	patLookingForChanges, patFileProgress, patWaitingForChanges, patReconcilingChanges, patPlanBeginning)

func (c *Core) procBufStartup() Update {
	switch pat, m, upd, _ := expStartup(&c.buf); pat {
	case patContactingServer, patLookingForChanges, patWaitingForChanges, patReconcilingChanges:
		c.Status = m[1]
		c.Progress = ""
		c.ProgressFraction = 0
		return upd.join(c.next())

	case patPermissionDenied:
		return upd.join(c.next())

	case patFileProgress:
		upd.Progressed = true
		c.Progress = m[1]
		c.ProgressFraction = -1
		c.procBuffer = c.procBufFileProgress
		return upd.join(c.next())

	case patPlanBeginning:
		c.Left = strings.TrimSpace(m[1])
		c.Right = strings.TrimSpace(m[2])
		upd.Input = []byte("l\n")
		return upd.join(c.transition(Core{
			Running: true,
			Busy:    true,
			Status:  "Assembling plan",

			procBuffer: c.makeProcBufPlan(),
			ProcExit:   c.procExitBeforeSync,
			ProcError:  c.procErrorUnrecoverable,
			Interrupt:  c.interrupt,
			Kill:       c.kill,
		}))

	default:
		return upd
	}
}

var expFileProgress = makeExpecter(false, patFileProgressCont)

func (c *Core) procBufFileProgress() Update {
	// We're here when Unison has printed something like "- path/to/file". Because there is
	// no newline or other delimiter, we can't know if "path/to/file" is the entire path or just
	// the chunk that happened to fit into some buffer.
	switch pat, m, upd, _ := expFileProgress(&c.buf); pat {
	case patFileProgressCont: // So, if the line continues, it's more of the same path.
		c.Progress += m[0]
		return upd.join(c.next())

	default: // But if there's anything else, we revert to the previous state.
		// (There has to be something else, because procBuffer is only called on a non-empty buffer.)
		c.procBuffer = c.procBufStartup
		return upd.join(c.next())
	}
}

func (c *Core) makeProcBufPlan() func() Update {
	items := make([]Item, 0)
	patItemSide := line("(" + regexp.QuoteMeta(c.Left) + "|" + regexp.QuoteMeta(c.Right) + ")\\s*" +
		patItemSideInfo)
	expPlan := makeExpecter(true, patItemHeader, patItemSide, patItemPrompt)

	return func() Update {
		pat, m, upd, extra := expPlan(&c.buf)
		if extra != "" {
			return upd.join(c.fatalf(true, "Cannot parse the following output from Unison:\n%s", extra))
		}

		switch pat {
		case patItemHeader:
			items = append(items, Item{
				Action: parseAction[m[1]],
				Path:   m[2],
			})
			return upd.join(c.next())

		case patItemSide:
			if len(items) == 0 {
				return upd.join(c.fatalf(true, "Got item details before item header"))
			}
			item := &items[len(items)-1]
			sideName := m[1]
			side := &item.Left
			if sideName == c.Right {
				side = &item.Right
			}
			if *side != (Content{}) {
				return upd.join(c.fatalf(true, "Got duplicate details for %s in %s", item.Path, sideName))
			}

			switch m[2] {
			case "absent":
				side.Type = Absent
			case "deleted":
				side.Type = Absent
				side.Status = Deleted
			default:
				ts := parseTypeStatus[m[3]]
				side.Type = ts.Type
				side.Status = ts.Status
			}
			side.Props = m[4]
			side.Modified, _ = time.ParseInLocation("2006-01-02 at 15:04:05", m[5], time.Local)
			side.Size, _ = strconv.ParseInt(m[6], 10, 64)
			return upd.join(c.next())

		case patItemPrompt:
			plan := make(map[string]Action, len(items))
			for _, item := range items {
				plan[item.Path] = item.Action
			}
			return upd.join(c.transition(Core{
				Running: true,
				Status:  "Ready to synchronize",
				Items:   items,
				Plan:    plan,

				ProcExit:  c.procExitBeforeSync,
				ProcError: c.procErrorUnrecoverable,
				Diff:      c.diff,
				Sync:      c.sync,
				Quit:      c.quit,
				Interrupt: c.interrupt,
				Kill:      c.kill,
			}))

		default:
			return upd
		}
	}
}

func (c *Core) diff(string) Update {
	panic("not implemented yet")
}

func (c *Core) sync() Update {
	return Update{Input: []byte("0\n")}.join(c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Starting synchronization",

		procBuffer: c.procBufStartSync,
		ProcExit:   c.procExitBeforeSync,
		ProcError:  c.procErrorUnrecoverable,
		Abort:      c.interrupt,
		Interrupt:  c.interrupt,
		Kill:       c.kill,
	}))
}

var expStartSync = makeExpecter(true, patItemPrompt, patItemHeader, patProceedUpdates)

func (c *Core) procBufStartSync() Update {
	pat, m, upd, extra := expStartSync(&c.buf)
	if extra != "" {
		return upd.join(c.fatalf(false, "Cannot parse the following output from Unison:\n%s", extra))
	}

	switch pat {
	case patItemPrompt:
		path := m[2]
		act, ok := c.Plan[path]
		if !ok {
			return upd.join(c.fatalf(false,
				"Failed to start synchronization because this path is missing from Gunison's plan: %s\n"+
					"This is probably a bug in Gunison.", path))
		}
		upd.Input = sendAction[act]
		return upd.join(c.next())

	case patProceedUpdates:
		upd.Input = []byte("y\n")
		return upd.join(c.transition(Core{
			Running: true,
			Busy:    true,
			Status:  "Starting synchronization",

			procBuffer: c.procBufSync,
			ProcExit:   c.procExitSync,
			ProcError:  echoError,
			Abort:      c.interrupt,
			Interrupt:  c.interrupt,
			Kill:       c.kill,
		}))

	case none:
		return upd

	default:
		return upd.join(c.next())
	}
}

func (c *Core) procExitSync(code int, err error) Update {
	return c.handleExit(code, err, map[int]string{
		// These codes, documented in the Unison manual, take on their meaning
		// only after synchronization begins.
		0: "Finished successfully",
		1: "Finished successfully (some files skipped)",
		2: "Finished with errors",
	})
}

var expSync = makeExpecter(false, patPropagatingUpdates, patStartedFinishedPropagating,
	patSyncThreadStatus, patSyncProgress, patWhySkipped, patShortcut,
	patSavingState, patSomeLine)

func (c *Core) procBufSync() Update {
	switch pat, m, upd, _ := expSync(&c.buf); pat {
	case patPropagatingUpdates, patSavingState:
		c.Status = m[1]
		c.Progress = ""
		c.ProgressFraction = 0
		return upd.join(c.next())

	case patSyncProgress:
		c.Progress = strings.TrimSpace(m[0])
		percent, _ := strconv.Atoi(m[1])
		c.ProgressFraction = float64(percent) / 100
		return upd.join(c.next())

	case patSomeLine: // something we don't explicitly recognize and consume
		// (it's not enough to rely on makeExpecter's echo because
		// at this point we want to echo lines as soon as they come)
		return upd.join(echo(m[1])).join(c.next())

	case none:
		return upd

	default: // all the noise we recognize and ignore, such as patWhySkipped, etc.
		return upd.join(c.next())
	}
}

func makeExpecter(raw bool, patterns ...string) func(*bytes.Buffer) (string, []string, Update, string) {
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

	return func(buf *bytes.Buffer) (pattern string, match []string, upd Update, extra string) {
		data := buf.String()
		m := exp.FindStringSubmatch(data)
		if m == nil {
			return
		}
		offset := strings.Index(data, m[0])
		buf.Next(offset + len(m[0]))
		if raw {
			extra = strings.TrimSpace(data[:offset])
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
		log.Printf("match: %q", match)
		return
	}
}

const none = ""

var (
	expWarning = regexp.MustCompile(`(?i)^(?:warning|synchronization incomplete)`)
	expError   = regexp.MustCompile(`(?i)^((?:fatal )?error|can't |failed)`)
)

func echo(output string) Update {
	text := strings.TrimSpace(output)
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
