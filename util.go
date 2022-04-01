package quickbuddy
/// File: util.go
/// Purpose: Provides some simple utility functions such as checking
/// file types.
/// Author: Damian Eads
import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

/// Creates a new named FIFO pipe.
///
/// @param filename The filename of the new named FIFO pipe.
func MkFIFO(filename string) error {
	return syscall.Mknod(filename, syscall.S_IFIFO|0660, 0)
}

/// Flushes a named FIFO pipe.
///
/// @param filename The filename of the new named FIFO pipe.
func FlushFIFO(filename string) error {
	// Open the file as non-blocking.
	file, err := os.OpenFile(filename, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	defer file.Close()
	if err != nil {
		return err
	}
	// Create a buffer.
	buffer := make([]byte, 1024)
	var file_err error = nil
	for file_err != nil {
		// Do nothing. We just want to flush the data out of the named pipe.
		_, file_err = file.Read(buffer)
	}
	if file_err == io.EOF {
		return nil
	}
	return file_err
}

// Replaces all occurrences of a string with another string in a text file.
func ReplaceAllInFile(filename string, old_str string, new_str string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	s := string(b)
	s = strings.Replace(s, old_str, new_str, -1)
	file_info, err := os.Stat(filename)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, []byte(s), file_info.Mode())
	return err
}

// Returns true if and only if a file with filename exists and the file is a directory
func DirExists(dir string) bool {
	file_info, err := os.Stat(dir)
	return err == nil && file_info.IsDir()
}

// Returns true if and only if a file with filename exists and the file is not a directory
func NonDirExists(filename string) bool {
	file_info, err := os.Stat(filename)
	return err == nil && !file_info.IsDir()
}

// Returns true if and only if a file of a given pathname exists.
func FileExists(pathname string) bool {
	_, err := os.Stat(pathname)
	return err == nil
}

// Returns true if and only if a file with filename exists and is a named FIFO pipe.
func IsFIFO(filename string) bool {
	file_info, err := os.Stat(filename)
	return err == nil && (file_info.Mode() & os.ModeNamedPipe) != 0;
}

// Returns true if and only if a file with filename exists and is a device.
func IsDevice(filename string) bool {
	file_info, err := os.Stat(filename)
	return err == nil && (file_info.Mode() & os.ModeDevice) != 0;
}

// Returns true if and only if a file with filename exists and is a device.
func IsSymlink(filename string) bool {
	file_info, err := os.Stat(filename)
	return err == nil && (file_info.Mode() & os.ModeSymlink) != 0;
}

// Returns true if and only if a file with filename exists and is a temporary file.
func IsTemporary(filename string) bool {
	file_info, err := os.Stat(filename)
	return err == nil && (file_info.Mode() & os.ModeTemporary) != 0;
}

// Writes a command server upstart script.
func WriteCommandServerConfiguration(user string, home_dir string, rootfs string) error {
	return WriteCommandServerConfigurationWithResourceLimits(user, home_dir, rootfs, nil, nil)
}

// Writes a command server upstart script.
func WriteCommandServerConfigurationWithResourceLimits(user string, home_dir string, rootfs st
	ring, soft *ResourceLimits, hard *ResourceLimits) error {
	cmd := exec.Command("/bin/bash", "-s")
	in, _ := cmd.StdinPipe()
	resource_args := ResourceLimitsToIexecArgString(soft, hard)
	fmt.Fprintf(in,
		‘buddy_user="%s"
		buddy_home_dir="%s"
		if [ -z "${buddy_user}" ]; then
		   buddy_user="error"
		fi
		rootfs="%s"
		cat <<EOF > "${rootfs}/etc/init/cmdbuddy-${buddy_user}.conf"
		start on local-filesystems
		console output
		author "Webifide Employee"
		description "This job runs the cmdbuddy-daemon for ${buddy_user}"
		# The setuid/setgid features are brand new (November 2011) so they are
		# not supported in the Ubuntu 11.10 or lower.
		#
		# setuid ${buddy_user}
		# setgid ${buddy_user}
		expect fork
		respawn
		respawn limit 10 5
		chdir ${buddy_home_dir}
		script
		su -l -c "iexec -o ${buddy_home_dir}/.cmd.out -e ${buddy_home_dir}/.cmd.err -- cmd-server ${buddy_home_dir}/.cmd" ${buddy_user}
		#iexec %s -u ${buddy_user} -o ${buddy_home_dir}/.cmd.out -e ${buddy_home_dir}/.cmd.err -- cmd-server ${buddy_home_dir}/.cmd
		end script
		EOF
‘, user, home_dir, rootfs, resource_args);
	in.Close()
	return cmd.Run()
}

// Configures IP tables for this root filesystem. (uses Brad’s config files)
//
// @param rootfs The full pathname to a root filesystem for an OS.
func ConfigureIPTables(rootfs string) error {
	err := ioutil.WriteFile(path.Join(rootfs, "/root/iptables.conf"),
		[]byte(‘*filter
			:INPUT ACCEPT [0:0]
			:FORWARD ACCEPT [0:0]
			:OUTPUT ACCEPT [0:0]
			-A INPUT -p tcp -m tcp --dport 8080 -j ACCEPT
			COMMIT
			*nat
			:PREROUTING ACCEPT [5:1083]
			:INPUT ACCEPT [1:32]
			:OUTPUT ACCEPT [0:0]
			:POSTROUTING ACCEPT [0:0]
			COMMIT
			‘), 500)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(rootfs, "/etc/rc.local"),
		[]byte(‘#!/bin/sh -e
			/sbin/iptables-restore < /root/iptables.conf
			exit 0
			‘), 500)
	return err
}

// Returns false iff a file exists and at least one byte can be read
// from it without any I/O errors.
//
// @param fn The filename of the file to check for emptiness.
func IsEmptyFile(fn string) bool {
	file, err := os.OpenFile(fn, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return true
	}
	defer file.Close()
	buffer := make([]byte, 1)
	n, err := file.Read(buffer)
	if err != nil {
		return true
	}
	return n == 0
}
