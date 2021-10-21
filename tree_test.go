package main

import (
	"fmt"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/gotk3/gotk3/gtk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestDisplayItems is a table-driven test for displayItems.
func TestDisplayItems(t *testing.T) {
	cases := []struct {
		name     string
		items    []Item
		squash   bool
		showRoot bool
		sort     sortRule
		expected []interface{}
	}{
		{
			name:     lineno(),
			items:    []Item{},
			expected: []interface{}{},
		},
		{
			name:     lineno(),
			items:    []Item{},
			showRoot: true,
			expected: []interface{}{},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo"),
			},
			expected: []interface{}{
				o, "foo", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo"),
			},
			showRoot: true,
			expected: []interface{}{
				o, "root", "→",
				o__o, "foo", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo"),
				item("bar", RightToLeft),
			},
			expected: []interface{}{
				o, "foo", "→",
				o, "bar", "←",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo"),
				item("bar", RightToLeft),
			},
			showRoot: true,
			expected: []interface{}{
				o, "root", "•••",
				o__o, "foo", "→",
				o__o, "bar", "←",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar/baz"),
			},
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar", "→",
				o__o__o, "baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar/baz"),
			},
			squash: true,
			expected: []interface{}{
				o, "foo/bar/baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo", Directory, PropsChanged, LeftToRight, Directory),
				item("foo/bar"),
			},
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo", Directory, PropsChanged, LeftToRight, Directory),
				item("foo/bar"),
			},
			squash: true,
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo", Directory, PropsChanged, RightToLeft, Directory),
				item("foo/bar"),
			},
			expected: []interface{}{
				o, "foo", "•••",
				o__o, "bar", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar"),
				item("foo/baz"),
			},
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar", "→",
				o__o, "baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar"),
				item("foo/baz"),
			},
			squash: true,
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar", "→",
				o__o, "baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar", Directory, PropsChanged, LeftToRight, Directory),
				item("foo/bar/baz"),
			},
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar", "→",
				o__o__o, "baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar", Directory, PropsChanged, LeftToRight, Directory),
				item("foo/bar/baz"),
			},
			showRoot: true,
			expected: []interface{}{
				o, "root", "→",
				o__o, "foo", "→",
				o__o__o, "bar", "→",
				o__o__o__o, "baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar", Directory, PropsChanged, LeftToRight, Directory),
				item("foo/bar/baz"),
			},
			squash: true,
			expected: []interface{}{
				o, "foo/bar", "→",
				o__o, "baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar", Directory, PropsChanged, LeftToRight, Directory),
				item("foo/bar/baz"),
			},
			squash:   true,
			showRoot: true,
			expected: []interface{}{
				o, "root", "→",
				o__o, "foo/bar", "→",
				o__o__o, "baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar/baz", Modified, Merge, Modified),
				item("foo/bar", Directory, PropsChanged, LeftToRight, Directory),
			},
			expected: []interface{}{
				o, "foo", "•••",
				o__o, "bar/baz", "←M→",
				o__o, "bar", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar", Directory, PropsChanged, LeftToRight, Directory),
				item("foo/bar/baz"),
				item("foo/bar/qux"),
			},
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar", "→",
				o__o__o, "baz", "→",
				o__o__o, "qux", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar"),
				item("foo/baz", RightToLeft),
			},
			expected: []interface{}{
				o, "foo", "•••",
				o__o, "bar", "→",
				o__o, "baz", "←",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("", Directory, PropsChanged, LeftToRight, Directory),
				item("foo"),
			},
			expected: []interface{}{
				o, "root", "→",
				o__o, "foo", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("", Directory, PropsChanged, LeftToRight, Directory),
				item("foo"),
			},
			showRoot: true,
			expected: []interface{}{
				o, "root", "→",
				o__o, "foo", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo", Modified, Merge, Modified),
				item("", Directory, RightToLeft, Directory, PropsChanged),
				item("foo/bar"),
			},
			expected: []interface{}{
				o, "foo", "←M→",
				o, "root", "←",
				o, "foo/bar", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar/baz"),
				item("foo/bar/qux"),
			},
			squash: true,
			expected: []interface{}{
				o, "foo/bar", "→",
				o__o, "baz", "→",
				o__o, "qux", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar/baz/1"),
				item("foo/bar/baz/2"),
				item("foo/qux"),
			},
			squash: true,
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar/baz", "→",
				o__o__o, "1", "→",
				o__o__o, "2", "→",
				o__o, "qux", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar/1", RightToLeft),
				item("foo/bar/2", RightToLeft),
				item("foo/baz", RightToLeft),
			},
			expected: []interface{}{
				o, "foo", "←",
				o__o, "bar", "←",
				o__o__o, "1", "←",
				o__o__o, "2", "←",
				o__o, "baz", "←",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar/1"),
				item("foo/bar/2"),
				item("foo/baz/3", RightToLeft),
				item("foo/baz/4", RightToLeft),
			},
			expected: []interface{}{
				o, "foo", "•••",
				o__o, "bar", "→",
				o__o__o, "1", "→",
				o__o__o, "2", "→",
				o__o, "baz", "←",
				o__o__o, "3", "←",
				o__o__o, "4", "←",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar/1", RightToLeft),
				item("foo/baz", RightToLeft),
				item("foo/bar/2", RightToLeft),
				item("foo/bar/3", RightToLeft),
			},
			expected: []interface{}{
				// Can't derive a parent for "bar" because it is split in two by "baz".
				o, "foo", "←",
				o__o, "bar/1", "←",
				o__o, "baz", "←",
				o__o, "bar/2", "←",
				o__o, "bar/3", "←",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/bar/1", RightToLeft),
				item("foo/baz", RightToLeft),
				item("foo/bar/2", RightToLeft),
				item("foo/bar/3", RightToLeft),
			},
			squash: true,
			expected: []interface{}{
				// Can't derive a parent for "bar" because it is split in two by "baz".
				o, "foo", "←",
				o__o, "bar/1", "←",
				o__o, "baz", "←",
				o__o, "bar/2", "←",
				o__o, "bar/3", "←",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/1"),
				item("foo/2"),
				item("foo/3"),
			},
			sort: sortRule{pathColumn, gtk.SORT_ASCENDING},
			expected: []interface{}{
				o, "foo", "→",
				o__o, "1", "→",
				o__o, "2", "→",
				o__o, "3", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/3"),
				item("foo/2"),
				item("foo/1"),
			},
			sort: sortRule{pathColumn, gtk.SORT_DESCENDING},
			expected: []interface{}{
				// Can't derive a parent for "foo" because it would appear
				// before "foo/3" in the tree and thus violate the sort rule.
				// TODO: Does sorting by path like this even make sense,
				// considering that the column actually shows names, not paths?
				o, "foo/3", "→",
				o, "foo/2", "→",
				o, "foo/1", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				item("foo/qux/1"),
				item("foo/qux/2", RightToLeft),
				item("foo/xyzzy", RightToLeft),
				item("bar/1", RightToLeft),
				item("bar/2", RightToLeft),
				item("baz/1", RightToLeft),
				item("baz/2", RightToLeft, Skip),
			},
			sort: sortRule{actionColumn, gtk.SORT_ASCENDING},
			expected: []interface{}{
				// Can't derive "foo" and "qux" parent nodes because they would have "•••" in the action
				// column, which sorts after "→" (Mixed > LeftToRight) and would thus violate the current
				// sort rule. We could make Mixed < LeftToRight and then fix this particular case,
				// but it would require a bunch of extra code, and would still not work for "baz" below,
				// nor with descending sort order, so we don't bother for now.
				// TODO: Again, is this too cautious? Would it be more useful to make derived nodes
				// exempt from the sort rule altogether?
				o, "foo/qux/1", "→",
				o, "foo/qux/2", "←",
				o, "foo/xyzzy", "←",
				o, "bar", "←",
				o__o, "1", "←",
				o__o, "2", "←",
				// Same with "baz" here: can't insert a "•••" between two "←".
				o, "baz/1", "←",
				o, "baz/2", "←?→",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			core.Items = c.items
			squash = c.squash
			showRoot = c.showRoot
			currentSort = c.sort
			displayItems()
			assertTree(t, []int{colName, colAction}, c.expected...)
		})
	}
}

