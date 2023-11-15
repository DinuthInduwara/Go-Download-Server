package utils

import (
	"fmt"
	"time"
)

type DownloadFile struct {
	Url            string
	Fname          string
	Size           int64
	Completed      bool
	Cancel         chan bool
	DownloadedSize int64
	Started        time.Time
}

func (d *DownloadFile) Speed() string {
	elapsedTime := time.Since(d.Started)
	downloadedSize := float64(d.DownloadedSize)
	speed := downloadedSize / elapsedTime.Seconds()

	// Convert speed to Kbps or Mbps
	if speed < 1024 {
		return fmt.Sprintf("%.2f bps", speed)
	} else if speed < 1024*1024 {
		return fmt.Sprintf("%.2f Kbps", speed/1024)
	} else {
		return fmt.Sprintf("%.2f Mbps", speed/1024/1024)
	}
}

func (d *DownloadFile) Percentage() float32 {
	if d.Size == 0 {
		return 0.0
	}
	return (float32(d.DownloadedSize) / float32(d.Size)) * 100.0
}
