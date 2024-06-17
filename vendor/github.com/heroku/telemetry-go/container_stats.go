package telemetry

import (
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	refreshInterval = 10 * time.Second
	diskUsageError  = -1.0
)

var (
	sh    *statsHolder = &statsHolder{}
	csMux sync.Mutex
)

type statsHolder struct {
	NumGoroutines int64
	NumGCDelta    int64
	PreviousNumGC uint32
	NumPauseDelta int64
	PreviousPause uint64
	HeapInuse     uint64
	HeapObjects   uint64
	StackInuse    uint64
	StackSys      uint64
	Sys           uint64
	MemUsed       float64
	LoadAvg1m     float64
	DiskUsed      float64
	LastRefreshed time.Time
}

// ContainerStats holds the container information at a given time
type ContainerStats struct {
	NumGoroutines int64
	NumGCDelta    int64
	NumPauseDelta int64
	HeapInuse     uint64
	HeapObjects   uint64
	StackInuse    uint64
	StackSys      uint64
	Sys           uint64
	MemUsed       float64
	LoadAvg1m     float64
	DiskUsed      float64
	LastRefreshed time.Time
	Elapsed       time.Duration
}

// ReadContainerStats reads the latest container stats values
func ReadContainerStats() ContainerStats {
	var elapsed time.Duration

	csMux.Lock()
	if sh.needsRefreshing() {
		elapsed = sh.refresh()
	}
	csMux.Unlock()

	return ContainerStats{
		NumGoroutines: sh.NumGoroutines,
		NumGCDelta:    sh.NumGCDelta,
		NumPauseDelta: sh.NumPauseDelta,
		HeapInuse:     sh.HeapInuse,
		HeapObjects:   sh.HeapObjects,
		StackInuse:    sh.StackInuse,
		StackSys:      sh.StackInuse,
		Sys:           sh.Sys,
		MemUsed:       sh.MemUsed,
		LoadAvg1m:     sh.LoadAvg1m,
		DiskUsed:      sh.DiskUsed,
		LastRefreshed: sh.LastRefreshed,
		Elapsed:       elapsed,
	}
}

func (s *statsHolder) needsRefreshing() bool {
	if s.LastRefreshed.IsZero() {
		return true
	}

	return time.Now().After(s.LastRefreshed.Add(refreshInterval))
}

func (s *statsHolder) refresh() time.Duration {
	start := time.Now()
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	s.NumGoroutines = int64(runtime.NumGoroutine())
	s.NumGCDelta = int64(stats.NumGC - uint32(s.PreviousNumGC))
	s.PreviousNumGC = stats.NumGC
	s.NumPauseDelta = int64(uint32(stats.PauseTotalNs) - uint32(s.PreviousNumGC))
	s.PreviousPause = stats.PauseTotalNs
	s.HeapInuse = stats.HeapInuse
	s.HeapObjects = stats.HeapObjects
	s.StackInuse = stats.StackInuse
	s.StackSys = stats.StackSys
	s.Sys = stats.PauseTotalNs
	s.MemUsed = s.readMemUsage()
	s.LoadAvg1m = s.readLoadAvg()
	s.DiskUsed = s.readDiskUsage()
	s.LastRefreshed = time.Now()

	return time.Since(start)
}

func (*statsHolder) readMemUsage() float64 {
	v, _ := mem.VirtualMemory()
	return v.UsedPercent
}

func (*statsHolder) readLoadAvg() float64 {
	l, _ := load.Avg()
	return l.Load1
}

func (*statsHolder) readDiskUsage() float64 {
	if config.DiskRoot == "" {
		return 0
	}

	u, err := disk.Usage(config.DiskRoot)
	if err != nil {
		return diskUsageError
	}
	return u.UsedPercent
}
