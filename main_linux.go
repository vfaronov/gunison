package main

import "syscall"

func init() {
	sysProcAttr = &syscall.SysProcAttr{
		// Without this, ssh prompts for password on the terminal instead of popping up an askpass GUI.
		Setsid: true,
	}
}
