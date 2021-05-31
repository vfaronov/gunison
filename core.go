package main

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// Core parses output from Unison, maintains a model of Unison's current state,
// accepts commands from the user, and produces input to drive Unison to the desired state.
// Core does not perform I/O itself. Think "sans I/O" protocol implementations,
// or the "functional core, imperative shell" pattern (but not actually functional).
// The main "shell" performs all I/O with Unison and the user, feeds events into Core by calling
// the respective methods/functions, and updates the UI according to the returned Update
// and the current values of the Core's fields.
type Core struct {
	// These fields describe the general state of the Core and the underlying Unison process.
	Running          bool    // true iff Unison is running
	Busy             bool    // true iff the core is waiting for events from Unison
	Status           string  // description of what's currently going on
	Progress         string  // description of the current operation's progress; empty string iff unknown
	ProgressFraction float64 // quantitative representation of Progress: 0 to 1, or -1 for "no estimate"

	// These fields become non-zero once the corresponding information is parsed from Unison.
	Left, Right string // names of replicas
	Items       []Item // items to synchronize - updated by the UI to set the desired Action

	// These functions must be called when the user requests the corresponding action via the UI.
	// Any of these fields may be nil, which means the action is impossible and must not be offered
	// to the user.
	Diff      func(string) Update // load differences for the item with the given Path
	Sync      func() Update       // start synchronization according to the Action of each of Items
	Quit      func() Update       // quit Unison gracefully
	Abort     func() Update       // abort current operation - often (but not always) same as Interrupt
	Interrupt func() Update       // interrupt the Unison process
	Kill      func() Update       // kill the Unison process

	buf        bytes.Buffer
	procBuffer func() Update
	exitCodes  map[int]string
	procError  func(error) Update
	seek       string
}

// Core is a kind of a state machine, but it doesn't have a discrete "state" field.
// Its state is the combination of all its fields. The most important are the various functions
// returning Update, which specify how to handle incoming events and transition to new states.

// transition replaces the state of c with that of newc, automatically maintaining pieces of state
// that must be preserved across all transitions. For example, even after Unison exits and there's
// nothing more to do, the UI is still displaying the tree, for which it still needs c.Items.
func (c *Core) transition(newc Core) Update {
	if newc.Left == "" {
		newc.Left = c.Left
	}
	if newc.Right == "" {
		newc.Right = c.Right
	}
	if newc.Items == nil {
		newc.Items = c.Items
	}
	if newc.exitCodes == nil {
		newc.exitCodes = c.exitCodes
	}
	newc.buf = c.buf
	*c = newc
	return c.next()
}

// Update describes the I/O that must be performed as a result of some event,
// before calling any further Update-returning methods/functions on Core.
type Update struct {
	Progressed bool      // if true, the user is to be informed that progress has been made
	Diff       []byte    // the diff (previously requested by calling Core.Diff) to be shown to the user
	Input      []byte    // to be written to Unison's stdin
	Interrupt  bool      // if true, the Unison process is to be interrupted
	Kill       bool      // if true, the Unison process is to be killed
	Messages   []Message // to be shown to the user
	Alert      Alert     // to be shown to the user if non-zero
}

// join returns an Update that is equivalent to first performing upd and then performing other.
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
		upd.Diff = other.Diff // should never both be non-nil
	}
	if other.Alert.Text != "" {
		if upd.Alert.Text != "" {
			panic("cannot join two Updates with non-zero Alert")
		}
		upd.Alert = other.Alert
	}
	return upd
}

// An Item is what will be synchronized by Unison.
type Item struct {
	Path           string
	Left, Right    Content
	Override       Action // set explicitly by the user (if any)
	Recommendation Action // original from Unison
}

func (it Item) Action() Action {
	if it.Override != NoAction {
		return it.Override
	}
	return it.Recommendation
}

func (it Item) IsOverridden() bool {
	return it.Override != NoAction
}

