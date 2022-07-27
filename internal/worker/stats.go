package worker

import (
	"fmt"

	"github.com/c9s/goprocinfo/linux"
)

type Stats struct {
	MemStats  *linux.MemInfo
	DiskStats *linux.Disk
	CPUStats  *linux.CPUStat
	LoadStats *linux.LoadAvg
	TaskCount int
}

type CPUStat struct {
	ID        string `json:"id"`
	User      uint64 `json:"user"`
	Nice      uint64 `json:"nice"`
	System    uint64 `json:"system"`
	Idle      uint64 `json:"idle"`
	IOWait    uint64 `json:"iowait"`
	IRQ       uint64 `json:"irq"`
	SoftIRQ   uint64 `json:"softirq"`
	Steal     uint64 `json:"steal"`
	Guest     uint64 `json:"guest"`
	GuestNice uint64 `json:"guest_nice"`
}

func (s *Stats) MemTotalKb() uint64 {
	return s.MemStats.MemTotal
}

func (s *Stats) MemAvailableKb() uint64 {
	return s.MemStats.MemAvailable
}

func (s *Stats) MemUsedKb() uint64 {
	return s.MemStats.MemTotal - s.MemStats.MemAvailable
}

func (s *Stats) MemUsedPercent() uint64 {
	return s.MemStats.MemAvailable / s.MemStats.MemTotal
}

func (s *Stats) DiskTotal() uint64 {
	return s.DiskStats.All
}

func (s *Stats) DiskFree() uint64 {
	return s.DiskStats.Free
}

func (s *Stats) DiskUsed() uint64 {
	return s.DiskStats.Used
}

func (s *Stats) CPUUsage() float64 {
	idle := s.CPUStats.Idle + s.CPUStats.IOWait
	nonIdle := s.CPUStats.User + s.CPUStats.Nice + s.CPUStats.System +
		s.CPUStats.IRQ + s.CPUStats.SoftIRQ + s.CPUStats.Steal
	total := idle + nonIdle

	if total == 0 {
		return 0.00
	}

	return (float64(total) - float64(idle)) / float64(total)
}

func GetStats() *Stats {
	return &Stats{
		MemStats:  GetMemoryInfo(),
		DiskStats: GetDiskInfo(),
		CPUStats:  GetCPUStats(),
		LoadStats: GetLoadAvg(),
	}
}

func GetMemoryInfo() *linux.MemInfo {
	memstats, err := linux.ReadMemInfo("/proc/meminfo")
	if err != nil {
		fmt.Printf("Error reading from /proc/meminfo")
		return &linux.MemInfo{}
	}

	return memstats
}

// GetDiskInfo See https://godoc.org/github.com/c9s/goprocinfo/linux#Disk
func GetDiskInfo() *linux.Disk {
	diskstats, err := linux.ReadDisk("/")
	if err != nil {
		fmt.Printf("Error reading from /")
		return &linux.Disk{}
	}

	return diskstats
}

// GetCPUStats See https://godoc.org/github.com/c9s/goprocinfo/linux#CPUStat
func GetCPUStats() *linux.CPUStat {
	stats, err := linux.ReadStat("/proc/stat")
	if err != nil {
		fmt.Printf("Error reading from /proc/stat")
		return &linux.CPUStat{}
	}

	return &stats.CPUStatAll
}

// GetLoadAvg See https://godoc.org/github.com/c9s/goprocinfo/linux#LoadAvg
func GetLoadAvg() *linux.LoadAvg {
	loadavg, err := linux.ReadLoadAvg("/proc/loadavg")
	if err != nil {
		fmt.Printf("Error reading from /proc/loadavg")
		return &linux.LoadAvg{}
	}

	return loadavg
}
