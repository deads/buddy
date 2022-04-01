/// File: container.go
/// Purpose: Creates, manipulates, and deletes AUFS-based LXC containers.
/// Author: Damian Eads
package quickbuddy
import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"syscall"
)

// Encapsulates information about a container such as pathnames. Performs
// creation, manipulation, mounting, unmounting, and deletion of containers.
type Container struct {

	/* The name of the container. */
	name string;

	/* The root directory of the container.*/
	cdir string;
	
	/* The meta-data directory of the container.*/
	meta_dir string;
	
	/* The root filesystem of the container. */
	rootfs string;
	
	/* The directory to store data during a Copy-on-Write (CoW). */
	private_dir string;
	
	/* The pathname of the LXC configuration. */
	config_pathname string;
	
	/* The pathname of the fstab for the container. */
	fstab_pathname string;
	
	/* The image set object of this container. */
	image_set *ImageSet;
	
	/* The cgroup configuration of this container. A default map
	   is provided.*/
	Cgroup_info CgroupInfo;
	
	/* The soft resource limits for the container.
	   nil by default. When both hard and soft are nil, existing
	   upstart script for the command server is unchanged. */
	Soft_limits *ResourceLimits;
	
	/* The hard resource limits for the container.
	   nil by default. */
	Hard_limits *ResourceLimits;
}

// Creates a new container object from the default cache.
//
// @param container_name The name of the container to create.
// @param containers_path The path of the containers (e.g. "/web").
func NewContainerFromDefaultCache(container_name string, containers_path string) *Container {

	var container_dir = path.Join(containers_path, container_name)
	var rootfs = path.Join(container_dir, "rootfs")
	var fstab = path.Join(container_dir, "fstab")
	return &Container{container_name,
		container_dir,
		path.Join(container_dir, "meta"),
		rootfs,
		path.Join(container_dir, "private-data"),
		path.Join(container_dir, "config"),
		fstab,
		nil,
		GetDefaultCgroupInfo(container_name, rootfs, fstab),
		nil,
		nil,
	};
}

// Creates a new container object from an image set.
// Note: this does not actually create a container but an object to hold
// information.
//
// @param container_name The name of the container to create.
// @param containers_path The path of the containers (e.g. "/web").
// @param image_set The image set to create the container from.
func NewContainerFromImageSet(container_name string, containers_path string, image_set *ImageS
et) *Container {

	var container_dir = path.Join(containers_path, container_name)
	var rootfs = path.Join(container_dir, "rootfs")
	var fstab = path.Join(container_dir, "fstab")
	return &Container{container_name,
		container_dir,
		path.Join(container_dir, "meta"),
		path.Join(container_dir, "rootfs"),
		path.Join(container_dir, "private-data"),
		path.Join(container_dir, "config"),
		path.Join(container_dir, "fstab"),
		image_set,
		GetDefaultCgroupInfo(container_name, rootfs, fstab),
		nil,
		nil,
	}
}

// Creates a new container object using meta data from
// /web/<container_name>/meta/image-set-name to acquire the image set name.
// This requires the container to exist on the system.
//
// Note: this does not actually create a container but an object to hold
// information.
//
// @param container_name The name of the container to create.
// @param container_path The path of the containers (e.g. "/web").
// @param image_sets_path The path of the image sets (e.g. "/isx").

