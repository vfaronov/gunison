// +build !coremock

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinimal(t *testing.T) {
	c := NewCore()
	assertEqual(t, c.Status, "Starting Unison")
	assert.False(t, c.Running)
	assert.True(t, c.Busy)
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)
	assert.Empty(t, c.Left)
	assert.Empty(t, c.Right)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Quit)
	assert.Nil(t, c.Abort)
	assert.NotNil(t, c.Interrupt)
	assert.NotNil(t, c.Kill)

	assert.Zero(t, c.ProcStart())
	assert.True(t, c.Running)
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assertEqual(t, c.Status, "Contacting server")
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))
	assertEqual(t, c.Status, "Looking for changes")
	assert.Zero(t, c.ProcOutput([]byte("Reconciling changes\n")))
	assertEqual(t, c.Status, "Reconciling changes")
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l")})
	assertEqual(t, c.Status, "Assembling plan")
	assertEqual(t, c.Left, "left")
	assertEqual(t, c.Right, "right")
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Status, "Ready to synchronize")
	assert.False(t, c.Busy)
	assertEqual(t, c.Items, []Item{
		{
			Path:   "one",
			Left:   Content{File, Modified, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--"},
			Right:  Content{File, Unchanged, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--"},
			Action: LeftToRight,
		},
	})
	require.NotNil(t, c.Sync)
	assert.NotNil(t, c.Quit)

	assertEqual(t, c.Sync(),
		Update{Input: []byte("0")})
	assertEqual(t, c.Status, "Starting synchronization")
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Quit)
	assert.NotNil(t, c.Abort)
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte(">")})
	assert.Zero(t, c.ProcOutput([]byte(">\r")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y")})
	assert.Zero(t, c.ProcOutput([]byte("y\n")))

	assert.Zero(t, c.ProcOutput([]byte("Propagating updates\n")))
	assertEqual(t, c.Status, "Propagating updates")
	assert.Zero(t, c.ProcOutput([]byte("\n\nUNISON 2.51.3 (OCAML 4.11.1) started propagating changes at 18:31:20.92 on 08 Feb 2021\n")))
	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assertEqual(t, c.ProcOutput([]byte("100%  00:00 ETA")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "100%  00:00 ETA")
	assertEqual(t, c.ProgressFraction, 1.00)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file one\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("UNISON 2.51.3 (OCAML 4.11.1) finished propagating changes at 18:31:20.92 on 08 Feb 2021\n\n\n")))
	assert.Zero(t, c.ProcOutput([]byte("100%  00:00 ETA")))
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assert.Zero(t, c.ProcOutput([]byte("Saving synchronizer state\n")))
	assertEqual(t, c.Status, "Saving synchronizer state")
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)

	assert.Zero(t, c.ProcOutput([]byte("Synchronization complete at 18:31:20  (1 item transferred, 0 skipped, 0 failed)\n")))
	assertEqual(t, c.Status, "Sync complete (1 item transferred)")
	assert.False(t, c.Busy)
	assert.True(t, c.Running)

	assert.Zero(t, c.ProcExit(0, nil))
	assert.False(t, c.Running)
}

