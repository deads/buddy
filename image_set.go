// File: image_set.go
// Purpose: Creates, manipulates, and deletes AUFS-based OS image sets for
// quickbuddy. Image sets are used to store a base configuration
// and OS to build containers from.
// Author: Damian Eads
package quickbuddy

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

// The default pathname where the LXC cache will be stored.
const DEFAULT_LXC_CACHE_PATH string = "/var/cache/lxc/oneiric/rootfs-amd64"
const DEFAULT_LXC_VAR_PATH string = "/usr/local/var/lib/lxc"

type ImageSet struct {
	name string; /* A name for the image set, which may contain letters, nu
mbers, dashes, or underscores. */
	idir string; /* The directory where the image set will live. */
	rootfs string; /* The image set OS’s root directory. */
	rootfs_archive_path string; /* The path of the image set’s archive of the rootfs. */
}

// Returns a new image set object, which represents an OS root filesystem,
// configuration, and meta-data to use to create containers.
//
// @param image_set_name The name of the image set as an alphanumeric string.
// @param image_sets_path The path to the image sets (e.g. "/isx")

func NewImageSet(image_set_name string, image_sets_path string) *ImageSet {
	var image_set_dir = path.Join(image_sets_path, image_set_name)
	return &ImageSet{image_set_name,
		image_set_dir,
		path.Join(image_set_dir, "rootfs"),
		path.Join(image_set_dir, "rootfs.tar.gz"),
	}
}

// Create image set and its meta-data from the default cache, which is
// determined by the DEFAULT_LXC_CACHE_PATH constant.
func (this *ImageSet) CreateDefault() error {
	return this.Create(DEFAULT_LXC_CACHE_PATH)
}

// Create image set and its meta-data from an OS cache, which is
// determined by the DEFAULT_LXC_CACHE_PATH constant. Upon
// successful completion, all necessary files and configuration
// will be created to build containers off the image set.
func (this *ImageSet) Create(lxc_cache_path string) error {
// Check if the image set already exists.
	if this.IsCreated() {
		return errors.New("The image set ’" + this.name + "’ already exists - cannot proceed.")
	}
	if !DirExists(lxc_cache_path) {
		return errors.New("The cache path directory ’" + this.name + "’ does not exist - cannot proceed.")
	}
	err_mkdir := os.Mkdir(this.idir, 755)
	if err_mkdir != nil {
		return err_mkdir
	}
	cmd := exec.Command("cp", "-a", lxc_cache_path, this.rootfs)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "stdout+stderr> %s", out);
		return err
	}
	err2 := WriteCommandServerConfiguration("web", "/home/web", this.rootfs)
	if err2 != nil {
		return err2
	}
	err3 := WriteCommandServerConfiguration("root", "/root", this.rootfs)
	if err3 != nil {
		return err3
	}
	err4 := os.Mkdir(path.Join(this.rootfs, "/var/lib/dhcp3"), 755)
	if err4 != nil {
		return err4
	}
	err5 := ioutil.WriteFile(path.Join(this.rootfs, "/etc/network/interfaces"),
		[]byte(
			‘auto lo
			iface lo inet loopback
			auto eth0
			iface eth0 inet dhcp
			‘), 644)
	if err5 != nil {
		return err5
	}
	err6 := this.ConfigureIPTables();
	return err6
}

// Copy the files and configuration of one image set into an non-existing
// image set defined by the target object (this).
//
// @param src The image set to copy from.
func (this *ImageSet) Copy(src *ImageSet) error {
	if this.IsCreated() {
		return errors.New("The destination image set ’" + this.name + "’ already exists")
	}
	if !src.IsCreated() {
		return errors.New("The source image set ’" + this.name + "’ does not exist - copy cannot proceed.")
	}
	if src.name == this.name {
		return errors.New("The source and destination image sets have the same name - copy cannot proceed.")
	}
	if src.idir == this.idir {
		return errors.New("The source and destination image sets use the same directory - copy cannot proceed.")
}
	cmd := exec.Command("cp", "-a", src.idir, this.idir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "stdout+stderr> %s", out);
		return err
	}
	return nil
}

// Deletes the files comprising image set.
//
// Note: this does not actually delete the target object containing
// information about the image set.
//
// FIXME/WARNING: This function does not check if containers are using
// the image set to be deleted.
func (this *ImageSet) Delete() error {
	if !this.IsCreated() {
		return errors.New("The image set to delete ’" + this.name + "’ does not exist.")
	}
	cmd := exec.Command("rm", "-rf", this.idir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "stdout+stderr> %s", out);
		return err
	}
	return nil
}

// Returns true iff the image set has been created on the filesystem.
func (this *ImageSet) IsCreated() bool {
	return DirExists(this.idir)
}

