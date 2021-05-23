package main

import (
	"fmt"
	"html"
	"log"
	"mime"
	"path"
	"sort"
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
	// We will also determine combined actions to be displayed on these parent nodes.
	type span struct {
		start, end int
		action     Action
		overridden bool
	}
	covers := make(map[string]span, 2*len(core.Items)) // at least one entry per item, plus some prefixes
	for i, item := range core.Items {
		path := item.Path
		// Each item must be a leaf node, so need to prevent deriving a parent node for the same path.
		covers[path] = span{end: invalid}
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
				cover.overridden = item.IsOverridden()
			case cover.end == i-1: // continuing prefix
				cover.end = i
			default: // discontiguous prefix
				cover.end = invalid
			}
			cover.action, cover.overridden = combineAction(cover.action, cover.overridden,
				item.Action(), item.IsOverridden())
			covers[prefix] = cover
		}
	}

	// On my system, this makes the following code 30% faster on a large plan.
	reattachModel := DetachModel(treeview)

	treestore.Clear()

	// As we generate tree nodes, we will be keeping a stack of parent nodes.
	// TODO: This should probably be refactored for ease of understanding, but
	// my first attempt to rewrite this as a recursive function made things worse.
	type frame struct {
		prefix string
		iter   *gtk.TreeIter
		end    int
	}
	parent := frame{end: len(core.Items) - 1}
	stack := []frame{}
	openParent := func(prefix string) {
		iter := treestore.Append(parent.iter)
		name := strings.TrimLeft(prefix[len(parent.prefix):], "/")
		mustf(treestore.SetValue(iter, colName, name), "set name column")
		mustf(treestore.SetValue(iter, colIconName, "folder"), "set icon-name column")
		mustf(treestore.SetValue(iter, colIdx, invalid), "set idx column")
		mustf(treestore.SetValue(iter, colPath, prefix), "set path column")
		displayAction(iter, covers[prefix].action, covers[prefix].overridden)
		stack = append(stack, parent)
		parent = frame{prefix, iter, covers[prefix].end}
	}
	closeParent := func() {
		parent = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
	}

	// Walk the items and generate a node for each.
	for i, item := range core.Items {
		for i > parent.end { // Pop stack frames for prefixes that are over.
			closeParent()
		}

		// Open new parents for prefixes that begin at this item and cover multiple items.
		// But postpone opening a parent until we see a prefix with a shorter span,
		// because if several prefixes cover the same span, we collapse them into one parent.
		path := item.Path
		var lastPrefix string
		lastCover := span{end: invalid}
		for j := 1; j < len(path); j++ { // Iterate over all prefixes of the path.
			if path[j] != '/' {
				continue
			}
			prefix := path[:j]
			cover := covers[prefix]
			if cover.start != i || cover.end <= i {
				continue
			}

			// Make sure that the selected sort order is respected.
			if currentSort.column == actionColumn && cover.action != item.Action() {
				continue
			}
			if currentSort.column == pathColumn && currentSort.order == gtk.SORT_DESCENDING {
				// A parent node's path is necessarily "less than" all of its children's.
				// TODO: Does sorting by path like this even make sense,
				// given that the column actually shows names, not paths?
				continue
			}

			if lastCover.end > cover.end {
				openParent(lastPrefix)
			}
			lastPrefix = prefix
			lastCover = cover
		}
		if lastCover.end != invalid {
			openParent(lastPrefix)
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
		displayAction(iter, item.Action(), item.IsOverridden())
	}

	reattachModel()

	// Kick off recursively expanding the tree (triggering onTreeviewRowExpanded) as necessary.
	for iter, ok := treestore.GetIterFirst(); ok; ok = treestore.IterNext(iter) {
		maybeExpandRow(iter)
	}
}