func item(path string, opts ...interface{}) Item {
	it := Item{
		Path:           path,
		Left:           Content{File, Modified, "modified on 2021-02-06 at 18:41:58  size 0         rwx------"},
		Right:          Content{File, Unchanged, "modified on 2021-02-05 at 18:41:58  size 0         rwx------"},
		Recommendation: LeftToRight,
	}
	side := &it.Left
	action := &it.Recommendation
	for _, opt := range opts {
		switch opt := opt.(type) {
		case Content:
			*side = opt
		case Type:
			side.Type = opt
		case Status:
			side.Status = opt
		case string:
			side.Props = opt
		case Action:
			*action = opt
			side = &it.Right
			action = &it.Override
		default:
			panic(fmt.Sprintf("unhandled item option: %#v", opt))
		}
	}
	return it
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
		if rapid.IntRange(0, 99).Draw(t, "empty").(int) > 0 { // Path may be empty (root).
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

		items[i] = item(newpath, rapid.SampledFrom(actions).Draw(t, "action"))
	}
	return items
}

// TestDisplayItemsIndices checks the following property:
// Except for implied parent nodes with colIdx == -1, nodes generated by displayItems correspond 1-to-1
// to the input items in the same order. In other words, displayItems only extracts contiguous groups
// from the items that it is given, never rearranges them.
func TestDisplayItemsIndices(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		squash = rapid.Bool().Draw(t, "squash").(bool)
		currentSort = sortRule{}
		displayItems()
		cur := 0
		forEachNode(func(iter *gtk.TreeIter) {
			idx := MustGetColumn(treestore, iter, colIdx)
			if idx == invalid {
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
		squash = rapid.Bool().Draw(t, "squash").(bool)
		currentSort = sortRule{}
		displayItems()
		forEachNode(func(iter *gtk.TreeIter) {
			var names []string
			for iter1 := iter; ; {
				name := MustGetColumn(treestore, iter1, colName).(string)
				name = strings.ReplaceAll(name, "root", "")
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
		squash = rapid.Bool().Draw(t, "squash").(bool)
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
					PathIsAncestor(path1, path2) && treestore.IterHasChild(iter1),
					treepath1.IsAncestor(treepath2),
					fmt.Sprintf("%s (%s) - %s (%s)", path1, treepath1, path2, treepath2),
				)
			})
		})
	})
}

