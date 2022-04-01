package quickbuddy
/// File: limits.go
/// Purpose: Provides utilities for limiting resources.
/// Author: Damian Eads
import (
"fmt"
// "io"
// "io/ioutil"
// "os"
// "os/exec"
// "path"
// "strings"
// "syscall"
)

// Unused by the OS, this constant indicates a rlimit resource should not be
// explicitly changed.
const RLIMIT_UNCHANGED = -2;
// Encapsulates resource limits for a command.
type ResourceLimits struct {
	cpu int;
	fsize int;
	data int;
	stack int;
	core int;
	rss int;
	nofile int;
	as int;
	nproc int;
	memproc int;
	locks int;
	sigpending int;
	msgqueue int;
	nice int;
	rtprio int;
};

// Ugly function that takes in soft and hard resource limits and constructs an argument
// string for iexec.
func ResourceLimitsToIexecArgString(soft *ResourceLimits, hard *ResourceLimits) string {
	buffer := make([]byte, 1024);
	if soft != nil {
		if (soft.cpu != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-cpu-soft %d", soft.cpu))
			buffer = append(buffer, x...)
		}
		if (soft.fsize != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-fsize-soft %d", soft.fsize))
			buffer = append(buffer, x...)
		}
		if (soft.data != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-data-soft %d", soft.data))
			buffer = append(buffer, x...)
		}
		if (soft.stack != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-stack-soft %d", soft.stack))
			buffer = append(buffer, x...)
		}
		if (soft.core != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-core-soft %d", soft.core))
			buffer = append(buffer, x...)
		}
		if (soft.rss != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-rss-soft %d", soft.rss))
			buffer = append(buffer, x...)
		}
		if (soft.nofile != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-nofile-soft %d", soft.nofile))
			buffer = append(buffer, x...)
		}
		if (soft.as != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-as-soft %d", soft.as))
			buffer = append(buffer, x...)
		}
		if (soft.nproc != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-nproc-soft %d", soft.nproc))
			buffer = append(buffer, x...)
		}
		if (soft.memproc != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-memproc-soft %d", soft.memproc))
			buffer = append(buffer, x...)
		}
		if (soft.locks != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-locks-soft %d", soft.locks))
			buffer = append(buffer, x...)
		}
		if (soft.sigpending != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-sigpending-soft %d", soft.sigpending))
			buffer = append(buffer, x...)
		}
		if (soft.msgqueue != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-msgqueue-soft %d", soft.msgqueue))
			buffer = append(buffer, x...)
		}
		if (soft.nice != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-nice-soft %d", soft.nice))
			buffer = append(buffer, x...)
		}
		if (soft.rtprio != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-rtprio-soft %d", soft.rtprio))
			buffer = append(buffer, x...)
		}
	}
	if hard != nil {
		if (hard.cpu != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-cpu-hard %d", hard.cpu))
			buffer = append(buffer, x...)
		}
		if (hard.fsize != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-fsize-hard %d", hard.fsize))
			buffer = append(buffer, x...)
		}
		if (hard.data != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-data-hard %d", hard.data))
			buffer = append(buffer, x...)
		}
		if (hard.stack != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-stack-hard %d", hard.stack))
			buffer = append(buffer, x...)
		}
		if (hard.core != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-core-hard %d", hard.core))
			buffer = append(buffer, x...)
		}
		if (hard.rss != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-rss-hard %d", hard.rss))
			buffer = append(buffer, x...)
		}
		if (hard.nofile != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-nofile-hard %d", hard.nofile))
			buffer = append(buffer, x...)
		}
		if (hard.as != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-as-hard %d", hard.as))
			buffer = append(buffer, x...)
		}
		if (hard.nproc != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-nproc-hard %d", hard.nproc))
			buffer = append(buffer, x...)
		}
		if (hard.memproc != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-memproc-hard %d", hard.memproc))
			buffer = append(buffer, x...)
		}
		if (hard.locks != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-locks-hard %d", hard.locks))
			buffer = append(buffer, x...)
		}
		if (hard.sigpending != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-sigpending-hard %d", hard.sigpending))
			buffer = append(buffer, x...)
		}
		if (hard.msgqueue != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-msgqueue-hard %d", hard.msgqueue
			))
			buffer = append(buffer, x...)
		}
		if (hard.nice != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-nice-hard %d", hard.nice))
			buffer = append(buffer, x...)
		}
		if (hard.rtprio != RLIMIT_UNCHANGED) {
			var x = []byte(fmt.Sprintf(" --rlimit-rtprio-hard %d", hard.rtprio))
			buffer = append(buffer, x...)
		}
	}
	return string(buffer);
}

func NewResourceLimits() *ResourceLimits {
	return &ResourceLimits{RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED, RLIMIT_UNCHANGED};
}
