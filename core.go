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
	Plan        []Item

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
	Dir
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

type Action int

const (
	Error Action = 1 + iota
	Conflict
	LeftToRight
	MaybeLeftToRight
	RightToLeft
	MaybeRightToLeft
	Merge
	Mixed
)
