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

// cgroup interface
type cgroup interface {
	// LogicalCores returns logical cores
	LogicalCores() (int, error)
	// CPULimits returns cpu limits.
	// If no limit is set, return ErrNoCFSLimit
	CPULimits() (float64, error)
	// CPUAcctUsageNs returns accumulated cpu usage in nanoseconds
	CPUAcctUsageNs() (uint64, error)
	// CPUSetCPUs returns the set of CPUs available to the cgroup
	CPUSetCPUs() ([]uint64, error)
}

// cgroupv1 Linux cgroup v1
type cgroupv1 struct {
	cgroupSet map[string]string
	rootDir   string
	readFile  func(string) (string, error)
}

// CPUCFSQuotaUs cpu.cfs_quota_us.
// If no limit is set, return ErrNoCFSLimit
func (c *cgroupv1) CPUCFSQuotaUs() (uint64, error) {
	data, err := c.readFile(path.Join(c.cgroupSet["cpu"], "cpu.cfs_quota_us"))
	if err != nil {
		return 0, err
	}
	quota, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, err
	}
	if quota == -1 {
		return 0, ErrNoCFSLimit
	}
	return uint64(quota), nil
}

// CPUCFSPeriodUs cpu.cfs_period_us
func (c *cgroupv1) CPUCFSPeriodUs() (uint64, error) {
	data, err := c.readFile(path.Join(c.cgroupSet["cpu"], "cpu.cfs_period_us"))
	if err != nil {
		return 0, err
	}
	period, err := parseUint(data)
	if err != nil {
		return 0, err
	}
	if period == 0 {
		return 0, errors.New("cpu.cfs_period_us is zero")
	}
	return period, nil
}

// CPUAcctUsageNs cpuacct.usage
func (c *cgroupv1) CPUAcctUsageNs() (uint64, error) {
	data, err := c.readFile(path.Join(c.cgroupSet["cpuacct"], "cpuacct.usage"))
	if err != nil {
		return 0, err
	}
	return parseUint(data)
}

// CPUAcctUsagePerCPU cpuacct.usage_percpu
func (c *cgroupv1) CPUAcctUsagePerCPU() ([]uint64, error) {
	data, err := c.readFile(path.Join(c.cgroupSet["cpuacct"], "cpuacct.usage_percpu"))
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
func (c *cgroupv1) CPUSetCPUs() ([]uint64, error) {
	data, err := c.readFile(path.Join(c.cgroupSet["cpuset"], "cpuset.cpus"))
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

// LogicalCores returns the number of logical cores.
func (c *cgroupv1) LogicalCores() (int, error) {
	usages, err := c.CPUAcctUsagePerCPU()
	if err != nil {
		return 0, err
	}
	return len(usages), nil
}

// CPULimits return get cpu limits
// If no limit is set, return ErrNoCFSLimit
func (c *cgroupv1) CPULimits() (float64, error) {
	quota, err := c.CPUCFSQuotaUs()
	if err != nil {
		return 0, err
	}
	period, err := c.CPUCFSPeriodUs()
	if err != nil {
		return 0, err
	}
	return float64(quota) / float64(period), nil
}

type cgroupv2 struct {
	rootDir  string
	readFile func(string) (string, error)
}

// CPUAcctUsageNs cpu.stat usage_usec * 1000
func (c *cgroupv2) CPUAcctUsageNs() (uint64, error) {
	data, err := c.readFile(path.Join(c.rootDir, "cpu.stat"))
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

// CPUSetCPUs cpuset.cpus.effective
func (c *cgroupv2) CPUSetCPUs() ([]uint64, error) {
	data, err := c.readFile(path.Join(c.rootDir, "cpuset.cpus.effective"))
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
func (c *cgroupv2) LogicalCores() (int, error) {
	sets, err := c.CPUSetCPUs()
	if err != nil {
		return 0, err
	}
	return len(sets), nil
}

// CPULimits get cpu limits.
// If no limit is set, return ErrNoCFSLimit
func (c *cgroupv2) CPULimits() (float64, error) {
	data, err := c.readFile(path.Join(c.rootDir, "cpu.max"))
	if err != nil {
		return 0, err
	}
	parts := strings.Fields(data)
	if len(parts) != 2 {
		return 0, errors.New("invalid cpu.max format")
	}
	if parts[0] == "max" {
		return 0, ErrNoCFSLimit
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

// newCGroup detects and returns current cgroup
func newCGroup() (cgroup, error) {
	// Detect if it's cgroup v2
	_, err := os.Stat(path.Join(cgroupRootDir, "cgroup.controllers"))
	if err == nil {
		return &cgroupv2{
			rootDir:  cgroupRootDir,
			readFile: readFile,
		}, nil
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
	return &cgroupv1{
		cgroupSet: cgroupSet,
		rootDir:   cgroupRootDir,
		readFile:  readFile,
	}, nil
}