// Configures IP tables for this image set. (uses Brad’s config files)
func (this *ImageSet) ConfigureIPTables() error {
	return ConfigureIPTables(this.rootfs)
}

// Trims the image set to remove unnecessary upstart services. This
// speeds start-up time of containers. Note: apt-get is not disabled
// in the new quickbuddy system.
func (this *ImageSet) Trim() error {
	if !this.IsCreated() {
		return errors.New("The image set ’" + this.name + "’ does not exist - cannot proceed.")
	}
	//FINDOUT: I don’t really know what this does.
	err := ioutil.WriteFile(path.Join(this.rootfs, "/etc/init/lxc.conf"),
		[]byte(
			`# fake some events needed for correct startup other services
			description "Container Upstart"
			start on startup
			script
			    rm -rf /var/run/*.pid
                            rm -rf /var/run/network/*
                            /sbin/initctl emit stopped JOB=udevtrigger --no-wait
                            /sbin/initctl emit started JOB=udev --no-wait
                        end script
`), 544)
	
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(this.rootfs, "/etc/init/ssh.conf"),
		[]byte(
			`# ssh - OpenBSD Secure Shell server
			 #
			 # The OpenSSH server provides secure shell access to the system.
                         description "OpenSSH server"
                         start on filesystem
                         stop on runlevel [!2345]
                         expect fork
                         respawn
                         respawn limit 10 5
                         umask 022
                         # replaces SSHD_OOM_ADJUST in /etc/default/ssh
                         oom never
                         pre-start script
                           test -x /usr/sbin/sshd || { stop; exit 0; }
                           test -e /etc/ssh/sshd_not_to_be_run && { stop; exit 0; }
                           test -c /dev/null || { stop; exit 0; }
                           mkdir -p -m0755 /var/run/sshd
                         end script
                         # if you used to set SSHD_OPTS in /etc/default/ssh, you can change the
                         # ’exec’ line here instead
                         exec /usr/sbin/sshd
                         `), 544)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(this.rootfs, "/etc/init/console.conf"),
		[]byte(
			`# console - getty
			#
			# This service maintains a console on tty1 from the point the system is
                        # started until it is shut down again.
                        start on stopped rc RUNLEVEL=[2345]
                        stop on runlevel [!2345]
                        respawn
                        exec /sbin/getty -8 38400 /dev/console
                        `), 544)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(this.rootfs, "/lib/init/fstab"),
		[]byte(
			`
			# /lib/init/fstab: cleared out for bare-bones lxc
			`), 544)
	if err != nil {
		return err
	}
	// Remove unnecessary ttys.
	tty5_filename := path.Join(this.rootfs, "/etc/init/tty5.conf")
	if FileExists(tty5_filename) {
		tty5_err := os.Remove(tty5_filename)
		if tty5_err != nil {
			fmt.Fprintf(os.Stderr, "warning: unable to remove %s\n", tty5_filename
			)
		}
	}
	tty6_filename := path.Join(this.rootfs, "/etc/init/tty6.conf")
	if FileExists(tty6_filename) {
		tty6_err := os.Remove(tty6_filename)
		if tty6_err != nil {
			fmt.Fprintf(os.Stderr, "warning: unable to remove %s\n", tty6_filename
			)
		}
	}
	udev_filename := path.Join(this.rootfs, "/etc/udev/udev.conf")
	err = ReplaceAllInFile(udev_filename, ‘="err"‘, "=0")
	if err != nil {
		return err
	}
	cmd := exec.Command("chroot", this.rootfs, "/bin/bash", "-s")
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	fmt.Fprintf(in,
		`
                if [ -z "$LANG" ]; then
		   locale-gen en_US.UTF-8
		   update-locale LANG=en_US.UTF-8
		else
                   locale-gen $LANG
                   update-locale LANG=$LANG
                fi
                /usr/sbin/update-rc.d -f ondemand remove
                cd /etc/init
                for filename in u*.conf tty[2-9].conf plymouth*.conf hwclock*.conf module*.conf; do
                   if [ -f "${filename}" ]; then
                      echo Removing unnecessary service "${filename}"
                      mv -- "${filename}" "${filename}.orig"
                   else
                      echo Filename does not exist - "${filename}"
                   fi
                done
                `)
	in.Close()
	var bout bytes.Buffer
	var berr bytes.Buffer
	cmd.Stdout = &bout
	cmd.Stderr = &berr
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cmd> %s\n", cmd)
		fmt.Fprintf(os.Stderr, "stdout> %s\n", bout.String())
		fmt.Fprintf(os.Stderr, "stderr> %s\n", berr.String())
	}
	return err
}
