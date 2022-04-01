/// File: qb.go
/// Purpose: Implements the quickbuddy command-line interface (CLI)
/// called qb for short.
/// Author: Damian Eads
package main
import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	. "quickbuddy"
	"os"
	"strings"
)

// Stores the required number of arguments for each of the qb command
// line interface commands.
var required_cmd_nargs = map[string] int{
	"execute-bclient": -2, //requires *at least* 2 arguments
	"execute-client": -2, //requires *at least* 2 arguments
	"execute-tty": -2, //requires *at least* 2 arguments
	"execute-server": 1, //requires exactly 1 argument
	"execute": -3, //requires cname user cmd [args]
	"bexecute": -3, //requires cname user cmd [args]
	"start": 1,
	"s": 1,
	"stop": 1,
	"st": 1,
	"bstart": 1,
	"bs": 1,
	"create": 2,
	"c": 2,
	"mount": 1,
	"m": 1,
	"remount": 1,
	"help": 0,
	"--help": 0,
	"-h": 0,
	"include": 1,
	"umount": 1,
	"unmount": 1,
	"u": 1,
	"destroy": 1,
	"delete": 1,
	"d": 1,
	"create-image-set": 1,
	"copy-image-set": 2,
	"delete-image-set": 1,
	"trim-image-set": 1,
}

// Test container creation and mounting with Aufs using N threads
// that create 25 containers each.
func AsynchronousMain(nthreads int) {
	channel := make(chan int)
	for i := 0; i < nthreads; i++ {
		go func(i int) {
			for j := 0; j < 25; j++ {
				k := i*25+j
				var container *Container = NewContainerFromDefaultCache(fmt.Sprintf("go%d", k), "/web");
				container.Create()
			}
			channel <- 1
		}(i)
	}
	<- channel
}

// Test container creation and mounting with Aufs using 1 thread that
// creates 100 containers.
func SerialMain() {
	for j := 0; j < 100; j++ {
		var container *Container = NewContainerFromDefaultCache(fmt.Sprintf("go%d", j)
			, "/web");
		container.Create()
	}
}

// Displays some very basic help to standard output.
func Help() {
	fmt.Printf(
`qb manages containers and image sets using AUFS filesystems
 usage: qb [command] [command-args]
		
                       * container commands *

  create/c cname [iname] Prepares a new container named ’cname’
                         from the image set named ’iname’ (default).
  destroy/d cname        Destroys the container named ’cname’.
  mount/m cname          Mounts the container named ’cname’.
  remount cname          Remounts the container named ’cname’.
  unmount/u cname        Unmounts the container named ’cname’.
  start/s cname          Starts the container named ’cname’ without
                         daemonizing. Log-in prompt will appear.
  bstart/bs cname        Starts the container named ’cname’
		         as a deamon and blocks until ready.
  dstart/s cname         Starts the container ’cname’ as a daemon
                         without blocking.
  stop/st cname          Stops the container.
  passwd/pw cname uid pw Changes password to ’pw’ for a uid on
                         container ’cname’.

                   * command server/client *

  execute cname user cmd args    Run command ’cmd’ in ’cname’ as ’user’.
  execute-client FILE cmd args   Run command ’cmd’ using FIFO pipe ’FILE’.
  execute-server FILE            Run command server using FIFO pipe ’FILE’.

                      * image set commands *

  create-image-set iname      Create an OS image set named ’iname’.
  delete-image-set iname      Delete the OS image set named ’iname’.
  trim-image-set iname        Removes unnecessary services from a container.
  copy-image-set src dest     Copy image set named ’src’ into image set ’dest’.
`)
}

