package utils

import (
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func DownloadDirect(dir, url, fname string, Downloads map[string]*DownloadFile) {
	download := &DownloadFile{
		Url:    url,
		Fname:  fname,
		Size:   0,
		Cancel: make(chan bool),
	}

	outputFile, err := os.Create(dir + "/" + download.Fname)
	if err != nil {
		log.Println("Error creating the output file:", err)
		return
	}

	// create request
	req, err := http.NewRequest("GET", download.Url, nil)
	if err != nil {
		log.Println("Error creating HTTP request:", err)
		return
	}
	defer outputFile.Close()

	// send the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error making HTTP request:", err)
		return
	}
	defer resp.Body.Close()

	// update file total size and started time
	download.Size = resp.ContentLength
	download.Started = time.Now()

	// create buffer chunk size
	buffer := make([]byte, 1024)

	// add download object to map
	Downloads[download.Url] = download

	for {
		select {
		case <-download.Cancel:
			log.Println("Download canceled.")
			close(download.Cancel)
			delete(Downloads, download.Url)
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
				download.DownloadedSize += int64(n)
			}

			if err == io.EOF {
				download.Completed = true
				delete(Downloads, download.Url)
				close(download.Cancel)
				return
			}
		}
	}

}
