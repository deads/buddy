/// File: commands.go
/// Purpose: Implements a named FIFO pipe-based command client and server.
/// Author: Damian Eads
package quickbuddy
import (
	"bytes"
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
)

/* A callback interface for retrieving the standard error, exit status,
and signal status of a FIFOCommand process once it terminates. */
type FIFOCallback interface {
	ShouldStoreStandardOutput() bool;
	ShouldStoreStandardError() bool;
	ShouldBlockUntilTerminated() bool;
	HandleTerminate(result *FIFOCommandResult);
	HandleError(err error);
}

/* An internal callack for implementing FIFOCommand execution with
blocking support on the client. */
type SimpleFIFOCallback struct {
	
	/* A place to store the result of a FIFO command. */
	store *FIFOCommandResult;
	
	/* Whether to block or not until terminatation. */
	Block bool
	
	/* Whether to store away standard output. */
	StoreStdout bool;
	
	/* Whether to store away standard error. */
	StoreStderr bool;
};

/* Returns true iff the standard output should be stored */
func (this SimpleFIFOCallback) ShouldStoreStandardOutput() bool {
	return this.StoreStdout;
}

/* Returns true iff the standard error should be stored */
func (this SimpleFIFOCallback) ShouldStoreStandardError() bool {
	return this.StoreStderr;
}

/* Returns true iff the client thread (not the command server in
   the container) should block until the command is terminated in
   the container.*/
func (this SimpleFIFOCallback) ShouldBlockUntilTerminated() bool {
	return this.Block;
}

/* Copies the appropriate data in the store. */
func (this SimpleFIFOCallback) HandleTerminate(result *FIFOCommandResult) {
	this.store.Out = result.Out
	this.store.Err = result.Err
	this.store.SignalCodes = result.SignalCodes
	this.store.ExitCode = result.ExitCode
	this.store.Pid = result.Pid
}

func (this SimpleFIFOCallback) HandleError(err error) {
	fmt.Fprintf(os.Stderr, "generic HandleError error: %s\n", err)
}

func NewSimpleFIFOCallback(result *FIFOCommandResult, block bool, store_stdout bool, store_stderr bool) *SimpleFIFOCallback {
	return &SimpleFIFOCallback{result, block, store_stdout, store_stderr}
}

/** Stores the standard output, the standard error, signal codes, and
exit code. */
type FIFOCommandResult struct {
	/** The standard output. */
	Out []byte;

	/** The standard error. */
	Err []byte;
	
	/** The signals received. */
	SignalCodes []int;
	
	/** The exit code (-1 if kill signaled). */
	ExitCode int;
	
	/** The pid of the command. */
	Pid int;
};

/** Encapsulates information for FIFO command servers and clients. */
type FIFOCommand struct {
	/** The name of the FIFO filename to create. */
	Filename string;
	
	/** When true, output is verbose. */
	Verbose bool;
	
	/** Whether the server should block and wait if it is unable to immediately
	  acquire the lock. */
	ServerNonBlocking bool;
	
	/** Whether the client should block and wait if it is unable to immediately acquire the lock. */
	ClientNonBlocking bool;
	
	/** Whether the client’s TTY filename should be used as the output
	  and error streams of the command to execute. (Experimental) */
	ClientTTYShare bool;

	/** The owner of the FIFO pipe. */
	Uid int;
	Gid int;
	
	/** The client filename prefix. */
	Rootfs string;
}

/// Creates a new FIFO command server/client for a root user. It
/// doesn’t actually start the server - it just holds relevant
/// information.
func NewFIFOCommand(filename string, client_prefix string) *FIFOCommand {
	return &FIFOCommand{
		filename,
		true,
		true,
		false,
		false, 0, 0, client_prefix}
}

/// Creates a new FIFO command server/client for a specific user. It
/// doesn’t actually start the server - it just holds relevant
/// information.
func NewFIFOCommandForUser(filename string, client_prefix string, uid, gid int) *FIFOCommand {
	return &FIFOCommand{
		filename,
		true,
		true,
		false,
		false, uid, gid, client_prefix}
}

/// CLIENT OR SERVER: Creates the FIFO Pipe.
func (this *FIFOCommand) Create() error {
	err := MkFIFO(this.Filename);
	if err != nil {
		return err
	}
	return os.Chown(this.Filename, this.Uid, this.Gid)
}