func displayAction(iter *gtk.TreeIter, act Action, overridden bool) {
	// XXX: The values set here are not just for display: they are later used in actionAt, etc.
	mustf(treestore.SetValue(iter, colAction, actionGlyphs[act]), "set action column")
	color := actionColors[act]
	if overridden {
		color = overriddenColor
	}
	mustf(treestore.SetValue(iter, colActionColor, color), "set action-color column")
}

func combineAction(act1 Action, overrid1 bool, act2 Action, overrid2 bool) (act Action, overrid bool) {
	switch act1 {
	case NoAction, act2:
		act = act2
	default:
		act = Mixed
	}
	overrid = overrid1 && overrid2
	return
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
	// TODO: Action glyphs and colors should be user-customizable (but beware unActionGlyphs).
	actionGlyphs = map[Action]string{
		Skip:               "←?→",
		LeftToRight:        "→",
		LeftToRightPartial: "?→",
		RightToLeft:        "←",
		RightToLeftPartial: "←?",
		Merge:              "←M→",
		Mixed:              "•••",
	}
	unActionGlyphs = map[string]Action{} // filled in init below
	actionColors   = map[Action]string{
		LeftToRight:        "#60C1F8",
		RightToLeft:        "#B980FF",
		Merge:              "#FDB363",
		Skip:               "#FF9780",
		LeftToRightPartial: "#FF9780",
		RightToLeftPartial: "#FF9780",
		Mixed:              "#BABABA",
	}
	overriddenColor    = "#4BC74A"
	actionDescriptions = map[Action]string{ // XXX: later changed by setReplicaNames
		Skip:               "skip",
		LeftToRight:        "propagate from left to right",
		LeftToRightPartial: "propagate from left to right, partial",
		RightToLeft:        "propagate from right to left",
		RightToLeftPartial: "propagate from right to left, partial",
		Merge:              "merge the versions",
		Mixed:              "varies between items",
	}
)

func init() {
	for action, glyph := range actionGlyphs {
		unActionGlyphs[glyph] = action
	}
}

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

type sortRule struct {
	column *gtk.TreeViewColumn
	order  gtk.SortType
}

var currentSort sortRule

func onPathColumnClicked()   { cycleSort(pathColumn) }
func onActionColumnClicked() { cycleSort(actionColumn) }

func cycleSort(col *gtk.TreeViewColumn) {
	if currentSort == (sortRule{col, gtk.SORT_ASCENDING}) {
		setSort(sortRule{col, gtk.SORT_DESCENDING})
	} else {
		setSort(sortRule{col, gtk.SORT_ASCENDING})
	}
}

func setSort(rule sortRule) {
	// We don't use GtkTreeModelSortable and its associated facilities, because we
	// don't just sort the nodes that are already being shown in the tree; instead,
	// we sort the flat list of Items and *then* rearrange them into a tree, which
	// becomes very different depending on the sort rule.
	currentSort = rule
	if rule != (sortRule{}) {
		sort.SliceStable(core.Items, func(i, j int) bool {
			// TODO: remember the original order as produced by Unison, fall back to it on equals,
			// and allow the user to return to that original order.
			switch rule {
			case sortRule{pathColumn, gtk.SORT_ASCENDING}:
				return core.Items[i].Path < core.Items[j].Path
			case sortRule{pathColumn, gtk.SORT_DESCENDING}:
				return core.Items[i].Path > core.Items[j].Path
			case sortRule{actionColumn, gtk.SORT_ASCENDING}:
				return core.Items[i].Action() < core.Items[j].Action()
			case sortRule{actionColumn, gtk.SORT_DESCENDING}:
				return core.Items[i].Action() > core.Items[j].Action()
			}
			// XXX: When adding new sort rules, don't forget to update TestDisplayItemsSorted.
			panic("impossible case")
		})
		displayItems()
	}
	DisplaySort(treeview, rule.column, rule.order)
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
	mergeMenuItem.SetSensitive(allowAction) // TODO: for files only
	skipMenuItem.SetSensitive(allowAction)
	revertMenuItem.SetSensitive(allowAction)

	diffMenuItem.SetSensitive(core.Diff != nil && nsel == 1) // TODO: for files only
}

