// +build gtk

package main

import (
	"testing"
	"time"

	"github.com/gotk3/gotk3/gtk"
	"github.com/stretchr/testify/require"
)

func init() {
	needGTK = true
}

func TestDisplayItems(t *testing.T) {
	initTreeStore(t)
	core = &Core{
		Items: []Item{
			{
				Path: "seventeen",
				Left: Content{File, Created, "modified on 2021-02-06 at 18:42:07  size 0         rw-r--r--",
					time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local), 0},
				Right:  Content{Absent, Unchanged, "", time.Time{}, 0},
				Action: LeftToRight,
			},
			{
				Path: "six/eight",
				Left: Content{File, Unchanged, "modified on 2021-02-06 at 18:42:07  size 1146      rw-r--r--",
					time.Date(2021, 2, 6, 18, 42, 7, 0, time.Local), 1146},
				Right: Content{File, Modified, "modified on 2021-02-06 at 18:42:08  size 1147000   rw-rw-r--",
					time.Date(2021, 2, 6, 18, 42, 8, 0, time.Local), 1147000},
				Action: RightToLeft,
			},
		},
	}
	displayItems()

	assertEqual(t, probeRow(t, treestore, "1", colPath, colLeft, colAction, colRight),
		[]interface{}{"six/eight", "", "←", "changed"})
}

func initTreeStore(t *testing.T) {
	t.Helper()
	builder, err := gtk.BuilderNewFromString(ui)
	require.NoError(t, err)
	treestore = mustGetObject(builder, "treestore").(*gtk.TreeStore)
}

func probeRow(t *testing.T, store *gtk.TreeStore, path string, cols ...int) []interface{} {
	t.Helper()
	p, err := gtk.TreePathNewFromString(path)
	require.NoError(t, err)
	iter, err := store.GetIter(p)
	require.NoError(t, err)

	row := make([]interface{}, len(cols))
	for i, col := range cols {
		v, err := store.GetValue(iter, col)
		require.NoError(t, err)
		row[i], err = v.GetString()
		require.NoError(t, err)
	}
	return row
}
