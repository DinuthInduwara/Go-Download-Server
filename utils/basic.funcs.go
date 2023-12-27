package utils

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
)

func FolderSize(path string) int64 {
	var size int64

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0
	}

	return size
}

func CountFilesAndFolders(path string) (int, int) {
	var fileCount, folderCount int

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println(err)

			return err
		}
		if info.IsDir() {
			folderCount++
		} else {
			fileCount++
		}
		return nil
	})

	if err != nil {
		log.Println(err)
		return 0, 0
	}

	return fileCount, folderCount
}

func CpuCount() int                  { return runtime.NumCPU() }
func Memory() *mem.VirtualMemoryStat { memory, _ := mem.VirtualMemory(); return memory }

func Disk() *disk.UsageStat { usage, _ := disk.Usage("./"); return usage }

func NetworkSpeed(interval time.Duration) (float64, float64) {
	var downloadSpeed float64
	var uploadSpeed float64

	// Get initial network statistics
	stats1, err := net.IOCounters(false)
	if err != nil {
		log.Println(err)
		return 0, 0
	}

	// Wait for the specified interval
	time.Sleep(interval)

	// Get network statistics again
	stats2, err := net.IOCounters(false)
	if err != nil {
		log.Println(err)
		return 0, 0
	}

	// Calculate download and upload speeds
	downloadSpeed = float64(stats2[0].BytesRecv-stats1[0].BytesRecv) / interval.Seconds()
	uploadSpeed = float64(stats2[0].BytesSent-stats1[0].BytesSent) / interval.Seconds()

	return downloadSpeed, uploadSpeed
}

func NetUsageStats() uint64 {
	interfaces, err := net.IOCounters(true)
	if err != nil {
		log.Println("Error:", err)
		return 0
	}
	var usage uint64

	for _, iface := range interfaces {
		usage += iface.BytesSent + iface.BytesRecv
	}
	return usage
}
