/// File: aufs.go
/// Purpose: A wrapper around system calls to mount AUFS filesystems.
/// Author: Damian Eads
package quickbuddy

import (
	// "errors"
	"fmt"
	// "io/ioutil"
	// "os"
	// "os/exec"
	// "path"
	"syscall"
)

// Returns a mount system call option string to create a Copy-on-Write (CoW)
// filesystem when using AUFS.
//
// @param read_only_dir The bottom layer that will not change.
// @param copy_on_write_dir The directory to store changed files or meta-data.
// @param mount_point The directory to mount the copy-on-writ filesystem.
func GetAufsCowMountOptionString(read_only_dir string, copy_on_write_dir string, mount_point string) string {
	return fmt.Sprintf("br=%s=rw:%s=ro", copy_on_write_dir, read_only_dir)
}

// Stack a read-only directory (e.g. an OS rootfs) and writable directory
// as an AUFS filesystem.
//
// @param read_only_dir The bottom layer that will not change.
// @param copy_on_write_dir The directory to store changed files or meta-data.
// @param mount_point The directory to mount the copy-on-writ filesystem.
func MountAufsCoW(read_only_dir string, copy_on_write_dir string, mount_point string) error {
	mount_option_string := GetAufsCowMountOptionString(read_only_dir, copy_on_write_dir, mount_point)
	return syscall.Mount("aufs", mount_point, "aufs", syscall.MS_MGC_VAL, mount_option_string)
}

// Remounts a Aufs filesystem as read-write. An AUFS filesystem used
// for the root filesystem of a container often is (strangely)
// remounted as read-only after a container is shutdown via poweroff
// or lxc-stop. A read-only root filesystem will hang on subsequent
// lxc-starts unless remounted.
//
// FIXME: Use the option information from /proc/mounts or /etc/mtab
// and reduce the number of parameters to a single argument (the mounting
// point).
func RemountAufsCoWReadWrite(read_only_dir string, copy_on_write_dir string, mount_point string) error {
	mount_option_string := GetAufsCowMountOptionString(read_only_dir, copy_on_write_dir, mount_point)
	return syscall.Mount("aufs", mount_point, "aufs",
		syscall.MS_MGC_VAL | syscall.MS_REMOUNT, mount_option_string)
}
