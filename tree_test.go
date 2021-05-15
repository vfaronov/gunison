package main

import (
	"path"
	"strings"
	"testing"

	"github.com/gotk3/gotk3/gtk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestDisplayItems(t *testing.T) {
	core.Items = []Item{
		{
			Path:           "", // entire replica
			Left:           Content{Directory, PropsChanged, "modified on 2021-02-06 at 18:41:58  size 0         rwx------"},
			Right:          Content{Directory, Unchanged, "modified on 2021-02-05 at 18:41:58  size 0         rwxr-xr-x"},
			Recommendation: LeftToRight,
		},
		{
			Path:           "foo/baz/789",
			Left:           Content{File, Created, "modified on 2021-02-06 at 18:41:58  size 1146      rw-r--r--"},
			Right:          Content{Directory, Created, "modified on 2021-02-05 at 18:41:58  size 0         rwxr-xr-x"},
			Recommendation: Skip,
		},
		ltr("bar"),
		rtl("foo/123/456/789"),
		rtl("foo/123/456/000"),
		rtl("foo/123/abc"),
		ltr("foo/bar/baz"),
		rtl("foo/bar/qux"),
		ltr("foo/baz/123"),
		{
			Path:           "foo/baz/456",
			Left:           Content{Directory, PropsChanged, "modified on 2021-02-06 at 18:41:58  size 0         rwx------"},
			Right:          Content{Directory, Unchanged, "modified on 2021-02-05 at 18:41:58  size 0         rwxr-xr-x"},
			Recommendation: LeftToRight,
		},
		ltr("foo/baz/456/file1"),
		ltr("foo/baz/456/file2"),
		rtl("foo/baz/xyzzy/a"),
		rtl("foo/baz/xyzzy/b"),
		ltr("foo/qux/123"),
		ltr("foo/qux/456"),
		ltr("foo/qux/789/subdir/a"),
		ltr("foo/qux/789/subdir/b"),
	}

	displayItems()

	cols := []int{colName, colAction}
	assertEqual(t, probeRow(t, "0", cols...), []interface{}{"", "→"})
	assertEqual(t, probeRow(t, "1", cols...), []interface{}{"foo/baz/789", "←?→"})
	assertEqual(t, probeRow(t, "2", cols...), []interface{}{"bar", "→"})
	assertEqual(t, probeRow(t, "3", cols...), []interface{}{"foo/123", "←"})
	assertEqual(t, probeRow(t, "3:0", cols...), []interface{}{"456", "←"})
	assertEqual(t, probeRow(t, "3:0:0", cols...), []interface{}{"789", "←"})
	assertEqual(t, probeRow(t, "3:0:1", cols...), []interface{}{"000", "←"})
	assertEqual(t, probeRow(t, "3:1", cols...), []interface{}{"abc", "←"})
	assertEqual(t, probeRow(t, "4", cols...), []interface{}{"foo/bar", "•••"})
	assertEqual(t, probeRow(t, "4:0", cols...), []interface{}{"baz", "→"})
	assertEqual(t, probeRow(t, "4:1", cols...), []interface{}{"qux", "←"})
	assertEqual(t, probeRow(t, "5", cols...), []interface{}{"foo/baz/123", "→"})
	assertEqual(t, probeRow(t, "6", cols...), []interface{}{"foo/baz/456", "→"})
	assertEqual(t, probeRow(t, "7", cols...), []interface{}{"foo/baz/456/file1", "→"})
	assertEqual(t, probeRow(t, "8", cols...), []interface{}{"foo/baz/456/file2", "→"})
	assertEqual(t, probeRow(t, "9", cols...), []interface{}{"foo/baz/xyzzy", "←"})
	assertEqual(t, probeRow(t, "9:0", cols...), []interface{}{"a", "←"})
	assertEqual(t, probeRow(t, "9:1", cols...), []interface{}{"b", "←"})
	assertEqual(t, probeRow(t, "10", cols...), []interface{}{"foo/qux", "→"})
	assertEqual(t, probeRow(t, "10:0", cols...), []interface{}{"123", "→"})
	assertEqual(t, probeRow(t, "10:1", cols...), []interface{}{"456", "→"})
	assertEqual(t, probeRow(t, "10:2", cols...), []interface{}{"789/subdir", "→"})
	assertEqual(t, probeRow(t, "10:2:0", cols...), []interface{}{"a", "→"})
	assertEqual(t, probeRow(t, "10:2:1", cols...), []interface{}{"b", "→"})

	total := 0
	forEachNode(func(*gtk.TreeIter) { total++ })
	assertEqual(t, total, 24)
}

