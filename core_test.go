// +build !coremock

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinimal(t *testing.T) {
	c := NewCore()
	assert.Equal(t, "Starting Unison", c.Status)
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

	var upd Update
	upd = c.ProcStart()
	assert.Zero(t, upd)
	assert.True(t, c.Running)
	upd = c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n"))
	assert.Zero(t, upd)
	assert.Equal(t, "Contacting server", c.Status)
	upd = c.ProcOutput([]byte("Looking for changes\n"))
	assert.Zero(t, upd)
	assert.Equal(t, "Looking for changes", c.Status)
	upd = c.ProcOutput([]byte("Reconciling changes\n"))
	assert.Zero(t, upd)
	assert.Equal(t, "Reconciling changes", c.Status)
	upd = c.ProcOutput([]byte("\nleft           right              \n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  [f] "))
	assert.Equal(t, "Assembling plan", c.Status)
	assert.Equal(t, "left", c.Left)
	assert.Equal(t, "right", c.Right)
	assert.Equal(t, []byte("l"), upd.Input)
	upd = c.ProcOutput([]byte("l\n  "))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  \n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("left         : changed file       modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  [f] "))
	assert.Equal(t, Update{PlanReady: true}, upd)
	assert.Equal(t, "Ready to synchronize", c.Status)
	assert.False(t, c.Busy)
	expectedItems := []Item{
		{
			Path:   "one",
			Left:   Content{File, Modified, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--"},
			Right:  Content{File, Unchanged, "modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--"},
			Action: LeftToRight,
		},
	}
	assert.Equal(t, expectedItems, c.Items)
	require.NotNil(t, c.Sync)
	assert.NotNil(t, c.Quit)

	upd = c.Sync()
	assert.Equal(t, Update{Input: []byte("0")}, upd)
	assert.Equal(t, "Starting synchronization", c.Status)
	assert.True(t, c.Busy)
	assert.Nil(t, c.Sync)
	assert.Nil(t, c.Quit)
	assert.NotNil(t, c.Abort)
	upd = c.ProcOutput([]byte("0\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  [f] "))
	assert.Equal(t, Update{Input: []byte(">")}, upd)
	upd = c.ProcOutput([]byte(">\r"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  \n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("\nProceed with propagating updates? [] "))
	assert.Equal(t, Update{Input: []byte("y")}, upd)
	upd = c.ProcOutput([]byte("y\n"))
	assert.Zero(t, upd)

	upd = c.ProcOutput([]byte("Propagating updates\n"))
	assert.Zero(t, upd)
	assert.Equal(t, "Propagating updates", c.Status)
	upd = c.ProcOutput([]byte("\n\nUNISON 2.51.3 (OCAML 4.11.1) started propagating changes at 18:31:20.92 on 08 Feb 2021\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("100%  00:00 ETA"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "100%  00:00 ETA", c.Progress)
	assert.Equal(t, 1.00, c.ProgressFraction)
	upd = c.ProcOutput([]byte("\r               \r"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("[END] Updating file one\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("100%  00:00 ETA"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("\r               \r"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("UNISON 2.51.3 (OCAML 4.11.1) finished propagating changes at 18:31:20.92 on 08 Feb 2021\n\n\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("100%  00:00 ETA"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("\r               \r"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("Saving synchronizer state\n"))
	assert.Zero(t, upd)
	assert.Equal(t, "Saving synchronizer state", c.Status)
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)

	upd = c.ProcOutput([]byte("Synchronization complete at 18:31:20  (1 item transferred, 0 skipped, 0 failed)\n"))
	assert.Zero(t, upd)
	assert.Equal(t, "Sync complete (1 item transferred)", c.Status)
	assert.False(t, c.Busy)
	assert.True(t, c.Running)

	upd = c.ProcExit(0, nil)
	assert.Zero(t, upd)
	assert.False(t, c.Running)
}

func TestTerse(t *testing.T) { // unison -terse
	c := NewCore()
	var upd Update

	c.ProcStart()
	upd = c.ProcOutput([]byte("\nleft           right              \n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  [f] "))
	assert.Equal(t, "Assembling plan", c.Status)
	assert.Equal(t, "left", c.Left)
	assert.Equal(t, "right", c.Right)
	assert.Equal(t, Update{Input: []byte("l")}, upd)
	upd = c.ProcOutput([]byte("l\n  "))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  \n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  [f] "))
	assert.Zero(t, upd)
	assert.Equal(t, "Ready to synchronize", c.Status)

	upd = c.Sync()
	assert.Equal(t, Update{Input: []byte("0")}, upd)
	upd = c.ProcOutput([]byte("0\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  [f] "))
	assert.Equal(t, []byte(">"), upd.Input)
	upd = c.ProcOutput([]byte(">\r"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  \n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("\nProceed with propagating updates? [] "))
	assert.Equal(t, Update{Input: []byte("y")}, upd)
	upd = c.ProcOutput([]byte("y"))
	assert.Zero(t, upd)
	assert.Equal(t, "Propagating updates", c.Status)

	upd = c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("[END] Updating file one\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("Synchronization complete at 01:50:54  (1 item transferred, 0 skipped, 0 failed)\n"))
	assert.Zero(t, upd)
	upd = c.ProcExit(0, nil)
	assert.Zero(t, upd)
}

func TestProgressLookingForChanges(t *testing.T) {
	c := NewCore()
	var upd Update
	c.ProcStart()
	c.ProcOutput([]byte("Unison 2.51.3 (ocaml 4.11.1): Contacting server...\n"))
	c.ProcOutput([]byte("Looking for changes\n"))

	upd = c.ProcOutput([]byte("| some/file"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "some/file", c.Progress)
	assert.Equal(t, -1, c.ProgressFraction)

	upd = c.ProcOutput([]byte("\r           \r"))
	assert.Zero(t, upd)

	upd = c.ProcOutput([]byte("| another/file"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "another/file", c.Progress)
	c.ProcOutput([]byte("\r              \r"))

	upd = c.ProcOutput([]byte("/ another/file"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "another/file", c.Progress)
	c.ProcOutput([]byte("\r              \r"))

	upd = c.ProcOutput([]byte("- another/file"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "another/file", c.Progress)
	c.ProcOutput([]byte("\r              \r"))

	upd = c.ProcOutput([]byte("\\ another/file"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "another/file", c.Progress)
	c.ProcOutput([]byte("\r              \r"))

	upd = c.ProcOutput([]byte("\\ some file name/so long/that it has...ellipsized to fit into the output"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "some file name/so long/that it has...ellipsized to fit into the output", c.Progress)
	c.ProcOutput([]byte("\r                                                                      \r"))

	upd = c.ProcOutput([]byte("  Waiting for changes from server\n"))
	assert.Zero(t, upd)
	assert.Equal(t, "Waiting for changes from server", c.Status)
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)
}

func TestProgressPropagatingUpdates(t *testing.T) {
	c := NewCore()
	var upd Update
	c.ProcStart()
	c.ProcOutput([]byte("\nleft           right              \n"))
	c.ProcOutput([]byte("changed  ---->            one  [f] "))
	c.ProcOutput([]byte("l\n  "))
	c.ProcOutput([]byte("changed  ---->            one  \n"))
	c.ProcOutput([]byte("left         : changed file       modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-07 at  1:50:31  size 1146      rw-r--r--\n"))
	c.ProcOutput([]byte("changed  ---->            one  [f] "))
	c.Sync()
	c.ProcOutput([]byte("0\n"))
	c.ProcOutput([]byte("changed  ---->            one  [f] "))
	c.ProcOutput([]byte(">\r"))
	c.ProcOutput([]byte("changed  ---->            one  \n"))
	c.ProcOutput([]byte("\nProceed with propagating updates? [] "))
	c.ProcOutput([]byte("y"))

	upd = c.ProcOutput([]byte("[BGN] Updating file one from /home/vasiliy/tmp/gunison/left to /home/vasiliy/tmp/gunison/right\n"))
	assert.Zero(t, upd)
	assert.Empty(t, c.Progress)
	assert.Empty(t, c.ProgressFraction)

	upd = c.ProcOutput([]byte("  0%  73:28 ETA"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "0%  73:28 ETA", c.Progress)
	assert.Equal(t, 0.00, c.ProgressFraction)
	c.ProcOutput([]byte("\r               \r"))
	upd = c.ProcOutput([]byte("  8%  07:45 ETA"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "8%  07:45 ETA", c.Progress)
	assert.Equal(t, 0.08, c.ProgressFraction)
	c.ProcOutput([]byte("\r               \r"))

	upd = c.ProcOutput([]byte(" 13%  07:51 ETA"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "13%  07:51 ETA", c.Progress)
	assert.Equal(t, 0.13, c.ProgressFraction)
	c.ProcOutput([]byte("\r               \r"))

	upd = c.ProcOutput([]byte(" 94%  00:01 ETA"))
	assert.Equal(t, Update{Progressed: true}, upd)
	assert.Equal(t, "94%  00:01 ETA", c.Progress)
	assert.Equal(t, 0.94, c.ProgressFraction)
	c.ProcOutput([]byte("\r               \r"))

	upd = c.ProcOutput([]byte("[END] Updating file one\n"))
	assert.Zero(t, upd)
	assert.Equal(t, "94%  00:01 ETA", c.Progress)
	assert.Equal(t, 0.94, c.ProgressFraction)
}

func TestNonexistentProfile(t *testing.T) {
	c := NewCore()
	var upd Update
	c.ProcStart()
	upd = c.ProcOutput([]byte("Usage: unison [options]\n    or unison root1 root2 [options]\n    or unison profilename [options]\n\nFor a list of options, type \"unison -help\".\nFor a tutorial on basic usage, type \"unison -doc tutorial\".\nFor other documentation, type \"unison -doc topics\".\n\nProfile /home/vasiliy/tmp/gunison/.unison/nonexistent.prf does not exist\n"))
	assert.Zero(t, upd)
	assert.True(t, c.Busy)
	assert.Equal(t, "Starting unison", c.Status)
	upd = c.ProcExit(1, nil)
	expectedMessage := Message{
		Text:       "Unison exited, saying:\nUsage: unison [options]\n[...] Profile /home/vasiliy/tmp/gunison/.unison/nonexistent.prf does not exist",
		Importance: Error,
	}
	assert.Equal(t, Update{Message: expectedMessage}, upd)
	assert.False(t, c.Busy)
	assert.Equal(t, "Unison exited unexpectedly", c.Status)
	assert.Nil(t, c.Interrupt)
}