// TestDisplayItemsSquash checks the following property:
// When squash is enabled, displayItems generates implied parent nodes only for multiple children.
func TestDisplayItemsSquash(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		squash = true
		currentSort = sortRule{}
		displayItems()
		forEachNode(func(iter *gtk.TreeIter) {
			if MustGetColumn(treestore, iter, colIdx) == invalid {
				assert.NotEqual(t, 1, treestore.IterNChildren(iter),
					MustGetColumn(treestore, iter, colPath))
			}
		})
	})
}

// TestDisplayItemsNoSquash checks the following property:
// When squash is disabled, and items are sorted by path ascending,
// each tree node generated by displayItems represents a single path segment.
func TestDisplayItemsNoSquash(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		squash = false
		setSort(sortRule{column: pathColumn, order: gtk.SORT_ASCENDING}) // calls displayItems
		forEachNode(func(iter *gtk.TreeIter) {
			assert.NotContains(t, MustGetColumn(treestore, iter, colName), "/")
		})
	})
}

// TestDisplayItemsSorted checks the following property:
// Parent nodes are inserted by displayItems only where they respect the current sort order
// (if viewed as applying to the entire list of nodes, top to bottom).
func TestDisplayItemsSorted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		squash = rapid.Bool().Draw(t, "squash").(bool)
		setSort(rapid.SampledFrom(allSortRules()).Draw(t, "sortRule").(sortRule)) // calls displayItems

		var last interface{}
		forEachNode(func(iter *gtk.TreeIter) {
			var cur interface{}
			switch currentSort.column {
			case pathColumn:
				cur = pathAt(iter)
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

// TestDisplayItemsMixed checks the following property:
// If a tree node's action is mixed (•••), it has at least one child.
func TestDisplayItemsMixed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		squash = rapid.Bool().Draw(t, "squash").(bool)
		currentSort = sortRule{}
		displayItems()
		forEachNode(func(iter *gtk.TreeIter) {
			if MustGetColumn(treestore, iter, colAction) == "•••" {
				assert.NotZero(t, treestore.IterNChildren(iter),
					MustGetColumn(treestore, iter, colPath))
			}
		})
	})
}

