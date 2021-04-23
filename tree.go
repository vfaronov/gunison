package main

import (
	"fmt"
	"html"
	"mime"
	"path"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

const (
	colIdx = iota
	colName
	colLeft
	colRight
	colAction
	colIconName
	colActionColor
	colPath
)

const invalid = -1

// displayItems makes the treeview display core.Items, one leaf node per Item,
// possibly arranging them in a tree and generating parent nodes as appropriate.
// It satisfies several properties defined in tree_test.go.
func displayItems() {
	// First, we do a pass over all items to find path prefixes (directories)
	// covering contiguous runs of items, which we will group into parent nodes.
	type span struct{ start, end int }
	covers := make(map[string]span, 2*len(core.Items)) // at least one entry per item, plus some prefixes
	for i := range core.Items {
		path := core.Items[i].Path
		// Each item must be a leaf node, so need to prevent deriving a parent node for the same path.
		covers[path] = span{invalid, invalid}
		for j := 1; j < len(path); j++ { // Iterate over all prefixes of the path.
			if path[j] != '/' {
				continue
			}
			prefix := path[:j]
			cover, ok := covers[prefix]
			switch {
			case !ok: // new prefix
				cover.start = i
				cover.end = i
			case cover.end == i-1: // continuing prefix
				cover.end = i
			default: // discontiguous prefix
				cover.end = invalid
			}
			covers[prefix] = cover
		}
	}

	// On my system, this makes the following code 30% faster on a large plan.
	reattach := DetachModel(treeview)
	defer reattach()

	treestore.Clear()

	// Now generate the nodes, keeping a stack of parents.
	type frame struct {
		prefix string
		iter   *gtk.TreeIter
		end    int
	}
	parent := frame{end: len(core.Items) - 1}
	var stack []frame
	for i, item := range core.Items {
		// Close parents (pop frames) as necessary.
		for i > parent.end {
			parent = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		}

		// Open new parents for the prefixes that begin at this item and cover multiple items.
		// But postpone creating a parent until we see a prefix with a shorter span,
		// because if several prefixes cover the same span, we collapse them into one parent.
		path := item.Path
		var lastPrefix string
		lastCover := span{invalid, invalid}
		dumpPrefix := func() {
			iter := treestore.Append(parent.iter)
			name := strings.TrimLeft(lastPrefix[len(parent.prefix):], "/")
			mustf(treestore.SetValue(iter, colName, name), "set name column")
			mustf(treestore.SetValue(iter, colIconName, "folder"), "set icon-name column")
			mustf(treestore.SetValue(iter, colIdx, invalid), "set idx column")
			mustf(treestore.SetValue(iter, colPath, lastPrefix), "set path column")
			stack = append(stack, parent)
			parent = frame{lastPrefix, iter, lastCover.end}
		}
		for j := 1; j < len(path); j++ { // Iterate over all prefixes of the path.
			if path[j] != '/' {
				continue
			}
			prefix := path[:j]
			cover := covers[prefix]
			if cover.start != i || cover.end <= i {
				continue
			}
			if lastCover.end != invalid && lastCover.end > cover.end {
				dumpPrefix()
			}
			lastPrefix = prefix
			lastCover = cover
		}
		if lastCover.end != invalid {
			dumpPrefix()
		}

		// Finally, display the item itself.
		iter := treestore.Append(parent.iter)
		name := strings.TrimLeft(path[len(parent.prefix):], "/")
		// TODO: here and elsewhere: optimization opportunities that need more bindings in gotk3:
		// - set multiple columns in one cgo call to gtk_tree_store_set
		// - reuse GValues for left, right, icon-name, instead of allocating them anew for each node
		mustf(treestore.SetValue(iter, colName, name), "set name column")
		mustf(treestore.SetValue(iter, colLeft, describeContent(item.Left)), "set left column")
		mustf(treestore.SetValue(iter, colRight, describeContent(item.Right)), "set right column")
		mustf(treestore.SetValue(iter, colIconName, iconName(item)), "set icon-name column")
		mustf(treestore.SetValue(iter, colIdx, i), "set idx column")
		mustf(treestore.SetValue(iter, colPath, path), "set path column")
		displayItemAction(iter, core.Plan[path])
	}
}

func displayItemAction(iter *gtk.TreeIter, act Action) {
	mustf(treestore.SetValue(iter, colAction, actionGlyphs[act]), "set action column")

	var color string
	idx := MustGetColumn(treestore, iter, colIdx).(int)
	orig := core.Items[idx].Action
	// TODO: These colors are all arbitrary, and may not play well with themes.
	// Perhaps colors, as well as strings themselves, should be configurable.
	if act == orig {
		switch act {
		case LeftToRight:
			// Make it easier to distinguish LeftToRight and RightToLeft by painting them differently.
			// FIXME: this is supposed to be "uncolored", i.e. "foreground not set", but
			// setting "foreground-set" from a treestore column doesn't seem to work for me,
			// should find a proper way
			color = "#000000"
		case RightToLeft:
			color = "#3ea8d6"
		case Merge:
			color = "#8f9660"
		case Skip, LeftToRightPartial, RightToLeftPartial:
			color = "#d46526"
		}
	} else {
		color = "#5db55c"
	}
	mustf(treestore.SetValue(iter, colActionColor, color), "set action-color column")
}

func describeContent(c Content) string {
	//nolint:exhaustive
	switch c.Status {
	case Unchanged:
		return ""
	case Created:
		switch c.Type {
		case File:
			return "new file"
		case Symlink:
			return "new link"
		case Directory:
			return "new dir"
		}
	case Modified:
		switch c.Type {
		case File:
			return "changed"
		case Symlink:
			return "changed link"
		case Directory:
			return "changed dir"
		}
	case PropsChanged:
		return "props"
	case Deleted:
		return "deleted"
	}
	if c.Type == Absent {
		return ""
	}
	panic(fmt.Sprintf("impossible replica content: %+v", c))
}

func describeContentFull(c Content) string {
	//nolint:exhaustive
	switch c.Status {
	case Unchanged:
		switch c.Type {
		case File:
			return "unchanged file"
		case Symlink:
			return "unchanged symlink"
		case Directory:
			return "unchanged dir"
		}
	case Created:
		switch c.Type {
		case File:
			return "new file"
		case Symlink:
			return "new symlink"
		case Directory:
			return "new dir"
		}
	case Modified:
		switch c.Type {
		case File:
			return "changed file"
		case Symlink:
			return "changed symlink"
		case Directory:
			return "changed dir"
		}
	case PropsChanged:
		switch c.Type {
		case File:
			return "changed props"
		case Directory:
			return "dir props changed"
		}
	case Deleted:
		return "deleted"
	}
	if c.Type == Absent {
		return "absent"
	}
	panic(fmt.Sprintf("impossible replica content: %+v", c))
}

var (
	actionGlyphs = map[Action]string{
		Skip:               "←?→",
		LeftToRight:        "→",
		LeftToRightPartial: "?→",
		RightToLeft:        "←",
		RightToLeftPartial: "←?",
		Merge:              "←M→",
	}
	actionDescriptions = map[Action]string{
		// TODO: replace "left" and "right" with core.Left and core.Right (also in menus, etc.)
		Skip:               "skip",
		LeftToRight:        "propagate from left to right",
		LeftToRightPartial: "propagate from left to right, partial",
		RightToLeft:        "propagate from right to left",
		RightToLeftPartial: "propagate from right to left, partial",
		Merge:              "merge the versions",
	}
)

func iconName(item Item) string {
	content := item.Left
	if content.Type == Absent {
		content = item.Right
	}
	// This uses only names from freedesktop.org's Icon Naming Specification, which are surely available.
	// TODO: There are probably many more names, de-facto common across themes, that we could use.
	switch content.Type {
	case File:
		// TODO: Use GIO's type-from-filename facility? it probably has better coverage
		switch typ := mime.TypeByExtension(path.Ext(item.Path)); {
		case strings.HasPrefix(typ, "audio/"):
			return "audio-x-generic"
		case strings.HasPrefix(typ, "font/"):
			return "font-x-generic"
		case strings.HasPrefix(typ, "image/"):
			return "image-x-generic"
		case strings.HasPrefix(typ, "text/html"):
			return "text-html"
		case strings.HasPrefix(typ, "video/"):
			return "video-x-generic"
		default:
			return "text-x-generic"
		}
	case Directory:
		return "folder"
	case Symlink:
		return "emblem-symbolic-link"
	default:
		return ""
	}
}

func onTreeviewPopupMenu() {
	// TODO: position at the selected row
	itemMenu.PopupAtWidget(treeview, gdk.GDK_GRAVITY_SOUTH_EAST, gdk.GDK_GRAVITY_SOUTH_EAST, nil)
}

func onTreeviewButtonPressEvent(_ *gtk.TreeView, ev *gdk.Event) bool {
	evb := gdk.EventButtonNewFromEvent(ev)
	// TODO: use gdk_event_triggers_context_menu instead (not available in gotk3 at the moment)
	if evb.Type() == gdk.EVENT_BUTTON_PRESS && evb.Button() == gdk.BUTTON_SECONDARY {
		path, _, _, _, ok := treeview.GetPathAtPos(int(evb.X()), int(evb.Y()))
		if ok && !treeSelection.PathIsSelected(path) {
			treeSelection.UnselectAll()
			treeSelection.SelectPath(path)
		}
		itemMenu.PopupAtPointer(ev)
		return blockDefault // prevent default handler from messing with selection
	}
	return handleDefault
}

func onTreeSelectionChanged() {
	updateMenuItems()
}

func updateMenuItems() {
	selectedItems := 0
	onlyFiles := true
	for li := treeSelection.GetSelectedRows(treestore); li != nil; li = li.Next() {
		if _, item, ok := selectedItem(li); ok {
			selectedItems++
			onlyFiles = onlyFiles && item.Left.Type == File && item.Right.Type == File
		}
	}

	allowAction := core.Sync != nil && selectedItems > 0
	leftToRightMenuItem.SetSensitive(allowAction)
	rightToLeftMenuItem.SetSensitive(allowAction)
	mergeMenuItem.SetSensitive(allowAction && onlyFiles)
	skipMenuItem.SetSensitive(allowAction)

	diffMenuItem.SetSensitive(core.Diff != nil && selectedItems == 1 && onlyFiles)
}

func onLeftToRightMenuItemActivate() { setAction(LeftToRight) }
func onRightToLeftMenuItemActivate() { setAction(RightToLeft) }
func onMergeMenuItemActivate()       { setAction(Merge) }
func onSkipMenuItemActivate()        { setAction(Skip) }

func setAction(act Action) {
	// TODO: this crashes when many (thousands) items are selected: see tools/treecrash
	for li := treeSelection.GetSelectedRows(treestore); li != nil; li = li.Next() {
		if iter, item, ok := selectedItem(li); ok {
			core.Plan[item.Path] = act
			displayItemAction(iter, act)
		}
	}
}

func onDiffMenuItemActivate() {
	for li := treeSelection.GetSelectedRows(treestore); li != nil; li = li.Next() {
		if _, item, ok := selectedItem(li); ok {
			update(core.Diff(item.Path))
			return
		}
	}
}

func onTreeviewQueryTooltip(_ *gtk.TreeView, x, y int, keyboardMode bool, tip *gtk.Tooltip) bool {
	if keyboardMode {
		return treeTooltip(tip)
	}
	return treeTooltipAt(tip, x, y)
}

func treeTooltip(tip *gtk.Tooltip) bool {
	li := treeSelection.GetSelectedRows(treestore)
	if li == nil || li.Length() != 1 {
		return false // only show tooltip when a single row is selected
	}
	iter, item, ok := selectedItem(li)
	if !ok {
		tip.SetText(MustGetColumn(treestore, iter, colPath).(string))
		return true
	}
	tip.SetMarkup(fmt.Sprintf("%s\n<b>%s</b>:\t%s\t%s\n<b>%s</b>:\t%s\t%s\n<b>plan</b>:\t%s",
		html.EscapeString(item.Path),
		html.EscapeString(core.Left),
		html.EscapeString(describeContentFull(item.Left)),
		html.EscapeString(item.Left.Props),
		html.EscapeString(core.Right),
		html.EscapeString(describeContentFull(item.Right)),
		html.EscapeString(item.Right.Props),
		actionDescriptions[core.Plan[item.Path]],
	))
	return true
}

func treeTooltipAt(tip *gtk.Tooltip, x, y int) bool {
	var bx, by int
	treeview.ConvertWidgetToBinWindowCoords(x, y, &bx, &by)
	treepath, column, _, _, ok := treeview.GetPathAtPos(bx, by)
	if !ok {
		return false
	}
	treeview.SetTooltipCell(tip, treepath, column, nil)
	iter, err := treestore.GetIter(treepath)
	if !shouldf(err, "get treestore iter for %v", treepath) {
		return false
	}
	idx := MustGetColumn(treestore, iter, colIdx).(int)
	if idx == invalid {
		tip.SetText(MustGetColumn(treestore, iter, colPath).(string))
		return true
	}
	item := core.Items[idx]
	switch column.GetXOffset() { // don't know how else to determine the column's "identity" from gotk3
	case pathColumn.GetXOffset():
		tip.SetText(item.Path)
	case leftColumn.GetXOffset():
		tip.SetText(fmt.Sprintf("%s: %s %s", core.Left, describeContentFull(item.Left), item.Left.Props))
	case rightColumn.GetXOffset():
		tip.SetText(fmt.Sprintf("%s: %s %s", core.Right, describeContentFull(item.Right), item.Right.Props))
	case actionColumn.GetXOffset():
		tip.SetText(actionDescriptions[core.Plan[item.Path]])
	default:
		return false
	}
	return true
}

func selectedItem(li *glib.List) (*gtk.TreeIter, Item, bool) {
	iter, err := treestore.GetIter(li.Data().(*gtk.TreePath))
	mustf(err, "get tree iter")
	idx := MustGetColumn(treestore, iter, colIdx).(int)
	if idx == invalid {
		return iter, Item{}, false
	}
	return iter, core.Items[idx], true
}