func NewContainerFromImageSetMeta(container_name string, containers_path string) (*Container,
error) {

	container_dir := path.Join(containers_path, container_name)
	image_set_name_meta_filename := path.Join(container_dir, "/meta/image-set-name")
	image_set_dir_meta_filename := path.Join(container_dir, "/meta/image-set-dir")
	if !FileExists(image_set_name_meta_filename) {
		return nil, errors.New(fmt.Sprintf("could not ascertain image set for containe
r ’%s’", container_name))
	}
	image_set_name, err := ioutil.ReadFile(image_set_name_meta_filename)
	if err != nil {
		return nil, err
	}
	if !FileExists(image_set_dir_meta_filename) {
		return nil, errors.New(fmt.Sprintf("could not ascertain image set directory fo
r container ’%s’", container_name))
	}
	image_set_dir, err2 := ioutil.ReadFile(image_set_dir_meta_filename)
	if err2 != nil {
		return nil, err
	}
	var image_set *ImageSet = NewImageSet(string(image_set_name), path.Dir(string(image_set_dir)))
	var rootfs = path.Join(container_dir, "rootfs")
	var fstab = path.Join(container_dir, "fstab")
	return &Container{container_name,
		container_dir,
		path.Join(container_dir, "meta"),
		rootfs,
		path.Join(container_dir, "private-data"),
		path.Join(container_dir, "config"),
		fstab,
		image_set,
		GetDefaultCgroupInfo(container_name, rootfs, fstab),
		nil,
		nil,
	}, err
}

// Prepares the files, directories, configurations necessary to run
// a container. The container’s root filesystem is mounted using AUFS.
func (this *Container) Create() error {
	if DirExists(this.cdir) {
		return errors.New("cannot create container: directory ’" + this.cdir + "’ already exists.")
	}
	if this.IsCreated() {
		return errors.New("cannot create container: it already exists!")
	}
	if this.image_set != nil {
		if !this.image_set.IsCreated() {
			return errors.New(fmt.Sprintf("image set %s does not exist - cannot proceed.", this.image_set.name))
		}
	} else {
		if DirExists(DEFAULT_LXC_CACHE_PATH) {
			return errors.New(fmt.Sprintf("directory %s where OS cache is stored does not exist - cannot proceed.", DEFAULT_LXC_CACHE_PATH))
		}
	}
	os.Mkdir(this.cdir, 755)
	os.Mkdir(this.rootfs, 755)
	os.Mkdir(this.meta_dir, 755)
	os.Mkdir(this.private_dir, 755)
	if this.image_set != nil {
		image_set_name_meta_filename := path.Join(this.meta_dir, "/image-set-name")
		image_set_dir_meta_filename := path.Join(this.meta_dir, "/image-set-dir")
		ioutil.WriteFile(image_set_name_meta_filename, []byte(this.image_set.name), 44
			4)
		ioutil.WriteFile(image_set_dir_meta_filename, []byte(this.image_set.idir), 444
		)
	}
	if err := this.Mount(); err != nil {
		return err
	} else {
		this.WriteConfig()
		this.WriteFstab()
		this.WriteNetworkConfiguration()
	}
	return nil
}

/// Returns the home directory of a user.
func (this *Container) GetHomeDirectory(user string) string {
	if user == "root" {
		return "/root"
	}
	return "/home/" + user
}

/// Starts the container and returns an error if the start was unsuccessful.
func (this *Container) Start() error {
	return this.AdvancedStart(false)
}

/// Starts the container and blocks until the command server is
/// ready. It returns an error if the start was unsuccessful.
func (this *Container) BlockedStart() error {
	return this.AdvancedStart(true)
}

/// Starts the container.
///
/// @param blocked_start Whether to block until the container’s command servers are ready for requests.
func (this *Container) AdvancedStart(blocked_start bool) error {
	if !this.IsCreated() {
		return errors.New("container " + this.name + " has not yet been created")
	}
	if !this.IsMounted() {
		return errors.New("container " + this.name + " is not mounted")
	}
	if this.IsRunning() {
		return errors.New("container " + this.name + " is already running - cannot start")
	}
	root_fifo := NewFIFOCommandForUser(path.Join(this.rootfs, "/root/.cmd"), this.rootfs, 0, 0)
	web_fifo := NewFIFOCommandForUser(path.Join(this.rootfs, "/home/web/.cmd"), this.rootfs, 1000, 1000)
	if root_fifo.FileExists() {
		os.Remove(root_fifo.Filename)
	}
	root_fifo.Create()
	if web_fifo.FileExists() {
		os.Remove(web_fifo.Filename)
	}
	web_fifo.Create()
	cmd := exec.Command("lxc-start", "-n", this.name, "-d", "-f", path.Join(this.cdir, "config"))
	var bout bytes.Buffer
	var berr bytes.Buffer
	cmd.Stdout = &bout
	cmd.Stderr = &berr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "stdout> %s\n", bout.String())
		fmt.Fprintf(os.Stderr, "stderr> %s\n", berr.String())
		return err
	}
	if blocked_start {
		root_err := root_fifo.WaitUntilServerIsAlive()
		if root_err != nil {
			return root_err
		}
		web_err := web_fifo.WaitUntilServerIsAlive()
		if web_err != nil {
			return web_err
		}
	}
	if err == nil {
		fmt.Fprintf(os.Stderr, "starting %s was successful!\n", this.name)
	} else {
		fmt.Fprintf(os.Stderr, "starting %s was not successful.\n", this.name)
	}
	return err
}

/// Stops the container.
func (this *Container) Stop() error {
	cmd := exec.Command("lxc-stop", "-n", this.name)
	var bout bytes.Buffer
	var berr bytes.Buffer
	cmd.Stdout = &bout
	cmd.Stderr = &berr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "stdout> %s\n", bout.String())
		fmt.Fprintf(os.Stderr, "stderr> %s\n", berr.String())
		return err
	} else {
		fmt.Fprintf(os.Stderr, "stopping %s was successful!\n", this.name)
	}
	fmt.Fprintf(os.Stderr, "remounting %s\n", this.name)
	this.Remount()
	return nil
}

