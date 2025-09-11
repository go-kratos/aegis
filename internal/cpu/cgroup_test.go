package cpu

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestCgroupv1_CPUCFSQuotaUs(t *testing.T) {
	cg := &cgroupv1{
		cgroupSet: map[string]string{"cpu": "/mock/cpu"},
		readFile: func(path string) (string, error) {
			if strings.HasSuffix(path, "cpu.cfs_quota_us") {
				return "100000", nil
			}
			return "", errors.New("not found")
		},
	}
	quota, err := cg.CPUCFSQuotaUs()
	if err != nil || quota != 100000 {
		t.Errorf("expected 100000, got %d, err: %v", quota, err)
	}
	// test no limit
	cg.readFile = func(path string) (string, error) { return "-1", nil }
	_, err = cg.CPUCFSQuotaUs()
	if !errors.Is(err, ErrNoCFSLimit) {
		t.Errorf("expected ErrNoCFSLimit, got %v", err)
	}
}

func TestCgroupv1_CPUCFSPeriodUs(t *testing.T) {
	cg := &cgroupv1{
		cgroupSet: map[string]string{"cpu": "/mock/cpu"},
		readFile: func(path string) (string, error) {
			return "100000", nil
		},
	}
	period, err := cg.CPUCFSPeriodUs()
	if err != nil || period != 100000 {
		t.Errorf("expected 100000, got %d, err: %v", period, err)
	}
	// test zero
	cg.readFile = func(path string) (string, error) { return "0", nil }
	_, err = cg.CPUCFSPeriodUs()
	if err == nil {
		t.Error("expected error for zero period")
	}
}

func TestCgroupv1_CPUAcctUsageNs(t *testing.T) {
	cg := &cgroupv1{
		cgroupSet: map[string]string{"cpuacct": "/mock/cpuacct"},
		readFile: func(path string) (string, error) {
			return "123456789", nil
		},
	}
	usage, err := cg.CPUAcctUsageNs()
	if err != nil || usage != 123456789 {
		t.Errorf("expected 123456789, got %d, err: %v", usage, err)
	}
}

func TestCgroupv1_CPUAcctUsagePerCPU(t *testing.T) {
	cg := &cgroupv1{
		cgroupSet: map[string]string{"cpuacct": "/mock/cpuacct"},
		readFile: func(path string) (string, error) {
			return "100 0 200 300", nil
		},
	}
	usages, err := cg.CPUAcctUsagePerCPU()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []uint64{100, 200, 300}
	if !reflect.DeepEqual(usages, want) {
		t.Errorf("expected %v, got %v", want, usages)
	}
}

func TestCgroupv1_CPUSetCPUs(t *testing.T) {
	cg := &cgroupv1{
		cgroupSet: map[string]string{"cpuset": "/mock/cpuset"},
		readFile: func(path string) (string, error) {
			return "0-2,4", nil
		},
	}
	cpus, err := cg.CPUSetCPUs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Slice(cpus, func(i, j int) bool { return cpus[i] < cpus[j] })
	want := []uint64{0, 1, 2, 4}
	if !reflect.DeepEqual(cpus, want) {
		t.Errorf("expected %v, got %v", want, cpus)
	}
}

func TestCgroupv1_LogicalCores(t *testing.T) {
	// Normal case
	cg := &cgroupv1{
		cgroupSet: map[string]string{"cpuacct": "/mock/cpuacct"},
		readFile: func(path string) (string, error) {
			return "100 200 300", nil
		},
	}
	cores, err := cg.LogicalCores()
	if err != nil || cores != 3 {
		t.Errorf("expected 3, got %d, err: %v", cores, err)
	}
	// Error from CPUAcctUsagePerCPU
	cg.readFile = func(path string) (string, error) { return "", errors.New("fail") }
	_, err = cg.LogicalCores()
	if err == nil {
		t.Error("expected error from LogicalCores when CPUAcctUsagePerCPU fails")
	}
}

