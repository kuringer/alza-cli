//go:build !unix

package client

import "os"

// ponytail: no file lock on non-unix (Windows), parallel writers may duplicate
// a day; stats dedupe per day. Upgrade: LockFileEx via golang.org/x/sys/windows.
func lockFile(*os.File) error { return nil }

func unlockFile(*os.File) {}
