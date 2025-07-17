package monitor

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/sirupsen/logrus"
)

// ResourceStats represents system resource statistics
type ResourceStats struct {
	CPU       CPUStats     `json:"cpu"`
	Memory    MemoryStats  `json:"memory"`
	Disk      DiskStats    `json:"disk"`
	Network   NetworkStats `json:"network"`
	Host      HostStats    `json:"host"`
	Runtime   RuntimeStats `json:"runtime"`
	Timestamp time.Time    `json:"timestamp"`
}

// CPUStats represents CPU statistics
type CPUStats struct {
	UsagePercent []float64 `json:"usage_percent"`
	Cores        int       `json:"cores"`
	TotalPercent float64   `json:"total_percent"`
}

// MemoryStats represents memory statistics
type MemoryStats struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	Free        uint64  `json:"free"`
	Cached      uint64  `json:"cached"`
	Buffers     uint64  `json:"buffers"`
}

// DiskStats represents disk statistics
type DiskStats struct {
	Total       uint64           `json:"total"`
	Free        uint64           `json:"free"`
	Used        uint64           `json:"used"`
	UsedPercent float64          `json:"used_percent"`
	Partitions  []PartitionStats `json:"partitions"`
}

// PartitionStats represents partition statistics
type PartitionStats struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkStats represents network statistics
type NetworkStats struct {
	BytesSent   uint64             `json:"bytes_sent"`
	BytesRecv   uint64             `json:"bytes_recv"`
	PacketsSent uint64             `json:"packets_sent"`
	PacketsRecv uint64             `json:"packets_recv"`
	Interfaces  []NetworkInterface `json:"interfaces"`
}

// NetworkInterface represents network interface statistics
type NetworkInterface struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
}

// HostStats represents host system statistics
type HostStats struct {
	Hostname        string    `json:"hostname"`
	OS              string    `json:"os"`
	Platform        string    `json:"platform"`
	PlatformFamily  string    `json:"platform_family"`
	PlatformVersion string    `json:"platform_version"`
	KernelVersion   string    `json:"kernel_version"`
	Uptime          uint64    `json:"uptime"`
	BootTime        time.Time `json:"boot_time"`
	Temperature     []float64 `json:"temperature,omitempty"`
}

// RuntimeStats represents Go runtime statistics
type RuntimeStats struct {
	Goroutines    int    `json:"goroutines"`
	MemAllocBytes uint64 `json:"mem_alloc_bytes"`
	MemTotalBytes uint64 `json:"mem_total_bytes"`
	MemSysBytes   uint64 `json:"mem_sys_bytes"`
	GCCycles      uint32 `json:"gc_cycles"`
	HeapObjects   uint64 `json:"heap_objects"`
	CGOCalls      int64  `json:"cgo_calls"`
}

// ResourceMonitor monitors system resources
type ResourceMonitor struct {
	logger            *logrus.Logger
	enableTemperature bool
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(logger *logrus.Logger) *ResourceMonitor {
	return &ResourceMonitor{
		logger:            logger,
		enableTemperature: runtime.GOARCH == "arm" || runtime.GOARCH == "arm64", // Enable for Raspberry Pi
	}
}

// GetResourceStats collects current system resource statistics
func (r *ResourceMonitor) GetResourceStats(ctx context.Context) (*ResourceStats, error) {
	stats := &ResourceStats{
		Timestamp: time.Now(),
	}

	// Collect CPU stats
	cpuStats, err := r.getCPUStats(ctx)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get CPU stats")
	} else {
		stats.CPU = *cpuStats
	}

	// Collect memory stats
	memStats, err := r.getMemoryStats(ctx)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get memory stats")
	} else {
		stats.Memory = *memStats
	}

	// Collect disk stats
	diskStats, err := r.getDiskStats(ctx)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get disk stats")
	} else {
		stats.Disk = *diskStats
	}

	// Collect network stats
	netStats, err := r.getNetworkStats(ctx)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get network stats")
	} else {
		stats.Network = *netStats
	}

	// Collect host stats
	hostStats, err := r.getHostStats(ctx)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get host stats")
	} else {
		stats.Host = *hostStats
	}

	// Collect runtime stats
	stats.Runtime = *r.getRuntimeStats()

	return stats, nil
}

// getCPUStats collects CPU statistics
func (r *ResourceMonitor) getCPUStats(ctx context.Context) (*CPUStats, error) {
	// Get per-CPU usage
	perCPU, err := cpu.PercentWithContext(ctx, time.Second, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get per-CPU usage: %w", err)
	}

	// Get total CPU usage
	totalCPU, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get total CPU usage: %w", err)
	}

	totalPercent := 0.0
	if len(totalCPU) > 0 {
		totalPercent = totalCPU[0]
	}

	return &CPUStats{
		UsagePercent: perCPU,
		Cores:        len(perCPU),
		TotalPercent: totalPercent,
	}, nil
}

