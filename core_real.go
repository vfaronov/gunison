// +build !coremock

package main

func NewCore() *Core {
	return &Core{}
}
