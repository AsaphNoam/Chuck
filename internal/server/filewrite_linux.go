package server

import "golang.org/x/sys/unix"

func fileExtraStamp(st unix.Stat_t) string { return "" }

func fileMetadataSupported(fd int, st unix.Stat_t) bool {
	// Linux POSIX ACLs are system.posix_acl_* xattrs, so refusing all xattrs
	// also refuses ACLs whose semantics replacement could otherwise discard.
	n, err := unix.Flistxattr(fd, nil)
	return err == nil && n == 0
}

func fileAccessMetadataEqual(a, b int) bool { return true }