// TestDisplayItemsShowRoot checks the following property:
// When "always show root" is enabled, and items are not sorted by path descending, the tree always
// contains a node corresponding to the root (empty) path. It need not be the tree's root node.
func TestDisplayItemsShowRoot(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		squash = rapid.Bool().Draw(t, "squash").(bool)
		showRoot = true
		sortRule := rapid.SampledFrom(allSortRules()).
			Filter(func(r sortRule) bool { return r != sortRule{pathColumn, gtk.SORT_DESCENDING} }).
			Draw(t, "sortRule").(sortRule)
		setSort(sortRule) // calls displayItem
		hasRoot := false
		forEachNode(func(iter *gtk.TreeIter) {
			if MustGetColumn(treestore, iter, colPath) == "" {
				hasRoot = true
			}
		})
		assert.True(t, hasRoot)
	})
}

// TestSetActionAsIfOriginal checks the following property:
// After selecting some nodes and setting some action for them, the tree shows all the same actions
// as if they were the plan originally, before displayItems().
func TestSetActionAsIfOriginal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		core.Items = rapid.Custom(genItems).Draw(t, "items").([]Item)
		squash = rapid.Bool().Draw(t, "squash").(bool)
		currentSort = sortRule{}

		displayItems()
		treeview.ExpandAll() // nodes whose parents are collapsed cannot be selected
		for i := 0; i < 2; i++ {
			treeSelection.UnselectAll()
			forEachNode(func(iter *gtk.TreeIter) {
				if rapid.Bool().Draw(t, "selected").(bool) {
					treeSelection.SelectIter(iter)
				}
			})
			var allActions = []Action{NoAction, Skip, LeftToRight, RightToLeft, Merge}
			setAction(rapid.SampledFrom(allActions).Draw(t, "action").(Action))
		}
		var actions1, colors1 []string
		forEachNode(func(iter *gtk.TreeIter) {
			actions1 = append(actions1, MustGetColumn(treestore, iter, colAction).(string))
			colors1 = append(colors1, MustGetColumn(treestore, iter, colActionColor).(string))
		})

		displayItems()
		var actions2, colors2 []string
		forEachNode(func(iter *gtk.TreeIter) {
			actions2 = append(actions2, MustGetColumn(treestore, iter, colAction).(string))
			colors2 = append(colors2, MustGetColumn(treestore, iter, colActionColor).(string))
		})

		assert.Equal(t, actions2, actions1)
		assert.Equal(t, colors2, colors1)
	})
}

func forEachNode(f func(*gtk.TreeIter)) {
	treestore.ForEach(gtk.TreeModelForeachFunc(
		func(_ *gtk.TreeModel, _ *gtk.TreePath, iter *gtk.TreeIter) bool {
			f(iter)
			return false // means "continue ForEach"
		},
	))
}

// assertTree checks that treestore contains expected, which must be structured as follows:
// row depth (1-based), then one element for each of columns, this all repeated for each row.
func assertTree(t *testing.T, columns []int, expected ...interface{}) { //nolint:thelper
	// t.Helper is useless here due to GTK cgo frames intervening between here and the main test function.
	stride := 1 + len(columns)
	require.Equal(t, 0, len(expected)%stride)
	i := 0
	treestore.ForEach(gtk.TreeModelForeachFunc(
		func(_ *gtk.TreeModel, treepath *gtk.TreePath, iter *gtk.TreeIter) bool {
			msg := fmt.Sprintf("wrong row %s", treepath)
			if !assert.Less(t, (i+1)*stride-1, len(expected), msg) {
				return true // means "break ForEach"
			}
			assertEqual(t, treepath.GetDepth(), expected[i*stride], msg)
			for j, column := range columns {
				gv, err := treestore.GetValue(iter, column)
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
	o__o__o__o
)

// lineno returns "line123" when called from line 123: a convenient name for table-driven subtests.
//go:noinline
func lineno() string {
	_, _, line, ok := runtime.Caller(1)
	if !ok {
		panic("lineno: failed to find Caller")
	}
	return fmt.Sprintf("line%d", line)
}