func onLeftToRightMenuItemActivate() { setAction(LeftToRight) }
func onRightToLeftMenuItemActivate() { setAction(RightToLeft) }
func onMergeMenuItemActivate()       { setAction(Merge) }
func onSkipMenuItemActivate()        { setAction(Skip) }
func onRevertMenuItemActivate()      { setAction(NoAction) }

func setAction(act Action) {
	// TODO: this crashes when many (thousands) items are selected: see tools/treecrash

	// Keep track of visited nodes (as sets of gtk_tree_path_to_string) to avoid repeating work.
	updated := map[string]bool{}       // nodes for which we directly set the new action
	invalidated := []map[string]bool{} // ancestors of updated nodes, sorted into groups by tree depth

	// Recursively set the new action on all selected nodes and their descendants,
	// while assembling a list of ancestors to refresh.
	for li := treeSelection.GetSelectedRows(treestore); li != nil; li = li.Next() {
		invalidated = setActionInner(li.Data().(*gtk.TreePath), act, updated, invalidated)
	}

	// Refresh combined actions on all ancestor nodes that have been invalidated,
	// beginning with the deepest ones and moving up the tree
	// (recomputing a node's action can affect its ancestors but not its descendants).
	for i := len(invalidated) - 1; i >= 0; i-- {
		for treepathS := range invalidated[i] {
			if !updated[treepathS] {
				refreshParentAction(treepathS)
			}
		}
	}

	// If the tree was sorted by action, we now have to either sort (and displayItems) again, or
	// just indicate that it's no longer sorted, which is easier and probably more useful.
	if currentSort.column == actionColumn {
		setSort(sortRule{})
	}
}

func setActionInner(
	treepath *gtk.TreePath,
	act Action,
	updated map[string]bool,
	invalidated []map[string]bool,
) []map[string]bool {
	// Avoid repeated work when both a node and its child have been selected by the user.
	if updated[treepath.String()] {
		return invalidated
	}

	iter, err := treestore.GetIter(treepath)
	mustf(err, "get tree iter for %s", treepath)

	// Set the new action on the node and its corresponding plan item (if any).
	if idx := MustGetColumn(treestore, iter, colIdx).(int); idx != invalid {
		item := &core.Items[idx]
		item.Override = act
		displayAction(iter, item.Action(), item.IsOverridden())
		updated[treepath.String()] = true
	} else if act != NoAction { // For parent nodes, this only works if we're actually *setting* action,
		// not resetting to Unison's recommendation, because we don't have a parent node's recommendation
		// precomputed. Instead, that will be updated in refreshParentAction.
		displayAction(iter, act, true)
		updated[treepath.String()] = true
	}

	// Invalidate all ancestors of the node.
	for treepath.Up() {
		depth := treepath.GetDepth()
		if depth < 1 {
			break
		}
		for len(invalidated) < depth {
			invalidated = append(invalidated, map[string]bool{})
		}
		invalidated[depth-1][treepath.String()] = true
	}

	// Recursively update children, if any.
	child, _ := treestore.GetIterFirst()
	for ok := treestore.IterChildren(iter, child); ok; ok = treestore.IterNext(child) {
		treepath, err := treestore.GetPath(child)
		mustf(err, "get tree path")
		invalidated = setActionInner(treepath, act, updated, invalidated)
	}

	return invalidated
}

func refreshParentAction(treepathS string) {
	iter, err := treestore.GetIterFromString(treepathS)
	mustf(err, "get tree iter for parent from %s", treepathS)
	var action Action
	overridden := true
	child, _ := treestore.GetIterFirst()
	for ok := treestore.IterChildren(iter, child); ok; ok = treestore.IterNext(child) {
		action, overridden = combineAction(action, overridden, actionAt(child), isOverriddenAt(child))
	}
	displayAction(iter, action, overridden)
}

