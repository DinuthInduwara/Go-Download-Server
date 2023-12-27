package utils

import (
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func NewDownloader(url, dir, fname string) *DownloadFile {
	return &DownloadFile{
		Url:      url,
		Fname:    dir + "/" + fname,
		Size:     0,
		paused:   false,
		canceled: false, Completed: false, DownloadedSize: 0,
		Started:    time.Now(),
		CancelChan: make(chan bool),
		PauseChan:  make(chan bool),
	}
}

type DownloadFile struct {
	Url            string
	paused         bool
	Fname          string
	Size           int64
	Completed      bool
	canceled       bool
	DownloadedSize int64
	Started        time.Time
	CancelChan     chan bool
	PauseChan      chan bool
}

func (d *DownloadFile) Speed() float64 {
	elapsedTime := time.Since(d.Started)
	downloadedSize := float64(d.DownloadedSize)
	return downloadedSize / elapsedTime.Seconds()
}

func (d *DownloadFile) Percentage() float32 {
	if d.Size == 0 {
		return 0.0
	}
	return (float32(d.DownloadedSize) / float32(d.Size)) * 100.0
}

func (d *DownloadFile) IsPaused() bool   { return !d.paused }
func (d *DownloadFile) IsCanceled() bool { return d.canceled }

func (d *DownloadFile) Pause() string {
	if !d.paused {
		d.PauseChan <- true
		return "Task Paused"
	}
	return "Task Already Paused"
}
func (d *DownloadFile) Resume() {

	if !d.IsPaused() {
		return
	}

	// create request
	req, err := http.NewRequest("GET", d.Url, nil)
	if err != nil {
		log.Println("Error creating HTTP request:", err)
		return
	}
	info, err := os.Stat(d.Fname)
	if err != nil {
		log.Println("Error getting file info:", err)
	} else {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(info.Size(), 10)+"-")
		d.DownloadedSize = info.Size()
	}

	// send the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error making HTTP request:", err)
		return
	}
	defer resp.Body.Close()
	d.paused = false

	// update file total size and started time
	d.Size = resp.ContentLength
	d.Started = time.Now()

	// create buffer chunk size
	buffer := make([]byte, 1024)

	//open output file
	outputFile, err := os.OpenFile(d.Fname, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Error opening the output file: %s", err)
		return
	}
	defer outputFile.Close()
	for {
		select {
		case <-d.CancelChan:
			log.Println("[X] Download canceled.")
			close(d.CancelChan)
			close(d.PauseChan)
			d.canceled = true
			return
		case <-d.PauseChan:
			log.Printf("[*] Download paused: %s", d.Url)
			return
		default:
			n, err := resp.Body.Read(buffer)
			if err != nil && err != io.EOF {
				log.Println("Error reading from response:", err)
				return
			}

			if n > 0 {
				// Write the chunk to the output file
				_, err := outputFile.Write(buffer[:n])
				if err != nil {
					log.Println("Error writing to the output file:", err)
					return
				}

				// Update DownloadedSize
				d.DownloadedSize += int64(n)
			}

			if err == io.EOF {
				d.Completed = true
				close(d.CancelChan)
				close(d.PauseChan)
				return
			}
		}
	}

}
