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

// TestDisplayItems is a table-driven test for displayItems.
func TestDisplayItems(t *testing.T) {
	cases := []struct {
		name     string
		items    []Item
		sort     sortRule
		expected []interface{}
	}{
		{
			name:     lineno(),
			items:    []Item{},
			expected: []interface{}{},
		},
		{
			name: lineno(),
			items: []Item{
				ltr("foo"),
			},
			expected: []interface{}{
				o, "foo", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				ltr("foo/bar/baz"),
			},
			expected: []interface{}{
				o, "foo/bar/baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				ltr("foo"),
				ltr("foo/bar"),
			},
			expected: []interface{}{
				o, "foo", "→",
				o, "foo/bar", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				ltr("foo/bar"),
				ltr("foo/baz"),
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
				ltr("foo/bar"),
				ltr("foo/bar/baz"),
			},
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar", "→",
				o__o, "bar/baz", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				ltr("foo/bar"),
				ltr("foo/bar/baz"),
				ltr("foo/bar/qux"),
			},
			expected: []interface{}{
				o, "foo", "→",
				o__o, "bar", "→",
				o__o, "bar/baz", "→",
				o__o, "bar/qux", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				ltr("foo/bar"),
				rtl("foo/baz"),
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
				{
					Path:           "",
					Left:           Content{Directory, PropsChanged, "modified on 2021-02-26 at 17:06:22  size 0         rwx------"},
					Right:          Content{Directory, PropsChanged, "modified on 2021-02-25 at 17:06:22  size 0         rwxr-xr-x"},
					Recommendation: LeftToRight,
				},
				ltr("foo"),
			},
			expected: []interface{}{
				o, "entire replica", "→",
				o, "foo", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				ltr("foo/bar/baz"),
				ltr("foo/bar/qux"),
			},
			expected: []interface{}{
				o, "foo/bar", "→",
				o__o, "baz", "→",
				o__o, "qux", "→",
			},
		},
		{
			name: lineno(),
			items: []Item{
				rtl("foo/bar/1"),
				rtl("foo/bar/2"),
				rtl("foo/baz"),
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
				ltr("foo/bar/1"),
				ltr("foo/bar/2"),
				rtl("foo/baz/3"),
				rtl("foo/baz/4"),
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
				rtl("foo/bar/1"),
				rtl("foo/baz"),
				rtl("foo/bar/2"),
				rtl("foo/bar/3"),
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
				ltr("foo/1"),
				ltr("foo/2"),
				ltr("foo/3"),
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
				ltr("foo/3"),
				ltr("foo/2"),
				ltr("foo/1"),
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
				ltr("foo/qux/1"),
				rtl("foo/qux/2"),
				rtl("foo/xyzzy"),
				rtl("bar/1"),
				rtl("bar/2"),
				rtl("baz/1"),
				{
					Path:           "baz/2",
					Left:           dontCare,
					Right:          dontCare,
					Recommendation: RightToLeft,
					Override:       Skip,
				},
			},
			sort: sortRule{actionColumn, gtk.SORT_ASCENDING},
			expected: []interface{}{
				// Can't derive "foo" and "qux" parent nodes because they would have "•••" in the action
				// column, which sorts after "→" (Mixed > LeftToRight) and would thus violate the current
				// sort rule. We could make Mixed < LeftToRight and then fix this particular case,
				// but it would require a bunch of extra code, and would still not work for "baz" below,
				// nor with descending sort order, so we don't bother for now.
				// TODO: Again, is this too cautious? Would it be more useful to make derived notes
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
			currentSort = c.sort
			displayItems()
			assertTree(t, treestore, []int{colName, colAction}, c.expected...)
		})
	}
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
				name = strings.ReplaceAll(name, "entire replica", "")
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
