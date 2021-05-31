// +build aix android darwin dragonfly freebsd illumos ios linux netbsd openbsd solaris

package main

import "syscall"

func init() {
	sysProcAttr = &syscall.SysProcAttr{
		// Without this, ssh prompts for password on the terminal instead of popping up an askpass GUI.
		Setsid: true,
	}
}