/// CLIENT OR SERVER: Checks if the FIFO Pipe exists.
func (this *FIFOCommand) Exists() bool {
	return FileExists(this.Filename) && IsFIFO(this.Filename)
}

/// CLIENT OR SERVER: Checks if the filename exists but don’t verify that it’s a FIFO pipe
func (this *FIFOCommand) FileExists() bool {
	return FileExists(this.Filename)
}

/// CLIENT OR SERVER: Flushes the named pipe.
func (this *FIFOCommand) Flush() error {
	return FlushFIFO(this.Filename)
}

/// CLIENT: Requests that the command server exit.
func (this *FIFOCommand) RequestServerShutdown() error {
	return this.ExecuteDaemonOnServerWithCallback([]string{"@exit",}, nil)
}

/// CLIENT: Requests that the command server restart.
func (this *FIFOCommand) RequestServerRestart() error {
	return this.ExecuteDaemonOnServerWithCallback([]string{"@restart",}, nil)
}

/// CLIENT: Waits until the command server is alive.
func (this *FIFOCommand) WaitUntilServerIsAlive() error {
	return this.ExecuteDaemonOnServerWithCallback([]string{"@alive",}, nil)
}

/// CLIENT: Executes a command as a daemon on the server without waiting.
func (this *FIFOCommand) ExecuteDaemonOnServer(args []string) error {
	err := this.ExecuteDaemonOnServerWithCallback(args, nil)
	return err
}

/// CLIENT: Executes a command as a daemon on the client but waits
/// (on the client) until the command is terminated through
/// exit or kill.
func (this *FIFOCommand) ExecuteDaemonOnServerBlockOnClient(args []string) (*FIFOCommandResult, error) {
	var result = &FIFOCommandResult{nil, nil, nil, 0, 0};
	var callback = NewSimpleFIFOCallback(result, true, true, true)
	var err = this.ExecuteDaemonOnServerWithCallback(args, *callback)
	return result, err
}

