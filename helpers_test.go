package main

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/gotk3/gotk3/gtk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertEqual is assert.Equal with arguments swapped, which makes for more readable code in places.
func assertEqual(t *testing.T, actual, expected interface{}, msgAndArgs ...interface{}) bool { //nolint:unparam
	t.Helper()
	return assert.Equal(t, expected, actual, msgAndArgs...)
}

// assertTree checks that store contains expected, which must be structured as follows:
// row depth (1-based), then one element for each of columns, this all repeated for each row.
func assertTree(t *testing.T, store *gtk.TreeStore, columns []int, expected ...interface{}) { //nolint:thelper
	// t.Helper is useless here due to GTK cgo frames intervening between here and the main test function.
	stride := 1 + len(columns)
	require.Equal(t, 0, len(expected)%stride)
	i := 0
	treestore.ForEach(gtk.TreeModelForeachFunc(
		func(_ *gtk.TreeModel, treepath *gtk.TreePath, iter *gtk.TreeIter) bool {
			msg := fmt.Sprintf("wrong row %s", treepath)
			if !assert.Less(t, (i+1)*stride-1, len(expected), msg) {
				return true // means "stop ForEach"
			}
			assertEqual(t, treepath.GetDepth(), expected[i*stride], msg)
			for j, column := range columns {
				gv, err := store.GetValue(iter, column)
				require.NoError(t, err, msg)
				value, err := gv.GoValue()
				require.NoError(t, err, msg)
				assertEqual(t, value, expected[i*stride+1+j], msg)
			}
			i++
			return false // means "continue ForEach"
		},
	))
	assertEqual(t, i, len(expected)/stride, "not all expected rows found")
}

// Visual representations of tree depth, for use with assertTree.
const (
	o = 1 + iota
	o__o
	o__o__o
)

// lineno returns "line123" when called from line 123: a convenient identifier for table-driven subtests.
//go:noinline
func lineno() string {
	_, _, line, ok := runtime.Caller(1)
	if !ok {
		panic("lineno: failed to find Caller")
	}
	return fmt.Sprintf("line%d", line)
}
