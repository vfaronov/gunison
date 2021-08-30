//go:build !(js || plan9 || windows)

package main

import "syscall"

func init() {
	sysProcAttr = &syscall.SysProcAttr{
		// This has two effects:
		// 1. makes ssh pop up an askpass GUI instead of trying to prompt for password on the terminal;
		// 2. establishes a process group, enabling Gunison to send signals to ssh and other children
		//    along with Unison itself.
		Setsid: true,
	}
}