// Content describes an Item in one of the replicas.
type Content struct {
	Type   Type
	Status Status // zero if unknown (only when Type == Absent)
	Props  string // human-readable description of properties
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
	NoAction Action = iota
	LeftToRight
	LeftToRightPartial
	RightToLeft
	RightToLeftPartial
	Merge
	Skip
	Mixed
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

// Alert describes a situation that requires a "proceed/abort" decision by the user.
// Exactly one of Proceed or Abort must be called before continuing normal calls to Core's
// methods/functions.
type Alert struct {
	Message
	Proceed func() Update
	Abort   func() Update
}

// NewCore returns a Core in its initial state, when Unison is about to be started.
func NewCore() *Core {
	c := &Core{
		Busy:   true,
		Status: "Starting Unison",
	}
	c.procError = c.procErrorBeforeStart
	return c
}

// next processes any events implicitly caused by state changes within the Core. For example,
// when procBuffer has changed, the new one might be able to parse more of the buffer's current contents.
// As a rule of thumb, whenever c has made some progress and wants to return upd, it must
// return upd.join(c.next()) instead.
func (c *Core) next() Update {
	var upd Update
	if c.buf.Len() > 0 && c.procBuffer != nil {
		upd = c.procBuffer()
	}
	if c.buf.Len() > 0 {
		upd = upd.join(c.procBufferCommon())
	}
	return upd
}

// ProcStart must be called when the Unison process is started.
func (c *Core) ProcStart() Update {
	return c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Starting Unison",

		procBuffer: c.procBufferStartup,
		procError:  c.procErrorUnrecoverable,

		Interrupt: c.interrupt,
		Kill:      c.kill,
	})
}

// ProcOutput must be called when data is received from Unison's stdout or stderr.
func (c *Core) ProcOutput(data []byte) Update {
	_, _ = c.buf.Write(data)
	return c.next()
}

func (c *Core) procErrorBeforeStart(err error) Update {
	return echoError(err).join(c.transition(Core{
		Status: "Failed to start Unison",
	}))
}

// ProcExit must be called when the Unison process exits with the given code
// and error condition as reported by os/exec.(*Cmd).Wait.
func (c *Core) ProcExit(code int, err error) Update {
	output := c.buf.String()
	c.buf.Reset()
	status := "Unison exited"
	if code == 0 {
		status = "Finished successfully"
	}
	if s, ok := c.exitCodes[code]; ok {
		status = s
	}
	return echo(output, Info).
		join(echoError(err)).
		join(c.transition(Core{Status: status}))
}

// ProcError must be called when an I/O error happens with Unison.
func (c *Core) ProcError(err error) Update {
	if c.procError == nil {
		return echoError(err)
	}
	return c.procError(err)
}

func (c *Core) quit() Update {
	return Update{Input: []byte("q\n")}.join(c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Quitting Unison",

		Interrupt: c.interrupt,
		Kill:      c.kill,
	}))
}

func (c *Core) interrupt() Update {
	return Update{Interrupt: true}.join(c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Interrupting Unison",

		Kill: c.kill,
	}))
}

func (c *Core) kill() Update {
	return Update{Kill: true}.join(c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Killing Unison",
	}))
}