/// CLIENT: Calls a new command on the server as a new daemon process.
///
/// @param args The command and its arguments.
/// @param block Whether to block until the command finishes or is terminated.
func (this *FIFOCommand) ExecuteDaemonOnServerWithCallback(args []string, callback FIFOCallback) error {
	var tmp_dir string = "";
	var server_tmp_dir string = "";
	var status_fn string = "";
	var stdout_fn string = "";
	var stderr_fn string = "";
	if (callback != nil) {
		var err_dir error = nil
		tmp_dir, err_dir = ioutil.TempDir(path.Join(this.Rootfs, "/tmp"), "iexec");
		if err_dir != nil {
			return err_dir
		}
		server_tmp_dir = path.Join("/tmp", path.Base(tmp_dir))
		os.Chown(tmp_dir, this.Uid, this.Gid)
		status_fn = path.Join(tmp_dir, "status")
		err_mkfifo := MkFIFO(status_fn)
		if err_mkfifo != nil {
			return err_mkfifo
		}
		os.Chown(status_fn, this.Uid, this.Gid)
		if (callback.ShouldStoreStandardOutput()) {
			stdout_fn = path.Join(tmp_dir, "stdout")
		}
		if (callback.ShouldStoreStandardError()) {
			stderr_fn = path.Join(tmp_dir, "stderr")
		}
	}
	// If the command file exists, make sure its a FIFO pipe. If it’s not,
	// return an error.
	if FileExists(this.Filename) {
		if !IsFIFO(this.Filename) {
			return errors.New(fmt.Sprintf("filename ’%s’ exists but is not a named FIFO pipe - cannot proceed.", this.Filename))
		}
	} else { // Otherwise, we have no way of communicating with the server
		// so do not proceed.
		return errors.New(fmt.Sprintf("filename ’%s’ does not exist - cannot proceed.", this.Filename))
	}
	// This is the point where we try to acquire the lock.
try_again:
	file, err := os.OpenFile(this.Filename, syscall.O_WRONLY, 660)
	if err != nil {
		return err
	}
	if this.Verbose { fmt.Printf("command-lock: waiting\n") }
	// Try to acquire an exclusive lock on the lock file. If
	// ClientNonBlocking is set, do not wait.
	if this.ClientNonBlocking {
		err = syscall.Flock(file.Fd(), syscall.LOCK_EX | syscall.LOCK_NB)
	} else {
		err = syscall.Flock(file.Fd(), syscall.LOCK_EX)
	}
	// If we weren’t able to acquire the lock, try to diagnose what
	// went wrong.
	if err != nil {
		switch (err) {
		case syscall.EWOULDBLOCK:
			file.Close()
			if this.ClientNonBlocking {
				return errors.New("command lock already held by another proces
s")
			} else {
				// Try to acquire the lock again. Technically, we shouldn’t
				// get here because we would have omitted LOCK_NB
				goto try_again
			}
		case syscall.EBADF:
			return errors.New("strange: got EBADF")
		case syscall.EINTR:
			file.Close()
			return errors.New("interrupted")
		case syscall.ENOLCK:
			return errors.New("out of kernel memory")
		default:
			return err
		}
	}
	// If we’ve got to this point, we’ve acquired the lock.
	if this.Verbose { fmt.Printf("command-lock: acquired\n") }
	if (args[0] != "@restart" && args[0] != "@exit" && args[0] != "@alive") {
		_, err = fmt.Fprintf(file, "iexec\n")
		if callback != nil {
			_, err = fmt.Fprintf(file, "-s\n")
			_, err = fmt.Fprintf(file, "%s\n", path.Join(server_tmp_dir, "status")
			)
			if (stdout_fn != "") {
				_, err = fmt.Fprintf(file, "-o\n")
				_, err = fmt.Fprintf(file, "%s\n", path.Join(server_tmp_dir, "
stdout"))
			}
			if (stderr_fn != "") {
				_, err = fmt.Fprintf(file, "-e\n")
				_, err = fmt.Fprintf(file, "%s\n", path.Join(server_tmp_dir, "
stderr"))
			}
			fmt.Fprintf(file, "--\n")
		}
		// if (this.Limits.hard_nproc != RLIMIT_UNCHANGED) {
		// _, err = fmt.Fprintf(file, "--rlimit-hard-nproc\n%d\n", this.Limits.hard_nproc)
		// }
		// if (this.Limits.soft_nproc != RLIMIT_UNCHANGED) {
		// _, err = fmt.Fprintf(file, "--rlimit-soft-nproc\n%d\n", this.Limits.soft_nproc)
		// }
		// if (this.Limits.hard_nofile != RLIMIT_UNCHANGED) {
		// _, err = fmt.Fprintf(file, "--rlimit-hard-nofile\n%d\n", this.Limits.hard_nofile)
		// }
		// if (this.Limits.soft_nofile != RLIMIT_UNCHANGED) {
		// _, err = fmt.Fprintf(file, "--rlimit-soft-nofile\n%d\n", this.Limits.soft_nofile)
		// }
	}
	if err != nil {
		return err;
	}
	// Write out the command and the arguments to the command file pipe.
	for i := 0; i < len(args); i++ {
		_, err = fmt.Fprintf(file, "%s\n", args[i])
		if err != nil {
			_, err = fmt.Fprintf(file, "-*-EOFENDEOFEND-*-\n")
			return err;
		}
	}
	// Write end-of-command sentinel.
	_, err = fmt.Fprintf(file, "-*-EOFENDEOFEND-*-\n")
	// Unlock the lock file and close.
	err = syscall.Flock(file.Fd(), syscall.LOCK_UN)
	file.Close()
	if this.Verbose { fmt.Printf("command-lock: released\n") }
	// If the callback is defined, then call it
	if callback != nil {
		// Invoke callback processing function in a blocking fashion
		if callback.ShouldBlockUntilTerminated() {
			process_err := WaitUntilDone(callback, status_fn, stdout_fn, stderr_fn
			)
			return process_err
		} else {
			// Invoke callback processing function in a non-blocking fashion
			go func() {
				process_err := WaitUntilDone(callback, status_fn, stdout_fn, stderr_fn)
				if process_err != nil {
					callback.HandleError(err)
				}
			}()
		}
	}
	return nil
}

/// CLIENT (internal): Waits for a command to complete by scanning a status file, which should
/// be a named FIFO pipe. It stops when "kill", "exit", or "err" tokens are encountered in
/// the fifo pipe
func WaitUntilDone(callback FIFOCallback, status_fn string, stdout_fn string, stderr_fn string) error {
	if FileExists(status_fn) {
		if !IsFIFO(status_fn) {
			return errors.New(fmt.Sprintf("status filename ’%s’ exists but is not a named FIFO pipe - cannot proceed.", status_fn))
		}
	} else { // Otherwise, create it and try this loop again.
		return errors.New(fmt.Sprintf("status filename ’%s’ does not exist - cannot proceed.", status_fn))
	}
	terminated := false
	signal_codes := make([]int, 0)
	exit_code := -1
	pid := 0
	err_occurred := false
	/// While a termination code has not be received.
	for ; !terminated; {
		file, err := os.OpenFile(status_fn, syscall.O_RDONLY, 660)
		defer file.Close()
		if err != nil {
			return err
		}
		// Use a BufferedReader so we can read lines instead of bytes.
		bufReader := bufio.NewReader(file)
		// While there is stuff left to read...
		for line, prefix, err := bufReader.ReadLine(); err != io.EOF; line, prefix, er
		r = bufReader.ReadLine() {
			sline := string(line)
			// If an error occured while reading, stop reading.
			if err != nil {
				return err
			}
			// If an entire line has been read.
			if !prefix {
				// If the line starts with kill then the process has been sign
				aled and terminated.
					// Parse the signal code.
					if strings.HasPrefix(sline, "kill ") {
					terminated = true
					vals := strings.Split(sline, " ")
					if len(vals) > 1 {
						code, _ := strconv.ParseInt(vals[1], 10, 64)
						signal_codes = append(signal_codes, int(code))
					}
				} else if strings.HasPrefix(sline, "exit ") {
					// If the line starts with exit then the process has been signaled and terminated.
					// Parse the exit code.
					terminated = true
					vals := strings.Split(sline, " ")
					if len(vals) > 1 {
						code, _ := strconv.ParseInt(vals[1], 10, 64)
						exit_code = int(code)
					}
				} else if strings.HasPrefix(sline, "pid ") {
					// If the line starts with pid then store away the pid
					terminated = true
					vals := strings.Split(sline, " ")
					if len(vals) > 1 {
						code, _ := strconv.ParseInt(vals[1], 10, 64)
						pid = int(code)
					}
				} else if strings.HasPrefix(sline, "stop ") || strings.HasPrefix(sline, "cont ") {
					// If the line starts with stop or cont then store the
					signal code
					vals := strings.Split(sline, " ")
					if len(vals) > 1 {
						code, _ := strconv.ParseInt(vals[1], 10, 64)
						signal_codes = append(signal_codes, int(code))
					}
				} else if strings.HasPrefix(sline, "err") {
					// If the line starts with err then an error occured d
					uring waitpid
					err_occurred = true
					terminated = true
				}
			}
		}
	}
	stdout := make([]byte, 0)
	if stdout_fn != "" {
		stdout, _ = ioutil.ReadFile(stdout_fn)
	}
	stderr := make([]byte, 0)
	if stderr_fn != "" {
		stderr, _ = ioutil.ReadFile(stderr_fn)
	}
	result := &FIFOCommandResult{stdout, stderr, signal_codes, exit_code, pid}
	// If an error occurred during waitpid, return it
	if err_occurred {
		return errors.New(fmt.Sprintf("error while waiting for status, status_fn=%s", status_fn))
	} else { // Otherwise, call the callback.
		callback.HandleTerminate(result)
	}
	tmp_dir := path.Dir(status_fn)
	rmerr := os.Remove(status_fn)
	if (stdout_fn != "") {
		rmerr = os.Remove(stdout_fn)
	}
	if (stderr_fn != "") {
		rmerr = os.Remove(stderr_fn)
	}
	rmerr = os.Remove(tmp_dir)
	return rmerr
}

/// Runs a new command server using a filename as a FIFO pipe.
///
/// @param filename The filename of the new named FIFO pipe.
/// @param return_if_locked If set to true, returns immediately if lock
/// could not be immediately acquired.
func (this *FIFOCommand) RunServer() error {
try_again:
	lock_file, lock_err := os.OpenFile(this.Filename + "˜", syscall.O_WRONLY | syscall.O_CREAT | syscall.O_TRUNC, 660)
	defer lock_file.Close()
	if lock_err != nil {
		return errors.New(fmt.Sprintf("server: cannot open daemon-lock file ’%s’", this.Filename))
	}
	if this.Verbose { fmt.Printf("daemon-lock: waiting\n") }
	if this.ServerNonBlocking {
		lock_err = syscall.Flock(lock_file.Fd(), syscall.LOCK_EX | syscall.LOCK_NB)
	} else {
		lock_err = syscall.Flock(lock_file.Fd(), syscall.LOCK_EX)
	}
	if lock_err != nil {
		switch (lock_err) {
		case syscall.EWOULDBLOCK:
			if this.ServerNonBlocking {
				return errors.New("daemon lock already held by another process")
			} else {
				lock_file.Close()
				goto try_again
			}
		case syscall.EBADF:
			return errors.New("strange: got EBADF")
		case syscall.EINTR:
			return errors.New("interrupted")
		case syscall.ENOLCK:
			return errors.New("out of kernel memory")
		}
	}
	if this.Verbose { fmt.Printf("daemon-lock: acquired\n") }
	// A place to store the command file and error objects.
	var err error = nil
	for true {
		// Check that the file is a FIFO pipe.
		if FileExists(this.Filename) {
			if !IsFIFO(this.Filename) {
				return errors.New(fmt.Sprintf("status filename ’%s’ exists but is not a named FIFO pipe - cannot proceed.", this.Filename))
			}
		} else { // Otherwise, create it and try this loop again.
			MkFIFO(this.Filename)
		}
		// If there was an error opening, exit out of the daemon.
		if err != nil {
			return err
		}
		// Try opening the command file for reading.
		file, err := os.OpenFile(this.Filename, syscall.O_RDONLY, 660)
		if err != nil {
			return err
		}
		// Use a BufferedReader so we can read lines instead of bytes.
		bufReader := bufio.NewReader(file)
		// Create a buffer.
		buffer := bytes.NewBuffer(make([]byte, 0))
		// An array to store command line arguments.
		// The first dimension are commands sent to the server.
		// The second dimension is the arguments.
		//
		// so args[i][j] is the j’th argument of the i’th command
		var arg_lists [][]string;
		arg_lists = append(arg_lists, []string{})
		args := &arg_lists[0]
		// While there is stuff left to read...
		for line, prefix, err := bufReader.ReadLine(); err != io.EOF; line, prefix, er
		r = bufReader.ReadLine() {
			buffer.Write(line)
			var sline string
			// If we’ve captured the entire line...
			if (!prefix) {
				sline = buffer.String()
				fmt.Println("sline", sline)
				buffer.Reset()
				if sline == "-*-EOFENDEOFEND-*-" {
					arg_lists = append(arg_lists, []string{})
					args = &arg_lists[len(arg_lists)-1]
				} else {
					*args = append(*args, sline)
				}
			}
		}
		file.Close()
		for k := 0; k < len(arg_lists); k++ {
			// If the command is ’@exit’ then it’s special, unlock and exit.
			if len(arg_lists[k]) >= 1 {
				switch (arg_lists[k][0]) {
				case "@exit":
					if this.Verbose { fmt.Printf("daemon-lock: unlocking\n
") }
					err = syscall.Flock(lock_file.Fd(), syscall.LOCK_UN)
					//lock_file.Close()
					return err
				case "@restart":
					if this.Verbose { fmt.Printf("daemon-lock: unlocking\n
") }
					err = syscall.Flock(lock_file.Fd(), syscall.LOCK_UN)
					lock_file.Close()
					goto try_again
				case "@alive":
					continue;
				}
			}
			// Now let’s build the command object.
			var cmd *exec.Cmd = nil
			if (len(arg_lists[k]) >= 2) {
				cmd = exec.Command(arg_lists[k][0], arg_lists[k][1:]...)
			} else if (len(arg_lists[k]) >= 1) {
				cmd = exec.Command(arg_lists[k][0], []string{}...)
			}
			// The command object will be non-nil when the command file just
			// parsed contains at least one line.
			if (cmd != nil) {
				fmt.Fprintf(os.Stdout, "cmd> %s\n", cmd)
				var bout bytes.Buffer
				var berr bytes.Buffer
				cmd.Stdout = &bout
				cmd.Stderr = &berr
				run_err := cmd.Run()
				fmt.Fprintf(os.Stdout, "stdout> %s\n", bout.String())
				fmt.Fprintf(os.Stdout, "stderr> %s\n", berr.String())
				if run_err != nil {
					return run_err
				}
			}
		}
	}
	return err
}