// Delete the files and meta-data for this container.
//
// Note: this does not actually delete the target object containing
// information about the container.
//
// FIXME/WARNING: This function does not check if containers are using
// the image set to be deleted.
func (this *Container) Delete() error {
	// First verify that the container exists and is not mounted. If these
	// conditions aren’t true, return an error immediately.
	if !this.IsCreated() {
		return errors.New("The container to delete ’" + this.name + "’ does not exist.
")
	}
	if this.IsMounted() {
		err := this.Unmount()
		if err != nil {
			return err
		}
		//return errors.New("The container to delete ’" + this.name + "’ is mounted. Unmount first.")
	}
	// First, delete the configuration from LXC’s registry.
	rm_lxc_reg_err := os.RemoveAll(path.Join(DEFAULT_LXC_VAR_PATH, this.name))
	if rm_lxc_reg_err != nil {
		return rm_lxc_reg_err
	}
	
	// Second, remove the rootfs directory. This is a secondary test whether
	// the container is mounted. If it is mounted then either a file exists
	// in the directory .
	rmdir_rootfs_err := os.Remove(this.rootfs)
	if rmdir_rootfs_err != nil {
		return rmdir_rootfs_err
	}
	
	// If the non-recursive remove was successful, it is then safe to
	// remove the container directory and any files that may be in it
	// (there shouldn’t be any excess files)
	rm_cdir_err := os.RemoveAll(this.cdir)
	if rm_cdir_err != nil {
		return rm_cdir_err
	}
	return nil
}

// Mounts the container’s root filesystem.
//
// FIXME: update /etc/mtab like the command line ’mount’
func (this *Container) Mount() error {
	var err error
	if this.IsMounted() {
		return errors.New("cannot mount container ’" + this.name +
			"’: directory ’" + this.rootfs + "’ is already mounted.")
	}
	if this.image_set == nil {
		err = MountAufsCoW(DEFAULT_LXC_CACHE_PATH, this.private_dir, this.rootfs)
	} else {
		err = MountAufsCoW(this.image_set.rootfs, this.private_dir, this.rootfs)
	}
	return err
}

// Mounts the container’s root filesystem as read-write. This must be called
// after lxc-stop and before a subsequent lxc-start.
//
// If the container is unmounted, it will automatically be mounted
// using Mount().
//
// FIXME: update /etc/mtab like the command line ’mount’
func (this *Container) Remount() error {
	var err error
	if this.IsMounted() {
		err = RemountAufsCoWReadWrite(this.image_set.rootfs, this.private_dir, this.rootfs)
	} else {
		err = this.Mount()
	}
	return err
}

// Unmount the resources used by this container.
//
// FIXME: update /etc/mtab like the command line ’mount’
func (this *Container) Unmount() error {
	return syscall.Unmount(this.rootfs, syscall.MNT_DETACH)
}

// Write this container’s LXC configuration to the file <cdir>/config and
// <prefix>/var/lib/lxc/cname/config
func (this *Container) WriteConfig() error {
	var configuration_bytes, err = GetCgroupInfoBytes(this.Cgroup_info)
	if err != nil {
		return err
	}	
	err = ioutil.WriteFile(this.config_pathname, configuration_bytes, 0644)
	if err != nil {
		return err
	}
	if !DirExists(DEFAULT_LXC_VAR_PATH) {
		err = os.MkdirAll(DEFAULT_LXC_VAR_PATH, 755)
		if err != nil {
			return err;
		}
		container_cache_dir := path.Join(DEFAULT_LXC_VAR_PATH, this.name)
		err = os.MkdirAll(container_cache_dir, 755)
		if err != nil {
			return err;
		}
		err = ioutil.WriteFile(path.Join(container_cache_dir, "config"), configuration
			_bytes, 0644)
	}
	if this.Hard_limits != nil || this.Soft_limits != nil {
		err2 := WriteCommandServerConfigurationWithResourceLimits("web", "/home/web",
			this.rootfs, this.Soft_limits, this.Hard_limits)
		if err2 != nil {
			return err2
		}
	}
	return err
}

// Write this container’s LXC configuration to the file <cdir>/config.
func (this *Container) WriteConfigOld() error {
	configuration_str := fmt.Sprintf(
		‘lxc.utsname = %s
		lxc.tty = 4
		lxc.pts = 1024
		lxc.rootfs = %s
		lxc.mount = %s
		lxc.arch = %s
		lxc.cap.drop = sys_module
		lxc.network.type = veth
		lxc.network.flags = up
		lxc.network.name = eth0
		lxc.network.link = br0
		lxc.network.ipv4 = 0.0.0.0
		lxc.cgroup.devices.deny = a
		# Allow any mknod (but not using the node)
		lxc.cgroup.devices.allow = c *:* m
		lxc.cgroup.devices.allow = b *:* m
		# /dev/null and zero
		lxc.cgroup.devices.allow = c 1:3 rwm
		lxc.cgroup.devices.allow = c 1:5 rwm
		# consoles
		lxc.cgroup.devices.allow = c 5:1 rwm
		lxc.cgroup.devices.allow = c 5:0 rwm
		#lxc.cgroup.devices.allow = c 4:0 rwm
		#lxc.cgroup.devices.allow = c 4:1 rwm
		# /dev/{,u}random
		lxc.cgroup.devices.allow = c 1:9 rwm
		lxc.cgroup.devices.allow = c 1:8 rwm
		lxc.cgroup.devices.allow = c 136:* rwm
		lxc.cgroup.devices.allow = c 5:2 rwm
		# rtc
		lxc.cgroup.devices.allow = c 254:0 rwm
		#fuse
		lxc.cgroup.devices.allow = c 10:229 rwm
		#tun
		lxc.cgroup.devices.allow = c 10:200 rwm
		‘, this.name, this.rootfs, this.fstab_pathname, "amd64")
	err := ioutil.WriteFile(this.config_pathname, []byte(configuration_str), 0644)
	// Now copy the config file to lxc’s var/lib directory so that
	// the container is "registered" with LXC and can be started with
	// lxc-start or lxc-block-start
	if err == nil {
		if !DirExists(DEFAULT_LXC_VAR_PATH) {
			os.MkdirAll(DEFAULT_LXC_VAR_PATH, 755)
		}
		container_cache_dir := path.Join(DEFAULT_LXC_VAR_PATH, this.name)
		os.Mkdir(container_cache_dir, 755)
		err = ioutil.WriteFile(path.Join(container_cache_dir, "config"), []byte(configuration_str), 0644)
	}
	return err
}

// Write this container’s LXC configuration to the file <cdir>/fstab.
func (this *Container) WriteFstab() error {
	err := ioutil.WriteFile(this.fstab_pathname, []byte(
		fmt.Sprintf(
			‘proc %s/proc proc nodev,noexec,nosuid 0 0
			sysfs %s/sys sysfs defaults 0 0
			‘,
			this.rootfs, this.rootfs)), 644)
	return err
}

// Write this container’s network configuration files, which include:
// - /etc/hostname
// - /etc/hosts
// - /etc/dhcp/dhclient.conf (or) /etc/dhcp3/dhclient.conf
func (this *Container) WriteNetworkConfiguration() error {
	err1 := ioutil.WriteFile(path.Join(this.rootfs, "/etc/hostname"),
		[]byte(this.name), 644)
	if err1 != nil {
		return err1
	}
	err2 := ioutil.WriteFile(path.Join(this.rootfs, "/etc/hosts"),
		[]byte(fmt.Sprintf("127.0.0.1 localhost %s", this.name)), 644)
	if err2 != nil {
		return err2
	}
	var err3 error = nil
	dhclient_path1 := path.Join(this.rootfs, "/etc/dhcp/dhclient.conf")
	dhclient_path2 := path.Join(this.rootfs, "/etc/dhcp3/dhclient.conf")
	if (syscall.Access(dhclient_path1, syscall.F_OK) == nil) {
		err3 = ReplaceAllInFile(dhclient_path1, "<hostname>", this.name)
		/**cmd := exec.Command("sed", "-i",
		  fmt.Sprintf("s/<hostname>/%s/", this.name),
		  dhclient_path1)
		  cmd.CombinedOutput()*/
	} else if (syscall.Access(dhclient_path2, syscall.F_OK) == nil){
		err3 = ReplaceAllInFile(dhclient_path1, "<hostname>", this.name)
		/**cmd := exec.Command("sed", "-i",
		  fmt.Sprintf("s/<hostname>/%s/", this.name),
		  dhclient_path2)
		  cmd.CombinedOutput()*/
	} else {
		err3 = errors.New("neither /etc/dhcp/dhclient.conf nor /etc/dhcp3/dhclient.conf exist - dhclient unlikely to work")
	}
	return err3
}

// Executes a command in the container as a daemon.
//
// @param user The username of the new process.
// @param args The command and arguments.
//
func (this *Container) Execute(user string, args []string) error {
	_, err := this.ExecuteImpl(user, args, false)
	return err
}

// Executes a command in the container as a daemon. The calling
// thread blocks until the command is terminated or a fatal
// error occurs. Under the hood, the calling thread is used
// to monitor a status file until an exit status is reported
// by (a second parent in a double fork()) iexec.
//
// @param user The username of the new process.
// @param args The command and arguments.
//

func (this *Container) ExecuteBlocked(user string, args []string) (*FIFOCommandResult, error)
{
	return this.ExecuteImpl(user, args, true)
}

func (this *Container) ExecuteImpl(user string, args []string, blocked bool) (*FIFOCommandResu
	lt, error) {
	home_dir := this.GetHomeDirectory(user)
	home_dir_on_host := path.Join(this.rootfs, home_dir)
	// If the home directory does not exist
	if !DirExists(home_dir_on_host) {
		return nil, errors.New(fmt.Sprintf("User %s home directory %s on container %s does not exist, full path %s", user, home_dir, this.name, home_dir_on_host))
	}
	cmd_file := path.Join(home_dir_on_host, ".cmd")
	var fifo_command *FIFOCommand = nil
	if user == "root" {
		fifo_command = NewFIFOCommandForUser(cmd_file, this.rootfs, 0, 0)
	} else if user == "web" {
		fifo_command = NewFIFOCommandForUser(cmd_file, this.rootfs, 1000, 1000)
	} else {
		errors.New("users other than web or root are not supported.")
	}
	var cmd_err error = nil
	var result *FIFOCommandResult = nil
	if blocked {
		result, cmd_err = fifo_command.ExecuteDaemonOnServerBlockOnClient(args)
	} else {
		cmd_err = fifo_command.ExecuteDaemonOnServer(args)
	}
	return result, cmd_err
}

// Returns true iff the target container is mounted.
//
// Note: This is a hack and is not a general solution for checking
// if a container is mounted.
func (this *Container) IsMounted() bool {
	return DirExists(path.Join(this.rootfs, "/etc"))
}

// Returns true iff the target container is running.
//
// Note: This is a hack and is not a general solution for checking
// if a container is mounted.
func (this *Container) IsRunning() bool {
	fn := path.Join("/cgroup/lxc", this.name)
	return DirExists(fn)
	//fn := path.Join("/cgroup/lxc", this.name, "tasks")
	//return FileExists(fn) && !IsEmptyFile(fn)
}

// Returns true iff the target container has been created, ie all of its
// necessary files and directories have been created.
func (this *Container) IsCreated() bool {
	return DirExists(this.rootfs) &&
		DirExists(this.meta_dir) &&
		DirExists(this.private_dir) &&
		FileExists(this.config_pathname) &&
		FileExists(this.fstab_pathname)
}
