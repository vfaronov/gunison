package main

import (
	"fmt"
	"html"
	"mime"
	"path"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

const (
	colPath = iota
	colLeft
	colRight
	colAction
	colIconName
	colActionColor
	colIdx
)

func displayItems() {
	for i, item := range core.Items {
		iter := treestore.Append(nil)
		mustf(treestore.SetValue(iter, colPath, item.Path), "set path column")
		mustf(treestore.SetValue(iter, colLeft, describeContent(item.Left)), "set left column")
		mustf(treestore.SetValue(iter, colRight, describeContent(item.Right)), "set right column")
		mustf(treestore.SetValue(iter, colIconName, iconName(item)), "set icon-name column")
		mustf(treestore.SetValue(iter, colIdx, i), "set idx column")
		displayItemAction(iter, item.Action)
	}
}

func displayItemAction(iter *gtk.TreeIter, act Action) {
	mustf(treestore.SetValue(iter, colAction, describeAction[act]), "set action column")

	var color string
	idx := MustGetColumn(treestore, iter, colIdx).(int)
	orig := core.Items[idx].Action
	// This choice of colors is close to that of unison-gtk, but left-to-right is uncolored.
	// This assymmetry between left-to-right and right-to-left makes them easier to tell apart.
	// TODO: This is all arbitrary, and may not play well with themes.
	// Perhaps colors, as well as strings themselves, should be configurable.
	if act == orig {
		switch act {
		case LeftToRight:
			// FIXME: this is supposed to be "uncolored", i.e. "foreground not set", but
			// setting "foreground-set" from a treestore column doesn't seem to work for me,
			// should find a proper way
			color = "#000000"
		case RightToLeft:
			color = "#3ea8d6"
		case Merge:
			color = "#8f9660"
		case Skip, MaybeLeftToRight, MaybeRightToLeft:
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
	panic(fmt.Sprintf("impossible replica content: %+v", c))
}

func describeContentFull(c Content) string {
	//nolint:exhaustive
	switch c.Status {
	case Unchanged:
		switch c.Type {
		case Absent:
			return "absent"
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
	panic(fmt.Sprintf("impossible replica content: %+v", c))
}

var (
	describeAction = map[Action]string{
		Skip:             "←?→",
		LeftToRight:      "→",
		MaybeLeftToRight: "?→",
		RightToLeft:      "←",
		MaybeRightToLeft: "←?",
		Merge:            "←M→",
	}
	describeActionFull = map[Action]string{
		Skip:        "skip",
		LeftToRight: "propagate from left to right",
		RightToLeft: "propagate from right to left",
		Merge:       "merge the versions",
	}
)

func iconName(item Item) string {
	content := item.Left
	if content.Type == Absent {
		content = item.Right
	}
	switch content.Type {
	case File:
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
	nsel := treeSelection.CountSelectedRows()

	allowAction := core.Sync != nil && nsel > 0
	leftToRightMenuItem.SetSensitive(allowAction)
	rightToLeftMenuItem.SetSensitive(allowAction)
	mergeMenuItem.SetSensitive(allowAction)
	skipMenuItem.SetSensitive(allowAction)

	diffMenuItem.SetSensitive(core.Diff != nil && nsel == 1)
}

func onLeftToRightMenuItemActivate() { setAction(LeftToRight) }
func onRightToLeftMenuItemActivate() { setAction(RightToLeft) }
func onMergeMenuItemActivate()       { setAction(Merge) }
func onSkipMenuItemActivate()        { setAction(Skip) }

func setAction(act Action) {
	forEachSelectedPath(func(iter *gtk.TreeIter, path string) {
		core.Plan[path] = act
		displayItemAction(iter, act)
	})
}

func onDiffMenuItemActivate() {
	forEachSelectedPath(func(_ *gtk.TreeIter, path string) {
		update(core.Diff(path))
	})
}

func forEachSelectedPath(f func(*gtk.TreeIter, string)) {
	treeSelection.SelectedForEach(gtk.TreeSelectionForeachFunc(
		func(_ *gtk.TreeModel, _ *gtk.TreePath, iter *gtk.TreeIter, _ ...interface{}) {
			path := MustGetColumn(treestore, iter, colPath).(string)
			f(iter, path)
		},
	))
}

func onTreeviewQueryTooltip(_ *gtk.TreeView, x, y int, keyboardMode bool, tip *gtk.Tooltip) bool {
	if keyboardMode {
		return treeTooltip(tip)
	}
	return treeTooltipAt(tip, x, y)
}

func treeTooltip(tip *gtk.Tooltip) bool {
	if treeSelection.CountSelectedRows() != 1 {
		return false
	}
	forEachSelectedPath(func(iter *gtk.TreeIter, path string) {
		idx := MustGetColumn(treestore, iter, colIdx).(int)
		item := core.Items[idx]
		tip.SetMarkup(fmt.Sprintf("%s\n<b>%s</b>:\t%s\t%s\n<b>%s</b>:\t%s\t%s\n<b>plan</b>:\t%s",
			html.EscapeString(path),
			html.EscapeString(core.Left),
			html.EscapeString(describeContentFull(item.Left)),
			html.EscapeString(item.Left.Props),
			html.EscapeString(core.Right),
			html.EscapeString(describeContentFull(item.Right)),
			html.EscapeString(item.Right.Props),
			describeActionFull[core.Plan[path]],
		))
	})
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
	item := core.Items[idx]
	switch column.GetXOffset() {
	case pathColumn.GetXOffset():
		tip.SetText(item.Path)
	case leftColumn.GetXOffset():
		tip.SetText(fmt.Sprintf("%s: %s %s", core.Left, describeContentFull(item.Left), item.Left.Props))
	case rightColumn.GetXOffset():
		tip.SetText(fmt.Sprintf("%s: %s %s", core.Right, describeContentFull(item.Right), item.Right.Props))
	default:
		return false
	}
	return true
}