// Implements the ’include’ qb CLI command.
func CommandIncludeFile(filename string) error {
	// Try opening the file for reading.
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	// Using a BufferedReader so we can read lines instead of bytes.
	bufReader := bufio.NewReader(reader)
	// Create a buffer.
	buffer := bytes.NewBuffer(make([]byte, 0))
	// Keep track of line numbers.
	line_no := 0
	// A place to store errors from commands. IO errors from reading
	// the include file will be stored in err.
	var command_err error = nil
	// The verbose flag is false by default.
	verbose := false
	// Fail on the first command failure
	fail := false
	// While there is stuff left to read...
	for line, prefix, err := bufReader.ReadLine(); err != io.EOF; line, prefix, err = bufR
	eader.ReadLine() {
		// Append the new bytes to the buffer.
		buffer.Write(line)
		// If we’ve captured the entire line
		if (!prefix) {
			// Increment the line number.
			line_no++
			// Copy the buffer into a string object.
			command_line := buffer.String()
			// Allow for a comment character.
			comment_index := strings.Index(command_line, "#")
			if comment_index != -1 {
				command_line = command_line[:comment_index]
			}
			// Trim whitespace (if any) once we’ve removed the comment (if any).
			command_line = strings.TrimSpace(command_line)
			// Now split apart the command into tokens using spaces as
			// the delimiter.
			tokens := strings.Split(command_line, " ")
			// Grab the number of tokens.
			ntokens := len(tokens)
			// Print the command out to the screen if verbose is set.
			if (verbose) {
				fmt.Println(command_line)
			}
			// If there is exactly one token, invoke with an empty
			// argument array.
			if ntokens == 1 {
				if tokens[0] == "verbose" {
					verbose = true;
				} else if tokens[0] == "quiet" {
					verbose = false;
				} else if tokens[0] == "fail" {
					fail = true;
				} else if tokens[0] == "nofail" {
					fail = false;
				} else {
					command_err = ProcessInput(tokens[0], []string{});
				}
			} else if ntokens > 1 { // Otherwise, pass the rest of the tokens
				// as command arguments.
				command_err = ProcessInput(tokens[0], tokens[1:]);
			} // Otherwise it’s all whitespace so skip the command.
			if command_err != nil {
				fmt.Fprintf(os.Stderr, "qb(include): error in file %s line %d (%s): %s\n", filename, line_no, tokens[0], command_err)
				if (fail) {
					os.Exit(1)
				}
			}
			command_err = nil
			// Clear the buffer.
			buffer.Reset()
		}
	}
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

// Implements the ’start’ CLI command.
func CommandStartContainer(cname string) error {
	container, err := NewContainerFromImageSetMeta(cname, "/web")
	if err != nil {
		return err
	}
	err = container.Start();
	return err
}

// Implements the ’bstart’ CLI command.
func CommandBlockedStartContainer(cname string) error {
	container, err := NewContainerFromImageSetMeta(cname, "/web")
	if err != nil {
		return err
	}
	err = container.BlockedStart();
	return err
}

// Implements the ’create’ CLI command.
func CommandCreateContainer(cname string, iname string) error {
	image_set := NewImageSet(iname, "/isx")
	container := NewContainerFromImageSet(cname, "/web", image_set)
	err := container.Create();
	return err
}

// Implements the ’stop’ CLI command.
func CommandStopContainer(cname string) error {
	container, err := NewContainerFromImageSetMeta(cname, "/web")
	if err != nil {
		return err
	}
	err = container.Stop();
	return err
}

// Implements the ’mount’ CLI command.
func CommandMountContainer(cname string) error {
	container, err := NewContainerFromImageSetMeta(cname, "/web")
	if err != nil {
		return err
	}
	err2 := container.Mount();
	return err2
}

// Implements the ’remount’ CLI command.
func CommandRemountContainer(cname string) error {
	container, err := NewContainerFromImageSetMeta(cname, "/web")
	if err != nil {
		return err
	}
	err2 := container.Remount();
	return err2
}

// Implements the ’unmount’ CLI command.
func CommandUnmountContainer(cname string) error {
	container, err := NewContainerFromImageSetMeta(cname, "/web")
	if err != nil {
		return err
	}
	err2 := container.Unmount();
	return err2
}

// Implements the ’delete’ CLI command.
func CommandDeleteContainer(cname string) error {
	container, err := NewContainerFromImageSetMeta(cname, "/web")
	if err != nil {
		return err
	}
	err2 := container.Delete();
	return err2
}

// Implements the ’create-image-set’ CLI command.
func CommandCreateImageSet(iname string) error {
	image_set := NewImageSet(iname, "/isx")
	return image_set.CreateDefault()
}

// Implements the ’create-image-set’ CLI command.
func CommandTrimImageSet(iname string) error {
	image_set := NewImageSet(iname, "/isx")
	return image_set.Trim()
}

// Implements the ’delete-image-set’ CLI command.
func CommandDeleteImageSet(iname string) error {
	image_set := NewImageSet(iname, "/isx")
	return image_set.Delete()
}

// Implements the ’copy-image-set’ CLI command.
func CommandCopyImageSet(src_iname string, dest_iname string) error {
	src_image_set := NewImageSet(src_iname, "/isx")
	dest_image_set := NewImageSet(dest_iname, "/isx")
	return dest_image_set.Copy(src_image_set)
}

// Implements the ’execute’ CLI command.
func CommandExecuteInContainer(cname string, user string, args []string) error {
	container, err := NewContainerFromImageSetMeta(cname, "/web")
	if err != nil {
		return err
	}
	return container.Execute(user, args)
}

// Implements the ’bexecute’ CLI command.
func CommandExecuteInContainerBlocked(cname string, user string, args []string) error {
	container, err := NewContainerFromImageSetMeta(cname, "/web")
	if err != nil {
		return err
	}
	result, err := container.ExecuteBlocked(user, args)
	if result != nil {
		OutputResult(result)
	}
	return err
}

// Implements the ’execute-client’ CLI command.
func CommandExecuteClient(args []string) error {
	command := NewFIFOCommand(args[0], "")
	command.Verbose = false
	return command.ExecuteDaemonOnServer(args[1:])
}

// Writes a FIFOCommandResult to standard output
func OutputResult(result *FIFOCommandResult) {
	fmt.Printf("---standard output---\n")
	fmt.Printf("%s", string(result.Out))
	fmt.Printf("---standard error---\n")
	fmt.Printf("%s", string(result.Err))
	fmt.Printf("---other info---\n")
	fmt.Printf("pid %d\n", result.Pid)
	fmt.Printf("signals %s\n", result.SignalCodes)
	fmt.Printf("exit %d\n", result.ExitCode)
}

// Implements the ’execute-client-blocked’ CLI command.
func CommandExecuteClientBlocked(args []string) error {
	command := NewFIFOCommand(args[0], "")
	command.Verbose = false
	result, err := command.ExecuteDaemonOnServerBlockOnClient(args[1:])
	if result != nil {
		OutputResult(result)
	}
	return err
}

// Implements the ’execute-tty’ CLI command.
func CommandExecuteClientTTY(args []string) error {
	command := NewFIFOCommand(args[0], "")
	command.ClientTTYShare = true
	command.Verbose = false
	return command.ExecuteDaemonOnServer(args[1:])
}

// Implements the ’execute-server’ CLI command.
func CommandExecuteServer(args []string) error {
	command := NewFIFOCommand(args[0], "")
	command.Verbose = false
	return command.RunServer()
}

// Executes a qb command-line interface command.
//
// @param command The qb-CLI command (e.g. create) to call.
// @param args The qb-CLI command’s arguments.
func ProcessInput(command string, args []string) error {
	// For each command and command alias, store exactly the number
	// of arguments to expect. In the future, we can implement
	// optional arguments.
	actual_nargs := len(args)
	required_nargs, present := required_cmd_nargs[command]
	var err error = nil
	required_nargs, present = required_cmd_nargs[command]
	// If the command exists in our map, check that the number of
	// arguments to it is correct.
	if present {
		// If the required_nargs is negative, then it is interpreted
		// as "at least".
		if required_nargs < 0 {
			atleast_nargs := int(math.Abs(float64(required_nargs)))
			if actual_nargs < atleast_nargs {
				return errors.New(fmt.Sprintf("command ’%s’ requires at least
%d argument(s) (%d given)", command, atleast_nargs, actual_nargs))
			}
			// If the required_nargs is nonnegative, then it is interpreted
			// as "equal".
		} else {
			if actual_nargs != required_nargs {
				return errors.New(fmt.Sprintf("command ’%s’ requires %d argume
nt(s) (%d given)", command, required_nargs, actual_nargs))
			}
		}
	} else {
		// Otherwise, return an error that the command does not exist.
		return errors.New(fmt.Sprintf("invalid command ’%s’", command))
	}
	// Now that the command exists and has the right number of arguments,
	// call a function that processes that command.
	switch (command) {
	case "start", "s":
		err = CommandStartContainer(args[0])
	case "bstart", "bs":
		err = CommandBlockedStartContainer(args[0])
	case "create", "c":
		err = CommandCreateContainer(args[0], args[1])
	case "stop", "st":
		err = CommandStopContainer(args[0])
	case "destroy", "delete", "d":
		err = CommandDeleteContainer(args[0])
	case "execute":
		err = CommandExecuteInContainer(args[0], args[1], args[2:])
	case "bexecute":
		err = CommandExecuteInContainerBlocked(args[0], args[1], args[2:])
	case "execute-client":
		err = CommandExecuteClient(args)
	case "execute-bclient":
		err = CommandExecuteClientBlocked(args)
	case "execute-tty":
		err = CommandExecuteClientTTY(args)
	case "execute-server":
		err = CommandExecuteServer(args)
	case "mount", "m":
		err = CommandMountContainer(args[0])
	case "unmount", "umount", "u":
		err = CommandUnmountContainer(args[0])
	case "remount":
		err = CommandRemountContainer(args[0])
	case "include":
		err = CommandIncludeFile(args[0])
	case "create-image-set":
		err = CommandCreateImageSet(args[0])
	case "copy-image-set":
		err = CommandCopyImageSet(args[0], args[1])
	case "trim-image-set":
		err = CommandTrimImageSet(args[0])
	case "delete-image-set":
		err = CommandDeleteImageSet(args[0])
	case "help", "--help", "-h":
		Help()
		os.Exit(0)
	}
	return err
}

// This function is the command line entry point to the Quickbuddy program.
func main() {
	nargs := len(os.Args)
	var err error = nil
	if nargs > 2 {
		err = ProcessInput(os.Args[1], os.Args[2:])
	} else if nargs == 2 {
		err = ProcessInput(os.Args[1], []string{})
	} else {
		Help()
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "qb(%s) error: %s\n", os.Args[1], err)
		os.Exit(1)
	}
	os.Exit(0)
}
