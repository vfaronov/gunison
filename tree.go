package main

import (
	"fmt"
	"mime"
	"path"
	"strings"
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
		mustf(treestore.SetValue(iter, colAction, describeAction[item.Action]), "set action column")
		mustf(treestore.SetValue(iter, colIconName, iconName(item)), "set icon-name column")
		mustf(treestore.SetValue(iter, colLeftProps, item.Left.Props), "set left-props column")
		mustf(treestore.SetValue(iter, colRightProps, item.Right.Props), "set right-props column")
	}
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