func TestCgroupv1_CPULimits(t *testing.T) {
	cg := &cgroupv1{
		cgroupSet: map[string]string{"cpu": "/mock/cpu"},
		readFile: func(path string) (string, error) {
			if strings.HasSuffix(path, "cpu.cfs_quota_us") {
				return "20000", nil
			}
			if strings.HasSuffix(path, "cpu.cfs_period_us") {
				return "100000", nil
			}
			return "", errors.New("not found")
		},
	}
	limit, err := cg.CPULimits()
	if err != nil || limit != 0.2 {
		t.Errorf("expected 0.2, got %v, err: %v", limit, err)
	}
	// Error from CPUCFSQuotaUs
	cg.readFile = func(path string) (string, error) {
		if strings.HasSuffix(path, "cpu.cfs_quota_us") {
			return "", errors.New("fail quota")
		}
		return "100000", nil
	}
	_, err = cg.CPULimits()
	if err == nil {
		t.Error("expected error from CPULimits when CPUCFSQuotaUs fails")
	}
	// Error from CPUCFSPeriodUs
	cg.readFile = func(path string) (string, error) {
		if strings.HasSuffix(path, "cpu.cfs_quota_us") {
			return "20000", nil
		}
		return "", errors.New("fail period")
	}
	_, err = cg.CPULimits()
	if err == nil {
		t.Error("expected error from CPULimits when CPUCFSPeriodUs fails")
	}
	// Division by zero (period = 0)
	cg.readFile = func(path string) (string, error) {
		if strings.HasSuffix(path, "cpu.cfs_quota_us") {
			return "20000", nil
		}
		if strings.HasSuffix(path, "cpu.cfs_period_us") {
			return "0", nil
		}
		return "", errors.New("not found")
	}
	_, err = cg.CPULimits()
	if err == nil {
		t.Error("expected error for zero period in CPULimits")
	}
}

func TestCgroupv2_CPUAcctUsageNs(t *testing.T) {
	cg := &cgroupv2{
		rootDir: "/mock/cgroupv2",
		readFile: func(path string) (string, error) {
			if strings.HasSuffix(path, "cpu.stat") {
				return "usage_usec 12345\nuser_usec 10000\nsystem_usec 2345", nil
			}
			return "", errors.New("not found")
		},
	}
	usage, err := cg.CPUAcctUsageNs()
	if err != nil || usage != 12345000 {
		t.Errorf("expected 12345000, got %d, err: %v", usage, err)
	}

	// test missing usage_usec
	cg.readFile = func(path string) (string, error) {
		return "user_usec 10000\nsystem_usec 2345", nil
	}
	_, err = cg.CPUAcctUsageNs()
	if err == nil {
		t.Error("expected error for missing usage_usec")
	}
}

func TestCgroupv2_CPUSetCPUs(t *testing.T) {
	cg := &cgroupv2{
		rootDir: "/mock/cgroupv2",
		readFile: func(path string) (string, error) {
			if strings.HasSuffix(path, "cpuset.cpus.effective") {
				return "0-2,4", nil
			}
			return "", errors.New("not found")
		},
	}
	cpus, err := cg.CPUSetCPUs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sort.Slice(cpus, func(i, j int) bool { return cpus[i] < cpus[j] })
	want := []uint64{0, 1, 2, 4}
	if !reflect.DeepEqual(cpus, want) {
		t.Errorf("expected %v, got %v", want, cpus)
	}
}

func TestCgroupv2_LogicalCores(t *testing.T) {
	cg := &cgroupv2{
		rootDir: "/mock/cgroupv2",
		readFile: func(path string) (string, error) {
			return "0-3", nil
		},
	}
	cores, err := cg.LogicalCores()
	if err != nil || cores != 4 {
		t.Errorf("expected 4, got %d, err: %v", cores, err)
	}
}

func TestCgroupv2_CPULimits(t *testing.T) {
	cg := &cgroupv2{
		rootDir: "/mock/cgroupv2",
		readFile: func(path string) (string, error) {
			if strings.HasSuffix(path, "cpu.max") {
				return "20000 100000", nil
			}
			return "", errors.New("not found")
		},
	}
	limit, err := cg.CPULimits()
	if err != nil || limit != 0.2 {
		t.Errorf("expected 0.2, got %v, err: %v", limit, err)
	}

	// test no limit
	cg.readFile = func(path string) (string, error) { return "max 100000", nil }
	_, err = cg.CPULimits()
	if !errors.Is(err, ErrNoCFSLimit) {
		t.Errorf("expected ErrNoCFSLimit, got %v", err)
	}

	// test invalid format
	cg.readFile = func(path string) (string, error) { return "invalidformat", nil }
	_, err = cg.CPULimits()
	if err == nil {
		t.Error("expected error for invalid cpu.max format")
	}

	// test zero period
	cg.readFile = func(path string) (string, error) { return "20000 0", nil }
	_, err = cg.CPULimits()
	if err == nil {
		t.Error("expected error for zero period")
	}
}