func TestTerse(t *testing.T) { // unison -terse
	c := NewCore()

	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l")})
	assertEqual(t, c.Status, "Assembling plan")
	assertEqual(t, c.Left, "left")
	assertEqual(t, c.Right, "right")
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")))
	assertEqual(t, c.Status, "Ready to synchronize")

	assertEqual(t, c.Sync(),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte(">")})
	assert.Zero(t, c.ProcOutput([]byte(">\r")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y")})
	assert.Zero(t, c.ProcOutput([]byte("y")))
	assertEqual(t, c.Status, "Propagating updates")

	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file one\n")))
	assert.Zero(t, c.ProcOutput([]byte("Synchronization complete at 01:50:54  (1 item transferred, 0 skipped, 0 failed)\n")))
	assert.Zero(t, c.ProcExit(0, nil))
}

func TestProgressLookingForChanges(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n")))
	assert.Zero(t, c.ProcOutput([]byte("Looking for changes\n")))

	assertEqual(t, c.ProcOutput([]byte("| some/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "some/file")
	assertEqual(t, c.ProgressFraction, -1)

	assert.Zero(t, c.ProcOutput([]byte("\r           \r")))

	assertEqual(t, c.ProcOutput([]byte("| another/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "another/file")
	assert.Zero(t, c.ProcOutput([]byte("\r              \r")))

	assertEqual(t, c.ProcOutput([]byte("/ another/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "another/file")
	assert.Zero(t, c.ProcOutput([]byte("\r              \r")))

	assertEqual(t, c.ProcOutput([]byte("- another/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "another/file")
	assert.Zero(t, c.ProcOutput([]byte("\r              \r")))

	assertEqual(t, c.ProcOutput([]byte("\\ another/file")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "another/file")
	assert.Zero(t, c.ProcOutput([]byte("\r              \r")))

	assertEqual(t, c.ProcOutput([]byte("\\ some file name/so long/that it has...ellipsized to fit into the output")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "some file name/so long/that it has...ellipsized to fit into the output")
	assert.Zero(t, c.ProcOutput([]byte("\r                                                                      \r")))

	assert.Zero(t, c.ProcOutput([]byte("  Waiting for changes from server\n")))
	assertEqual(t, c.Status, "Waiting for changes from server")
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)
}

func TestProgressPropagatingUpdates(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("\nleft           right              \n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte("l")})
	assert.Zero(t, c.ProcOutput([]byte("l\n  ")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assert.Zero(t, c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{PlanReady: true})
	assertEqual(t, c.Sync(),
		Update{Input: []byte("0")})
	assert.Zero(t, c.ProcOutput([]byte("0\n")))
	assertEqual(t, c.ProcOutput([]byte("changed  ---->            one  [f] ")),
		Update{Input: []byte(">")})
	assert.Zero(t, c.ProcOutput([]byte(">\r")))
	assert.Zero(t, c.ProcOutput([]byte("changed  ---->            one  \n")))
	assertEqual(t, c.ProcOutput([]byte("\nProceed with propagating updates? [] ")),
		Update{Input: []byte("y")})
	assert.Zero(t, c.ProcOutput([]byte("y")))

	assert.Zero(t, c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n")))
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)

	assertEqual(t, c.ProcOutput([]byte("  0%  73:28 ETA")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "0%  73:28 ETA")
	assertEqual(t, c.ProgressFraction, 0.00)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))
	assertEqual(t, c.ProcOutput([]byte("  8%  07:45 ETA")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "8%  07:45 ETA")
	assertEqual(t, c.ProgressFraction, 0.08)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assertEqual(t, c.ProcOutput([]byte(" 13%  07:51 ETA")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "13%  07:51 ETA")
	assertEqual(t, c.ProgressFraction, 0.13)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assertEqual(t, c.ProcOutput([]byte(" 94%  00:01 ETA")),
		Update{Progressed: true})
	assertEqual(t, c.Progress, "94%  00:01 ETA")
	assertEqual(t, c.ProgressFraction, 0.94)
	assert.Zero(t, c.ProcOutput([]byte("\r               \r")))

	assert.Zero(t, c.ProcOutput([]byte("[END] Updating file one\n")))
	assertEqual(t, c.Progress, "94%  00:01 ETA")
	assertEqual(t, c.ProgressFraction, 0.94)
}

func TestNonexistentProfile(t *testing.T) {
	c := NewCore()
	assert.Zero(t, c.ProcStart())
	assert.Zero(t, c.ProcOutput([]byte("Usage: unison [options]\n    or unison root1 root2 [options]\n    or unison profilename [options]\n\nFor a list of options, type \"unison -help\".\nFor a tutorial on basic usage, type \"unison -doc tutorial\".\nFor other documentation, type \"unison -doc topics\".\n\nProfile /home/vasiliy/tmp/gunison/.unison/nonexistent.prf does not exist\n")))
	assert.True(t, c.Busy)
	assertEqual(t, c.Status, "Starting unison")
	assertEqual(t, c.ProcExit(1, nil),
		Update{Message: Message{
			Text:       "Unison exited, saying:\nUsage: unison [options]\n[...] Profile /home/vasiliy/tmp/gunison/.unison/nonexistent.prf does not exist",
			Importance: Error,
		}})
	assert.False(t, c.Busy)
	assertEqual(t, c.Status, "Unison exited unexpectedly")
	assert.Nil(t, c.Interrupt)
}

// assertEqual is just assert.Equal with arguments swapped,
// which makes for more readable code in places.
func assertEqual(t *testing.T, actual, expected interface{}) bool { //nolint:unparam
	t.Helper()
	return assert.Equal(t, expected, actual)
}
