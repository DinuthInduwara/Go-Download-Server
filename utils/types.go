package utils

import (
	"fmt"
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
	Error          error
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

func (d *DownloadFile) Cancel() bool {
	if !d.canceled && !d.Completed { // if not already cancelled and not completed then cancel download progress
		d.CancelChan <- true
		return true
	}
	return false
}

func (d *DownloadFile) Resume() bool {

	if !d.IsPaused() {
		return true
	}

	// create request
	req, err := http.NewRequest("GET", d.Url, nil)
	if err != nil {
		log_and_set_error(d, "error creating HTTP request", err)
		return true
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
		log_and_set_error(d, "error making HTTP request", err)
		return true
	}
	if resp.StatusCode != 200 {
		if resp.StatusCode == 416 {
			d.Completed = true
			return false
		}
		log_and_set_error(d, resp.Status, err)
		return true
	}
	defer resp.Body.Close()
	d.paused = false

	// update file total size and started time
	d.Size = resp.ContentLength + d.DownloadedSize // total size with downloaded part
	d.Started = time.Now()

	// create buffer chunk size
	buffer := make([]byte, 1024)

	//open output file
	outputFile, err := os.OpenFile(d.Fname, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log_and_set_error(d, "error opening the output file", err)
		return true
	}
	defer outputFile.Close()
	for {
		select {
		case <-d.CancelChan:
			log.Println("[X] Download canceled.")
			close(d.CancelChan)
			close(d.PauseChan)
			d.canceled = true
			return true
		case <-d.PauseChan:
			log.Printf("[*] Download paused: %s", d.Url)
			return true
		default:
			n, err := resp.Body.Read(buffer)
			if err != nil && err != io.EOF {
				log_and_set_error(d, "error reading from response", err)
				return true
			}

			if n > 0 {
				// Write the chunk to the output file
				_, err := outputFile.Write(buffer[:n])
				if err != nil {
					log_and_set_error(d, "error writing to the output file", err)
					return true
				}

				// Update DownloadedSize
				d.DownloadedSize += int64(n)
			}

			if err == io.EOF {
				d.Completed = true
				close(d.CancelChan)
				close(d.PauseChan)
				return true
			}
		}
	}

}

func log_and_set_error(d *DownloadFile, msg string, err error) {
	err = fmt.Errorf("%s: %s", msg, err)
	d.Error = err
	log.Println(err)
}