var dontCare = Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 0         rwx------"}

func ltr(path string) Item {
	return Item{
		Path:           path,
		Left:           dontCare,
		Right:          dontCare,
		Recommendation: LeftToRight,
	}
}

func rtl(path string) Item {
	return Item{
		Path:           path,
		Left:           dontCare,
		Right:          dontCare,
		Recommendation: RightToLeft,
	}
}

func probeRow(t *testing.T, path string, cols ...int) []interface{} {
	t.Helper()
	p, err := gtk.TreePathNewFromString(path)
	require.NoError(t, err)
	iter, err := treestore.GetIter(p)
	require.NoError(t, err)

	row := make([]interface{}, len(cols))
	for i, col := range cols {
		v, err := treestore.GetValue(iter, col)
		require.NoError(t, err)
		row[i], _ = v.GetString()
	}
	return row
}

func genItems(t *rapid.T) []Item {
	// XXX: this algorithm is duplicated in tools/mockunison
	items := make([]Item, rapid.IntRange(0, 99).Draw(t, "len").(int))
	seen := make(map[string]bool)
	actions := []Action{LeftToRight, RightToLeft, Merge, Skip}
	for i := 0; i < len(items); i++ {
		// To generate a new Path, take the previous Path (if any),
		// chop off some of its final segments, and append some new segments.
		newpath := ""
		if i > 0 {
			newpath = items[i-1].Path
			if rapid.IntRange(0, 99).Draw(t, "choppy").(int) == 0 { // Occasionally
				// we may have e.g. "foo/bar" (dir props changed) and "foo/bar/baz" (modified).
				// To simulate this, don't chop off "bar", just append "baz".
				items[i-1].Left = Content{Type: Directory, Status: PropsChanged}
				items[i-1].Right = Content{Type: Directory, Status: PropsChanged}
			} else {
				maxchop := strings.Count(newpath, "/") + 1
				for nchop := rapid.IntRange(1, maxchop).Draw(t, "nchop").(int); nchop > 0; nchop-- {
					newpath = path.Dir(newpath)
				}
				if newpath == "." { // returned by path.Dir
					newpath = ""
				}
			}
		}
		if rapid.IntRange(0, 99).Draw(t, "empty").(int) > 0 { // Path may be empty ("entire replica").
			for ngrow := rapid.IntRange(1, 5).Draw(t, "ngrow").(int); ngrow > 0; ngrow-- {
				segment := rapid.StringMatching(`[a-z]{1,2}`).Draw(t, "segment").(string)
				newpath = path.Join(newpath, segment)
			}
		}

		// Avoid duplicate paths.
		for seen[newpath] {
			newpath += rapid.StringMatching(`[0-9]`).Draw(t, "uniq").(string)
		}
		seen[newpath] = true

		items[i] = Item{
			Path:           newpath,
			Left:           dontCare,
			Right:          dontCare,
			Recommendation: rapid.SampledFrom(actions).Draw(t, "action").(Action),
		}
	}
	return items
}

// TestDisplayItemsContiguous checks the following property:
// Leaf nodes generated by displayItems correspond 1-to-1 to the input items in the same order.
// In other words, displayItems only extracts contiguous groups from the items that it is given,
// never rearranges them.
func TestDisplayItemsContiguous(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		currentSort = sortRule{}
		displayItems()
		cur := 0
		forEachNode(func(iter *gtk.TreeIter) {
			idx := MustGetColumn(treestore, iter, colIdx).(int)
			if treestore.IterHasChild(iter) { // not a leaf node
				assert.Equal(t, invalid, idx)
				return
			}
			assert.Less(t, idx, len(core.Items))
			assert.Equal(t, cur, idx)
			cur++
		})
	})
}

// TestDisplayItemsNamesPaths checks the following property:
// The path of every node generated by displayItems equals a join of its name
// and the names of its ancestors (in reverse order).
func TestDisplayItemsNamesPaths(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		currentSort = sortRule{}
		displayItems()
		forEachNode(func(iter *gtk.TreeIter) {
			var names []string
			for iter1 := iter; ; {
				name := MustGetColumn(treestore, iter1, colName).(string)
				names = append([]string{name}, names...)
				parent, _ := treestore.GetIterFirst() // must be a valid TreeIter for the following call
				if !treestore.IterParent(parent, iter1) {
					break
				}
				iter1 = parent
			}
			p := MustGetColumn(treestore, iter, colPath).(string)
			assert.Equal(t,
				path.Join(names...),
				strings.TrimRight(p, "/"),
			)
		})
	})
}

