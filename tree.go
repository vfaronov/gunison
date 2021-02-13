package main

import (
	"fmt"
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
	colLeftProps
	colRightProps
)

func displayItems() {
	for _, item := range core.Items {
		iter := treestore.Append(nil)
		mustf(treestore.SetValue(iter, colPath, item.Path), "set path column")
		mustf(treestore.SetValue(iter, colLeft, describeContent(item.Left)), "set left column")
		mustf(treestore.SetValue(iter, colRight, describeContent(item.Right)), "set right column")
		mustf(treestore.SetValue(iter, colIconName, iconName(item)), "set icon-name column")
		mustf(treestore.SetValue(iter, colLeftProps, item.Left.Props), "set left-props column")
		mustf(treestore.SetValue(iter, colRightProps, item.Right.Props), "set right-props column")
		displayItemAction(iter, item.Action)
	}
}

func displayItemAction(iter *gtk.TreeIter, act Action) {
	mustf(treestore.SetValue(iter, colAction, describeAction[act]), "set action column")
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

var describeAction = map[Action]string{
	Skip:             "←?→",
	LeftToRight:      "→",
	MaybeLeftToRight: "?→",
	RightToLeft:      "←",
	MaybeRightToLeft: "←?",
	Merge:            "←M→",
}

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
			pathv, err := treestore.GetValue(iter, colPath)
			if !shouldf(err, "get path value") {
				return
			}
			path, err := pathv.GetString()
			if !shouldf(err, "get path string") {
				return
			}
			f(iter, path)
		},
	))
}
