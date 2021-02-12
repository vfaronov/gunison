package main

import (
	"bytes"
)

type Core struct {
	Running          bool
	Busy             bool
	Status           string
	Progress         string  // empty string iff not progressing
	ProgressFraction float64 // 0 to 1; or -1 for unknown

	Left, Right string
	Items       []Item
	Plan        map[string]Action

	ProcStart  func() Update
	ProcOutput func([]byte) Update
	ProcExit   func(int, error) Update
	ProcError  func(error) Update

	Sync      func() Update
	Quit      func() Update
	Abort     func() Update
	Interrupt func() Update
	Kill      func() Update

	buf bytes.Buffer
}

type Update struct {
	Progressed bool
	PlanReady  bool
	Input      []byte
	Interrupt  bool
	Kill       bool
}

type Item struct {
	Path        string
	Left, Right Content
	Action      Action
}

type Content struct {
	Type   Type
	Status Status
	Props  string
}

type Type byte

const (
	Absent Type = 1 + iota
	File
	Directory
	Symlink
)

type Status byte

const (
	Unchanged Status = 1 + iota
	Created
	Modified
	PropsChanged
	Deleted
)

type Action byte

const (
	Conflict Action = 1 + iota
	LeftToRight
	MaybeLeftToRight
	RightToLeft
	MaybeRightToLeft
	Merge
)
