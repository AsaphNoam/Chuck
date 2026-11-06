package server

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func fileExtraStamp(st unix.Stat_t) string { return strconv.FormatUint(uint64(st.Flags), 10) }

func fileMetadataSupported(fd int, st unix.Stat_t) bool {
	if st.Flags != 0 {
		return false
	}
	names := make([]byte, 4096)
	n, err := unix.Flistxattr(fd, names)
	if err != nil || (n != 0 && string(names[:n]) != "com.apple.provenance\x00") {
		return false
	}
	// com.apple.provenance is kernel-managed and commonly present even on new
	// files. It is admitted only when the replacement's value is byte-identical
	// (fileAccessMetadataEqual); every other extended attribute is refused.
	// macOS ACLs are not included in listxattr. Ask the kernel for the
	// descriptor's extended-security attribute reference; a nonempty blob is
	// unsupported, including ACLs inherited by a newly created temporary file.
	attrs := unix.Attrlist{Bitmapcount: 5, Commonattr: unix.ATTR_CMN_EXTENDED_SECURITY}
	var buf [4096]byte
	_, _, errno := syscall.Syscall6(unix.SYS_FGETATTRLIST, uintptr(fd), uintptr(unsafe.Pointer(&attrs)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0, 0)
	if errno != 0 {
		return false
	}
	// length (4 bytes), then attrreference_t { offset, length }.
	return binary.LittleEndian.Uint32(buf[:4]) >= 12 && binary.LittleEndian.Uint32(buf[8:12]) == 0
}

func fileAccessMetadataEqual(a, b int) bool {
	read := func(fd int) ([]byte, error) {
		buf := make([]byte, 4096)
		n, err := unix.Fgetxattr(fd, "com.apple.provenance", buf)
		if err == unix.ENOATTR {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}
	x, err := read(a)
	y, err2 := read(b)
	return err == nil && err2 == nil && bytes.Equal(x, y)
}
