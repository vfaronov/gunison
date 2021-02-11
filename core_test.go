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
	assert.Equal(t, "Assembling plan", c.Status)
	assert.Equal(t, "left", c.Left)
	assert.Equal(t, "right", c.Right)
	upd = c.ProcOutput([]byte("changed  ---->            one  [f] "))
	assert.Equal(t, []byte("l"), upd.Input)
	upd = c.ProcOutput([]byte("l\n  "))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  \n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("left         : changed file       modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\nright        : unchanged file     modified on 2021-02-08 at 18:30:50  size 1146      rw-r--r--\n"))
	assert.Zero(t, upd)
	upd = c.ProcOutput([]byte("changed  ---->            one  [f] "))
	assert.Zero(t, upd)
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
	assert.True(t, c.Running)
}
