/// File: cgroups.go
/// Purpose: Manipulates configurations for cgroups
/// Author: Damian Eads
package quickbuddy
import (
// "bytes"
	"errors"
	"fmt"
)

// Stores options for the container constructor to construct a new
// container object.
type CgroupInfo map[string][]string;
// Returns a list of valid LXC configuration key strings
func GetCGroupKeys() []string {
	return []string{ "lxc.utsname",
		"lxc.tty",
		"lxc.pts",
		"lxc.devttydir",
		"lxc.rootfs",
		"lxc.rootfs.mount",
		"lxc.mount",
		"lxc.arch",
		"lxc.cap.drop",
		"lxc.pivotdir",
		"lxc.network.type",
		"lxc.network.flags",
		"lxc.network.name",
		"lxc.network.link",
		"lxc.network.macvlan.mode",
		"lxc.network.hwaddr",
		"lxc.network.ipv4",
		"lxc.network.ipv4.gateway",
		"lxc.network.ipv6",
		"lxc.network.ipv6.gateway",
		"lxc.network.vlan.id",
		"lxc.network.mtu",
		"lxc.network.script.up",
		"lxc.network.veth.pair",
		"lxc.cgroup.devices.deny",
		"lxc.cgroup.devices.allow",
		"lxc.cgroup.cpu.shares",
		"lxc.cgroup.memory.force_empty",
		"lxc.cgroup.memory.limit_in_bytes",
		"lxc.cgroup.memory.memsw.limit_in_bytes",
		"lxc.cgroup.memory.move_charge_at_immigrate",
		"lxc.cgroup.memory.oom_control",
		"lxc.cgroup.memory.soft_limit_in_bytes",
		"lxc.cgroup.memory.swappiness",
		"lxc.cgroup.memory.usage_in_bytes",
		"lxc.cgroup.memory.use_hierarchy",
		"lxc.cgroup.cpuset.cpu_exclusive",
		"lxc.cgroup.cpuset.cpus",
		"lxc.cgroup.cpuset.mem_exclusive",
		"lxc.cgroup.cpuset.mem_hardwall",
		"lxc.cgroup.cpuset.memory_migrate",
		"lxc.cgroup.cpuset.memory_spread_page",
		"lxc.cgroup.cpuset.memory_spread_slab",
		"lxc.cgroup.cpuset.mems",
		"lxc.cgroup.cpuset.sched_load_balance",
		"lxc.cgroup.cpuset.sched_relax_domain_level",
		"lxc.cgroup.blkio.reset_stats",
		"lxc.cgroup.blkio.sectors",
		"lxc.cgroup.blkio.throttle.read_bps_device",
		"lxc.cgroup.blkio.throttle.read_iops_device",
		"lxc.cgroup.blkio.throttle.write_bps_device",
		"lxc.cgroup.blkio.throttle.write_iops_device",
		"lxc.cgroup.blkio.weight",
		"lxc.cgroup.blkio.weight_device",
	};
}

// Returns a set (map of strings to bools) of LXC configuration keys.
func GetCGroupKeySet() map[string]bool {
	var _, keyset = GetCGroupKeyArrayAndSet();
	return keyset;
}

// Returns a set (map of strings to bools) of LXC configuration keys.
func GetCGroupKeyArrayAndSet() ([]string, map[string]bool) {
	var keyset = make(map[string]bool);
	var keys = GetCGroupKeys();
	for _, key := range keys {
		keyset[key] = true;
	}
	return keys, keyset;
}

// Returns a byte array representation of an LXC configuration. Each
// line has the form key=value
func GetCgroupInfoBytes(info CgroupInfo) ([]byte, error) {
	var keys, keyset = GetCGroupKeyArrayAndSet()
	for key, _ := range info {
		if !keyset[key] {
			return nil, errors.New(fmt.Sprintf("invalid lxc configuration key: %s, valid keys are %s", key, keys))
		}
	}
	var buffer = make([]byte, 0);
	for _, key := range keys {
		var value_list = info[key];
		if value_list != nil {
			for _, value := range value_list {
				line := []byte(fmt.Sprintf("%s = %s\n", key, value))
				buffer = append(buffer, line...);
			}
		}
	}
	return buffer, nil
}

// Returns a pre-populated map of a default LXC configuration given
// a container name, root filesystem, and fstab
func GetDefaultCgroupInfo(container_name string, rootfs string, fstab string) CgroupInfo {
	var info = CgroupInfo {
		"lxc.utsname": {container_name},
			"lxc.tty": {"4"},
			"lxc.pts": {"1024"},
			"lxc.rootfs": {rootfs},
			"lxc.mount": {fstab},
			"lxc.arch": {"amd64"},
			"lxc.cap.drop": {"sys_module"},
			"lxc.network.type": {"veth"},
			"lxc.network.flags": {"up"},
			"lxc.network.name": {"eth0"},
			"lxc.network.link": {"br0"},
			"lxc.network.ipv4": {"0.0.0.0"},
			"lxc.cgroup.devices.deny": {"a"},
			"lxc.cgroup.devices.allow": {"c *:* m", "b *:* m", "c 1:3 rwm", "c 1:5 rwm", "c 5:1 rwm",
			"c 5:0 rwm", "c 4:0 rwm", "c 4:1 rwm", "c 1:9 rwm", "c 1:8 rwm", "c 136:* rwm",
			"c 5:2 rwm", "c 254:0 rwm", "c 10:229 rwm", "c 10:200 rwm"},
		}
	return info;
}

