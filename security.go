package main

import (
	"syscall"
)

/* root should always be 0. */
const Root = 0

/* nobody UID/GID on most BSD/Linux systems. */
const Nobody = 65534

func isUser(uid int) bool {
	if syscall.Getuid() == uid {
		return true
	} else {
		return false
	}
}
