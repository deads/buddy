# buddy
Buddy is a legacy (circa 2012) system for managing containers.

The qb command (quick buddy) manages containers and image sets
using LXC containers and stackable filesystems.

```
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
```