// TestDisplayItemsAncestors checks the following property:
// For any node1 and node2 (with path1 and path2) generated by displayItems,
// node1 is an ancestor of node2 iff path1 is an ancestor of path2 and node1 has any children.
// In other words, displayItems generates a parent node only when it can contain all the relevant items.
func TestDisplayItemsAncestors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		currentSort = sortRule{}
		displayItems()
		forEachNode(func(iter1 *gtk.TreeIter) {
			path1 := MustGetColumn(treestore, iter1, colPath).(string)
			treepath1, err := treestore.GetPath(iter1)
			require.NoError(t, err)
			forEachNode(func(iter2 *gtk.TreeIter) {
				path2 := MustGetColumn(treestore, iter2, colPath).(string)
				treepath2, err := treestore.GetPath(iter2)
				require.NoError(t, err)
				assert.Equal(t,
					strings.HasPrefix(path2, path1+"/") && treestore.IterHasChild(iter1),
					treepath1.IsAncestor(treepath2),
				)
			})
		})
	})
}

// TestDisplayItemsMultipleChildren checks the following property:
// displayItems generates parent nodes only for multiple children
// (because if there's only one child, it can be subsumed into the parent).
func TestDisplayItemsMultipleChildren(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		currentSort = sortRule{}
		displayItems()
		forEachNode(func(iter *gtk.TreeIter) {
			assert.NotEqual(t, 1, treestore.IterNChildren(iter))
		})
	})
}

// TestDisplayItemsSorted checks the following property:
// Parent nodes are inserted by displayItems only where they respect the current sort order
// (if viewed as applying to the entire list of nodes, top to bottom).
func TestDisplayItemsSorted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		var allSortRules = []sortRule{
			{pathColumn, gtk.SORT_ASCENDING},
			{pathColumn, gtk.SORT_DESCENDING},
			{actionColumn, gtk.SORT_ASCENDING},
			{actionColumn, gtk.SORT_DESCENDING},
		}
		setSort(rapid.SampledFrom(allSortRules).Draw(t, "sortRule").(sortRule)) // calls displayItems

		var last interface{}
		forEachNode(func(iter *gtk.TreeIter) {
			var cur interface{}
			switch currentSort.column {
			case pathColumn:
				cur = MustGetColumn(treestore, iter, colPath)
			case actionColumn:
				cur = actionAt(iter)
			}
			if last != nil {
				switch currentSort.order {
				case gtk.SORT_ASCENDING:
					assert.LessOrEqual(t, last, cur)
				case gtk.SORT_DESCENDING:
					assert.GreaterOrEqual(t, last, cur)
				}
			}
			last = cur
		})
	})
}

// TestSetActionAsIfOriginal checks the following property:
// After selecting some nodes and setting some action for them, the tree shows all the same actions
// as if they were the plan originally, before displayItems().
func TestSetActionAsIfOriginal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		currentSort = sortRule{}

		displayItems()
		treeview.ExpandAll() // nodes whose parents are collapsed cannot be selected
		for i := 0; i < 3; i++ {
			treeSelection.UnselectAll()
			forEachNode(func(iter *gtk.TreeIter) {
				if rapid.Bool().Draw(t, "selected").(bool) {
					treeSelection.SelectIter(iter)
				}
			})
			var allActions = []Action{Skip, LeftToRight, RightToLeft, Merge}
			setAction(rapid.SampledFrom(allActions).Draw(t, "action").(Action))
		}
		var actions1 []string
		forEachNode(func(iter *gtk.TreeIter) {
			actions1 = append(actions1, MustGetColumn(treestore, iter, colAction).(string))
		})

		displayItems()
		var actions2 []string
		forEachNode(func(iter *gtk.TreeIter) {
			actions2 = append(actions2, MustGetColumn(treestore, iter, colAction).(string))
		})

		assert.Equal(t, actions2, actions1)
	})
}

func forEachNode(f func(*gtk.TreeIter)) {
	treestore.ForEach(gtk.TreeModelForeachFunc(
		func(_ *gtk.TreeModel, _ *gtk.TreePath, iter *gtk.TreeIter, _ ...interface{}) bool {
			f(iter)
			return false // means "continue ForEach"
		},
	))
}