func (c *Core) procErrorUnrecoverable(err error) Update {
	upd := Update{Messages: []Message{
		{
			Text:       err.Error() + "\nThis is probably a bug in Gunison. Unison will be stopped now.",
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

	patPrompt             = " *\\[.*\\] $"
	patReallyProceed      = "Do you really want to proceed\\?" + patPrompt
	patPressReturn        = "Press return to continue\\." + patPrompt
	patContactingServer   = line("Unison [^:\n]+: (Contacting server)\\.\\.\\.")
	patPermissionDenied   = line("Permission denied, please try again\\.")
	patConnected          = line("Connected \\[[^\\]]+\\]")
	patLookingForChanges  = line("(Looking for changes)")
	patFileProgress       = lineBgn + "[-/|\\\\] ([^\r\n]+)"
	patFileProgressCont   = "^[^\r\n]+"
	patWaitingForChanges  = line(" *(Waiting for changes from server)")
	patReconcilingChanges = line("(Reconciling changes)")

	patPlanBeginning   = lineBgn + "(.{12})   (.{12}) +\r?" + patItemPrompt
	patItemPrompt      = lineBgn + patItem + patPrompt
	patItem            = " *" + patShortTypeStatus + " " + AnyOf(parseAction) + " " + patShortTypeStatus + "   (.*?)  "
	patShortTypeStatus = "(?:        |deleted |new file|file    |changed |props   |new link|link    |chgd lnk|new dir |dir     |chgd dir|props   )"
	patItemHeader      = line(patItem)
	patItemSideInfo    = " : (?:(absent|deleted)|" + AnyOf(parseTypeStatus) + "  (.*?))"

	// Unison prefixes diff output with a blank line, the command line, and two more blank lines.
	patDiffHeader = lineBgn + "\r?\n.+?\r?\n\r?\n"

	patProceedUpdates             = lineBgn + "Proceed with propagating updates\\?" + patPrompt
	patPropagatingUpdates         = line("(Propagating updates)")
	patStartedFinishedPropagating = line("(UNISON|Unison) [0-9.]+ \\((OCAML|ocaml) [0-9.]+\\) (?:started|finished) propagating changes at .*?")
	patSyncThreadStatus           = line("\\[(?:BGN|END|CONFLICT)\\] .*?")
	patSyncProgress               = lineBgn + " *([0-9]+)%  (?:[0-9]+:[0-9]{2}|--:--) ETA"
	patMergeNoise                 = line("(?:Merge command: .*?" +
		"|Merge result \\(exited \\(0\\)\\):\n.*?" +
		"|(?:No|One|Two|Three) outputs? detected *" +
		"|Two outputs not equal but merge command returned 0.*?" +
		"|No output from merge cmd and both original files are still present" +
		"|Merge program (?:made files equal|changed just (?:first|second) input)" +
		"|Merge program changed both of its inputs in different ways, but returned zero\\." +
		"|No outputs and (?:first|second) replica has been deleted *)")
	patWhySkipped  = line(" *(?:conflicting updates|skip requested|(?:contents|properties) changed on both sides)")
	patShortcut    = line("Shortcut: .+")
	patSavingState = line("(Saving synchronizer state)")
)

var parseAction = map[string]Action{
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
	// TODO: "error"
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

var expCommon = makeExpecter(true, &patReallyProceed, &patPressReturn)

func (c *Core) procBufferCommon() Update {
	switch pat, _, upd, extra := expCommon(&c.buf); pat {
	case &patReallyProceed:
		upd.Alert = Alert{
			Message: Message{strings.TrimSpace(extra) + "\n\nDo you really want to proceed?", Warning},
			Proceed: func() Update { return Update{Input: []byte("y\n")}.join(c.next()) },
			Abort:   c.quit,
		}
		return upd

	case &patPressReturn:
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

var expStartup = makeExpecter(false, &patContactingServer, &patPermissionDenied, &patConnected,
	&patLookingForChanges, &patFileProgress, &patWaitingForChanges, &patReconcilingChanges,
	&patPlanBeginning)

func (c *Core) procBufferStartup() Update {
	switch pat, m, upd, _ := expStartup(&c.buf); pat {
	case &patContactingServer, &patLookingForChanges, &patWaitingForChanges, &patReconcilingChanges:
		c.Status = m[1]
		c.Progress = ""
		c.ProgressFraction = 0
		return upd.join(c.next())

	case &patPermissionDenied:
		return upd.join(c.next())

	case &patFileProgress:
		upd.Progressed = true
		c.Progress = m[1]
		c.ProgressFraction = -1
		c.procBuffer = c.procBufferFileProgress
		return upd.join(c.next())

	case &patPlanBeginning:
		c.Left = strings.TrimSpace(m[1])
		c.Right = strings.TrimSpace(m[2])
		upd.Input = []byte("l\n")
		return upd.join(c.transition(Core{
			Running: true,
			Busy:    true,
			Status:  "Assembling plan",

			procBuffer: c.makeProcBufferPlan(),
			procError:  c.procErrorUnrecoverable,

			Interrupt: c.interrupt,
			Kill:      c.kill,
		}))

	default:
		return upd
	}
}

var expFileProgress = makeExpecter(false, &patFileProgressCont)

func (c *Core) procBufferFileProgress() Update {
	// We're here when Unison has printed something like "- path/to/file". Because there is
	// no newline or other delimiter, we can't know if "path/to/file" is the entire path or just
	// the chunk that happened to fit into some buffer.
	switch pat, m, upd, _ := expFileProgress(&c.buf); pat {
	case &patFileProgressCont: // So, if the line continues, it's more of the same path.
		c.Progress += m[0]
		return upd.join(c.next())

	default: // But if there's anything else, we revert to the previous state.
		// (There has to be something else, because procBuffer is only called on a non-empty buffer.)
		c.procBuffer = c.procBufferStartup
		return upd.join(c.next())
	}
}

func (c *Core) makeProcBufferPlan() func() Update {
	items := make([]Item, 0)
	patItemSide := line("(" + regexp.QuoteMeta(c.Left) + "|" + regexp.QuoteMeta(c.Right) + ") *" +
		patItemSideInfo)
	expPlan := makeExpecter(true, &patItemHeader, &patItemSide, &patItemPrompt)

	return func() Update {
		pat, m, upd, extra := expPlan(&c.buf)
		extra = strings.TrimSpace(extra)
		if extra != "" {
			return upd.join(c.fatalf(true, "Cannot parse the following output from Unison:\n%s", extra))
		}

		switch pat {
		case &patItemHeader:
			items = append(items, Item{
				Recommendation: parseAction[m[1]],
				Path:           m[2],
			})
			return upd.join(c.next())

		case &patItemSide:
			if len(items) == 0 {
				return upd.join(c.fatalf(true, "Got item details before item header."))
			}
			item := &items[len(items)-1]
			sideName := m[1]
			side := &item.Left
			if sideName == c.Right {
				side = &item.Right
			}
			if *side != (Content{}) {
				return upd.join(c.fatalf(true,
					"Got duplicate details for '%s' in %s.", item.Path, sideName))
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
				side.Props = m[4]
			}
			return upd.join(c.next())

		case &patItemPrompt:
			c.Items = items
			return upd.join(c.transitionToReady())

		default:
			return upd
		}
	}
}

func (c *Core) transitionToReady() Update {
	return c.transition(Core{
		Running: true,
		Status:  "Ready to synchronize",

		procError: c.procErrorUnrecoverable,

		Diff:      c.diff,
		Sync:      c.sync,
		Quit:      c.quit,
		Interrupt: c.interrupt,
		Kill:      c.kill,
	})
}

func (c *Core) restorePrompt() Update {
	return c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Waiting for Unison",

		procBuffer: c.procBufferRestorePrompt,
		procError:  c.procErrorUnrecoverable,

		Interrupt: c.interrupt,
		Kill:      c.kill,
	})
}

var expSeek = makeExpecter(false, &patItemPrompt, &patProceedUpdates)

func (c *Core) procBufferRestorePrompt() Update {
	switch pat, _, upd, _ := expSeek(&c.buf); pat {
	case &patItemPrompt, &patProceedUpdates:
		return upd.join(c.transitionToReady())

	default:
		return upd
	}
}

func (c *Core) diff(path string) Update {
	return Update{Input: []byte("0\n")}.join(c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Requesting diff",

		procBuffer: c.procBufferDiffSeek,
		procError:  c.procErrorUnrecoverable,

		Abort:     c.restorePrompt,
		Interrupt: c.interrupt,
		Kill:      c.kill,

		seek: path,
	}))
}

func (c *Core) procBufferDiffSeek() Update {
	switch pat, m, upd, _ := expSeek(&c.buf); pat {
	case &patItemPrompt:
		if m[2] == c.seek { // found the path to diff
			upd.Input = []byte("d\n")
			c.procBuffer = c.procBufferDiffBegin
			c.Abort = c.interrupt
		} else {
			upd.Input = []byte("n\n")
		}
		return upd.join(c.next())

	case &patProceedUpdates: // there's no next item to seek to
		// This is fatal in the sense that we screwed up so badly, we better not try to continue.
		return upd.join(c.fatalf(false, "Failed to find '%s' in Unison prompts.", c.seek))

	default:
		return upd
	}
}

var expDiffBegin = makeExpecter(true, &patShortcut, &patDiffHeader, &patItemPrompt)

func (c *Core) procBufferDiffBegin() Update {
	switch pat, _, upd, extra := expDiffBegin(&c.buf); pat {
	case &patDiffHeader:
		c.procBuffer = c.procBufferDiffOutput
		return upd.
			join(echo(extra, Warning)). // diff's stderr (if any) gets printed before the "header"
			join(c.next())

	case &patItemPrompt:
		return upd.
			join(echo(extra, Error)).
			join(c.transitionToReady())

	default:
		return upd.join(echo(extra, Info))
	}
}

var expDiffOutput = makeExpecter(true, &patItemPrompt)

func (c *Core) procBufferDiffOutput() Update {
	switch pat, _, upd, out := expDiffOutput(&c.buf); pat {
	case &patItemPrompt:
		if strings.TrimSpace(out) != "" {
			upd.Diff = []byte(out)
		}
		return upd.join(c.transitionToReady())

	default:
		return upd
	}
}

func (c *Core) sync() Update {
	return Update{Input: []byte("0\n")}.join(c.transition(Core{
		Running: true,
		Busy:    true,
		Status:  "Starting synchronization",

		procBuffer: c.makeProcBufferStartSync(),
		procError:  c.procErrorUnrecoverable,

		Abort:     c.interrupt,
		Interrupt: c.interrupt,
		Kill:      c.kill,
	}))
}

var expStartSync = makeExpecter(true, &patItemPrompt, &patItemHeader, &patProceedUpdates)

func (c *Core) makeProcBufferStartSync() func() Update {
	plan := make(map[string]Action, len(c.Items))
	for _, item := range c.Items {
		plan[item.Path] = item.Action()
	}

	return func() Update {
		pat, m, upd, extra := expStartSync(&c.buf)
		extra = strings.TrimSpace(extra)
		if extra != "" {
			// Any unexpected output at this crucial phase is too risky to ignore (echo).
			return upd.join(c.fatalf(false, "Cannot parse the following output from Unison:\n%s", extra))
		}

		switch pat {
		case &patItemPrompt:
			path := m[2]
			act, ok := plan[path]
			if !ok {
				return upd.join(c.fatalf(false,
					"Failed to start synchronization because this path is missing from Gunison's plan: %s",
					path))
			}
			upd.Input = sendAction[act]
			return upd.join(c.next())

		case &patItemHeader:
			return upd.join(c.next())

		case &patProceedUpdates:
			upd.Input = []byte("y\n")
			return upd.join(c.transition(Core{
				Running: true,
				Busy:    true,
				Status:  "Starting synchronization",

				procBuffer: c.procBufferSync,
				exitCodes: map[int]string{
					// These codes, documented in the Unison manual, actually take on their meaning
					// only after synchronization begins.
					0: "Finished successfully",
					1: "Finished successfully (some files skipped)",
					2: "Finished with errors",
				},

				Abort:     c.interrupt,
				Interrupt: c.interrupt,
				Kill:      c.kill,
			}))

		default:
			return upd
		}
	}
}

var expSync = makeExpecter(false, &patPropagatingUpdates, &patStartedFinishedPropagating,
	&patSyncThreadStatus, &patSyncProgress, &patMergeNoise, &patWhySkipped, &patShortcut,
	&patSavingState, &patSomeLine)

func (c *Core) procBufferSync() Update {
	switch pat, m, upd, _ := expSync(&c.buf); pat {
	case &patPropagatingUpdates, &patSavingState:
		c.Status = m[1]
		c.Progress = ""
		c.ProgressFraction = 0
		return upd.join(c.next())

	case &patSyncProgress:
		c.Progress = strings.TrimSpace(m[0])
		percent, _ := strconv.Atoi(m[1])
		c.ProgressFraction = float64(percent) / 100
		return upd.join(c.next())

	case &patSomeLine: // something we don't explicitly recognize and consume
		// (it's not enough to rely on makeExpecter's echo because
		// at this point we want to echo lines as soon as they come)
		return upd.
			join(echo(m[1], Info)).
			join(c.next())

	case nil:
		return upd

	default: // all the noise we recognize and ignore, such as patWhySkipped, etc.
		return upd.join(c.next())
	}
}

// makeExpecter creates a function to match the buffer's current contents according to regexp patterns.
// That function returns all zeroes if none of the patterns match. Otherwise, it consumes from the buffer
// and returns the pattern that matched (exactly one of the given patterns) and its submatches.
//
// If multiple patterns match, the one that matches earlier in the buffer (not in the argument list)
// is used.
//
// Any text before the match is returned as the extra string. Additionally,
// if raw is false, it is echoed in the returned Update's Messages.
func makeExpecter(raw bool, patterns ...*string,
) func(*bytes.Buffer) (pattern *string, sub []string, upd Update, extra string) {
	start := make([]int, len(patterns))
	start[0] = 2
	combined := ""
	for i, pat := range patterns {
		if i > 0 {
			start[i] = start[i-1] + 2*(1+regexp.MustCompile(*patterns[i-1]).NumSubexp())
			combined += "|"
		}
		combined += "(" + *pat + ")"
	}
	exp := regexp.MustCompile(combined)

	return func(buf *bytes.Buffer) (pattern *string, sub []string, upd Update, extra string) {
		data := buf.Bytes() // not String: we'll only take small slices, shouldn't keep it all in memory
		m := exp.FindSubmatchIndex(data)
		if m == nil {
			return
		}
		buf.Next(m[1])
		extra = string(data[:m[0]])
		if !raw {
			upd = echo(extra, Info)
		}
		for i, pat := range patterns {
			if m[start[i]] != -1 {
				pattern = pat
				if i < len(patterns)-1 {
					sub = SliceString(data, m[start[i]:start[i+1]])
				} else {
					sub = SliceString(data, m[start[i]:])
				}
				break
			}
		}
		log.Printf("match: %q", sub)
		return
	}
}

var (
	expWarning = regexp.MustCompile(`(?i)^(?:warning|synchronization incomplete|merge result)`)
	expError   = regexp.MustCompile(`(?i)^((?:fatal )?error|can't |cannot |failed|uncaught|invalid|bad )`)
)

func echo(output string, minImportance Importance) Update {
	text := strings.TrimSpace(output)
	if text == "" {
		return Update{}
	}
	msg := Message{text, minImportance}
	switch {
	case msg.Importance < Error && expError.MatchString(text):
		msg.Importance = Error
	case msg.Importance < Warning && expWarning.MatchString(text):
		msg.Importance = Warning
	}
	return Update{Messages: []Message{msg}}
}
