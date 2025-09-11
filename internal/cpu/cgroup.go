package cpu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
)

const cgroupRootDir = "/sys/fs/cgroup"

// cgroup Linux cgroup
type cgroup struct {
	cgroupSet map[string]string
	isV2      bool
}

// CPUCFSQuotaUs cpu.cfs_quota_us
func (c *cgroup) CPUCFSQuotaUs() (int64, error) {
	if c.isV2 {
		data, err := readFile(path.Join(cgroupRootDir, "cpu.max"))
		if err != nil {
			return 0, err
		}
		parts := strings.Fields(data)
		if len(parts) != 2 {
			return 0, errors.New("invalid cpu.max format")
		}
		if parts[0] == "max" {
			return -1, nil
		}
		return strconv.ParseInt(parts[0], 10, 64)
	}

	// cgroup v1
	data, err := readFile(path.Join(c.cgroupSet["cpu"], "cpu.cfs_quota_us"))
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(data, 10, 64)
}

// CPUCFSPeriodUs cpu.cfs_period_us
func (c *cgroup) CPUCFSPeriodUs() (uint64, error) {
	if c.isV2 {
		data, err := readFile(path.Join(cgroupRootDir, "cpu.max"))
		if err != nil {
			return 0, err
		}
		parts := strings.Fields(data)
		if len(parts) != 2 {
			return 0, errors.New("invalid cpu.max format")
		}
		return parseUint(parts[1])
	}

	// cgroup v1
	data, err := readFile(path.Join(c.cgroupSet["cpu"], "cpu.cfs_period_us"))
	if err != nil {
		return 0, err
	}
	return parseUint(data)
}

// CPUAcctUsage cpuacct.usage
func (c *cgroup) CPUAcctUsage() (uint64, error) {
	if c.isV2 {
		data, err := readFile(path.Join(cgroupRootDir, "cpu.stat"))
		if err != nil {
			return 0, err
		}
		var usageUs uint64
		scanner := bufio.NewScanner(strings.NewReader(data))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Fields(line)
			if len(parts) != 2 {
				continue
			}
			if parts[0] == "usage_usec" {
				usageUs, err = parseUint(parts[1])
				if err != nil {
					return 0, err
				}
				return usageUs * 1000, nil // convert to nanoseconds
			}
		}
		if err = scanner.Err(); err != nil {
			return 0, err
		}
		return 0, errors.New("usage_usec not found in cpu.stat")
	}

	data, err := readFile(path.Join(c.cgroupSet["cpuacct"], "cpuacct.usage"))
	if err != nil {
		return 0, err
	}
	return parseUint(data)
}

// CPUAcctUsagePerCPU cpuacct.usage_percpu
func (c *cgroup) CPUAcctUsagePerCPU() ([]uint64, error) {
	if c.isV2 {
		return nil, errors.New("cpuacct.usage_percpu not available in cgroup v2")
	}

	// cgroup v1
	data, err := readFile(path.Join(c.cgroupSet["cpuacct"], "cpuacct.usage_percpu"))
	if err != nil {
		return nil, err
	}
	var usages []uint64
	for _, v := range strings.Fields(data) {
		var u uint64
		if u, err = parseUint(v); err != nil {
			return nil, err
		}
		// fix possible_cpu:https://www.ibm.com/support/knowledgecenter/en/linuxonibm/com.ibm.linux.z.lgdd/lgdd_r_posscpusparm.html
		if u != 0 {
			usages = append(usages, u)
		}
	}
	return usages, nil
}

// CPUSetCPUs cpuset.cpus
func (c *cgroup) CPUSetCPUs() ([]uint64, error) {
	var (
		data string
		err  error
	)
	if c.isV2 {
		data, err = readFile(path.Join(cgroupRootDir, "cpuset.cpus.effective"))
	} else {
		data, err = readFile(path.Join(c.cgroupSet["cpuset"], "cpuset.cpus"))
	}

	if err != nil {
		return nil, err
	}
	cpus, err := ParseUintList(data)
	if err != nil {
		return nil, err
	}
	sets := make([]uint64, 0)
	for k := range cpus {
		sets = append(sets, uint64(k))
	}
	return sets, nil
}

// LogicalCores get logical cores
func (c *cgroup) LogicalCores() (int, error) {
	if c.isV2 {
		sets, err := c.CPUSetCPUs()
		if err != nil {
			return 0, err
		}
		return len(sets), nil
	}

	usages, err := c.CPUAcctUsagePerCPU()
	if err != nil {
		return 0, err
	}
	return len(usages), nil
}

// CPULimits get cpu limits.
// -1 means no limit
func (c *cgroup) CPULimits() (float64, error) {
	if c.isV2 {
		data, err := readFile(path.Join(cgroupRootDir, "cpu.max"))
		if err != nil {
			return 0, err
		}
		parts := strings.Fields(data)
		if len(parts) != 2 {
			return 0, errors.New("invalid cpu.max format")
		}
		if parts[0] == "max" {
			return -1, nil
		}
		quota, err := parseUint(parts[0])
		if err != nil {
			return 0, err
		}
		period, err := parseUint(parts[1])
		if err != nil {
			return 0, err
		}
		if period == 0 {
			return 0, errors.New("cpu.max period is zero")
		}
		return float64(quota) / float64(period), nil
	}

	// cgroup v1
	quota, err := c.CPUCFSQuotaUs()
	if err != nil {
		return 0, err
	}
	if quota <= 0 {
		return -1, nil
	}
	period, err := c.CPUCFSPeriodUs()
	if err != nil {
		return 0, err
	}
	if period == 0 {
		return 0, errors.New("cpu.cfs_period_us is zero")
	}
	return float64(quota) / float64(period), nil
}

// currentcGroup get current process cgroup
func currentcGroup() (*cgroup, error) {
	// Detect if it's cgroup v2
	_, err := os.Stat(path.Join(cgroupRootDir, "cgroup.controllers"))
	if err == nil {
		return &cgroup{isV2: true, cgroupSet: make(map[string]string)}, nil
	}

	pid := os.Getpid()
	cgroupFile := fmt.Sprintf("/proc/%d/cgroup", pid)
	cgroupSet := make(map[string]string)
	fp, err := os.Open(cgroupFile)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	buf := bufio.NewReader(fp)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		col := strings.Split(strings.TrimSpace(line), ":")
		if len(col) != 3 {
			return nil, fmt.Errorf("invalid cgroup format %s", line)
		}
		dir := col[2]
		// When dir is not equal to /, it must be in docker
		if dir != "/" {
			cgroupSet[col[1]] = path.Join(cgroupRootDir, col[1])
			if strings.Contains(col[1], ",") {
				for _, k := range strings.Split(col[1], ",") {
					cgroupSet[k] = path.Join(cgroupRootDir, k)
				}
			}
		} else {
			cgroupSet[col[1]] = path.Join(cgroupRootDir, col[1], col[2])
			if strings.Contains(col[1], ",") {
				for _, k := range strings.Split(col[1], ",") {
					cgroupSet[k] = path.Join(cgroupRootDir, k, col[2])
				}
			}
		}
	}
	return &cgroup{cgroupSet: cgroupSet}, nil
}