func onDiffMenuItemActivate() {
	if core.Diff == nil {
		log.Println("cannot invoke core.Diff because it is already nil")
		update(Update{})
		return
	}
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
	var markup string
	if iter, item, ok := selectedItem(li); ok {
		markup = fmt.Sprintf("%s\n<b>%s</b>:\t%s\t%s\n<b>%s</b>:\t%s\t%s\n<b>action</b>:\t%s",
			html.EscapeString(item.Path),
			html.EscapeString(core.Left),
			html.EscapeString(describeContentFull(item.Left)),
			html.EscapeString(item.Left.Props),
			html.EscapeString(core.Right),
			html.EscapeString(describeContentFull(item.Right)),
			html.EscapeString(item.Right.Props),
			actionDescriptions[item.Action()],
		)
		if item.Action() != item.Recommendation {
			markup += fmt.Sprintf("\n<b>Unison’s recommendation</b>: %s",
				actionDescriptions[item.Recommendation],
			)
		}
	} else {
		markup = fmt.Sprintf("%s\n<small>directory containing items</small>\n<b>action</b>:\t%s",
			html.EscapeString(pathAt(iter)),
			actionDescriptions[actionAt(iter)],
		)
	}
	tip.SetMarkup(markup)
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
	if !shouldf(err, "get tree iter for %v", treepath) {
		return false
	}

	switch column.Native() {
	case pathColumn.Native():
		tip.SetText(pathAt(iter))

	case actionColumn.Native():
		tip.SetText(actionDescriptions[actionAt(iter)])

	case leftColumn.Native(), rightColumn.Native():
		idx := MustGetColumn(treestore, iter, colIdx).(int)
		if idx == invalid {
			return false
		}
		item := core.Items[idx]
		side, content := core.Left, item.Left
		if column.Native() == rightColumn.Native() {
			side, content = core.Right, item.Right
		}
		tip.SetText(fmt.Sprintf("%s: %s %s", side, describeContentFull(content), content.Props))

	default:
		return false
	}
	return true
}

func onTreeviewRowExpanded(_ *gtk.TreeView, iter *gtk.TreeIter) {
	delete(collapsed, pathAt(iter))
	// Automatically expand children unless they have been collapsed by the user.
	// (This will trigger the row-expanded signal on each child, and so proceed recursively.)
	child, _ := treestore.GetIterFirst()
	for ok := treestore.IterChildren(iter, child); ok; ok = treestore.IterNext(child) {
		maybeExpandRow(child)
	}
}

func onTreeviewRowCollapsed(_ *gtk.TreeView, iter *gtk.TreeIter) {
	collapsed[pathAt(iter)] = true
}

func maybeExpandRow(iter *gtk.TreeIter) {
	if collapsed[pathAt(iter)] {
		return
	}
	treepath, err := treestore.GetPath(iter)
	if !shouldf(err, "get treepath from iter") {
		return
	}
	treeview.ExpandRow(treepath, false)
}

func selectedItem(li *glib.List) (*gtk.TreeIter, *Item, bool) {
	iter, err := treestore.GetIter(li.Data().(*gtk.TreePath))
	mustf(err, "get tree iter")
	idx := MustGetColumn(treestore, iter, colIdx).(int)
	if idx == invalid {
		return iter, nil, false
	}
	return iter, &core.Items[idx], true
}

func pathAt(iter *gtk.TreeIter) string {
	return MustGetColumn(treestore, iter, colPath).(string)
}

func actionAt(iter *gtk.TreeIter) Action {
	return unActionGlyphs[MustGetColumn(treestore, iter, colAction).(string)]
}

func isOverriddenAt(iter *gtk.TreeIter) bool {
	return MustGetColumn(treestore, iter, colActionColor).(string) == overriddenColor
}