// getMemoryStats collects memory statistics
func (r *ResourceMonitor) getMemoryStats(ctx context.Context) (*MemoryStats, error) {
	vmem, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory stats: %w", err)
	}

	return &MemoryStats{
		Total:       vmem.Total,
		Available:   vmem.Available,
		Used:        vmem.Used,
		UsedPercent: vmem.UsedPercent,
		Free:        vmem.Free,
		Cached:      vmem.Cached,
		Buffers:     vmem.Buffers,
	}, nil
}

// getDiskStats collects disk statistics
func (r *ResourceMonitor) getDiskStats(ctx context.Context) (*DiskStats, error) {
	// Get partitions
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk partitions: %w", err)
	}

	var totalDisk, freeDisk, usedDisk uint64
	var partitionStats []PartitionStats

	for _, partition := range partitions {
		usage, err := disk.UsageWithContext(ctx, partition.Mountpoint)
		if err != nil {
			r.logger.WithError(err).WithField("partition", partition.Device).Warn("Failed to get partition usage")
			continue
		}

		partStat := PartitionStats{
			Device:      partition.Device,
			Mountpoint:  partition.Mountpoint,
			Fstype:      partition.Fstype,
			Total:       usage.Total,
			Free:        usage.Free,
			Used:        usage.Used,
			UsedPercent: usage.UsedPercent,
		}
		partitionStats = append(partitionStats, partStat)

		// Accumulate totals (only for root filesystem or first partition)
		if partition.Mountpoint == "/" || totalDisk == 0 {
			totalDisk = usage.Total
			freeDisk = usage.Free
			usedDisk = usage.Used
		}
	}

	usedPercent := 0.0
	if totalDisk > 0 {
		usedPercent = float64(usedDisk) / float64(totalDisk) * 100
	}

	return &DiskStats{
		Total:       totalDisk,
		Free:        freeDisk,
		Used:        usedDisk,
		UsedPercent: usedPercent,
		Partitions:  partitionStats,
	}, nil
}

// getNetworkStats collects network statistics
func (r *ResourceMonitor) getNetworkStats(ctx context.Context) (*NetworkStats, error) {
	ioCounters, err := net.IOCountersWithContext(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get network IO counters: %w", err)
	}

	var totalBytesSent, totalBytesRecv, totalPacketsSent, totalPacketsRecv uint64
	var interfaces []NetworkInterface

	for _, counter := range ioCounters {
		// Skip loopback and virtual interfaces
		if counter.Name == "lo" || counter.Name == "docker0" {
			continue
		}

		iface := NetworkInterface{
			Name:        counter.Name,
			BytesSent:   counter.BytesSent,
			BytesRecv:   counter.BytesRecv,
			PacketsSent: counter.PacketsSent,
			PacketsRecv: counter.PacketsRecv,
		}
		interfaces = append(interfaces, iface)

		totalBytesSent += counter.BytesSent
		totalBytesRecv += counter.BytesRecv
		totalPacketsSent += counter.PacketsSent
		totalPacketsRecv += counter.PacketsRecv
	}

	return &NetworkStats{
		BytesSent:   totalBytesSent,
		BytesRecv:   totalBytesRecv,
		PacketsSent: totalPacketsSent,
		PacketsRecv: totalPacketsRecv,
		Interfaces:  interfaces,
	}, nil
}

// getHostStats collects host system statistics
func (r *ResourceMonitor) getHostStats(ctx context.Context) (*HostStats, error) {
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	stats := &HostStats{
		Hostname:        hostInfo.Hostname,
		OS:              hostInfo.OS,
		Platform:        hostInfo.Platform,
		PlatformFamily:  hostInfo.PlatformFamily,
		PlatformVersion: hostInfo.PlatformVersion,
		KernelVersion:   hostInfo.KernelVersion,
		Uptime:          hostInfo.Uptime,
		BootTime:        time.Unix(int64(hostInfo.BootTime), 0),
	}

	// Get temperature (for Raspberry Pi)
	if r.enableTemperature {
		temps, err := host.SensorsTemperaturesWithContext(ctx)
		if err == nil {
			var temperatures []float64
			for _, temp := range temps {
				temperatures = append(temperatures, temp.Temperature)
			}
			stats.Temperature = temperatures
		}
	}

	return stats, nil
}

// getRuntimeStats collects Go runtime statistics
func (r *ResourceMonitor) getRuntimeStats() *RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &RuntimeStats{
		Goroutines:    runtime.NumGoroutine(),
		MemAllocBytes: m.Alloc,
		MemTotalBytes: m.TotalAlloc,
		MemSysBytes:   m.Sys,
		GCCycles:      m.NumGC,
		HeapObjects:   m.HeapObjects,
		CGOCalls:      runtime.NumCgoCall(),
	}
}

// GetUsagePercentages returns simplified usage percentages for alerting
func (r *ResourceMonitor) GetUsagePercentages(ctx context.Context) (cpu, memory, disk float64, err error) {
	stats, err := r.GetResourceStats(ctx)
	if err != nil {
		return 0, 0, 0, err
	}

	return stats.CPU.TotalPercent, stats.Memory.UsedPercent, stats.Disk.UsedPercent, nil
}
