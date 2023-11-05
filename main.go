package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
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

var Downloads = []*DownloadFile{}

func IndexOf[T comparable](arr []T, item T) int {
	for i, val := range arr {
		if val == item {
			return i
		}
	}
	return -1
}

func main() {
	// Specify the directory you want to serve files from
	dir := "./static"

	// Create a file server handler
	fs := http.FileServer(http.Dir(dir))

	// Create a custom handler to log requests
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		fs.ServeHTTP(w, r)
	})

	// Create a ServeMux to handle custom routes
	mux := http.NewServeMux()

	//  Create a ServeMux to handle delete files
	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		fileToDelete := r.FormValue("file")
		if fileToDelete == "" {
			http.Error(w, "No file specified", http.StatusBadRequest)
			return
		}

		filePath := dir + "/" + fileToDelete
		err := os.Remove(filePath)
		if err != nil {
			http.Error(w, "Failed to delete the file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("File deleted successfully"))
	})

	//  Create a ServeMux to handle rename files
	mux.HandleFunc("/rename", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		oldName := r.FormValue("old_name")
		newName := r.FormValue("new_name")

		if oldName == "" || newName == "" {
			http.Error(w, "Both old_name and new_name must be specified", http.StatusBadRequest)
			return
		}

		oldPath := dir + "/" + oldName
		newPath := dir + "/" + newName

		oldExt := filepath.Ext(oldName)
		newExt := filepath.Ext(newPath)
		if newExt == "" {
			newPath += oldExt
		}

		err := os.Rename(oldPath, newPath)
		if err != nil {
			http.Error(w, "Failed to rename the file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("File renamed successfully"))
	})

	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		url := r.FormValue("url")
		fname := r.FormValue("file_name")

		if fname == "" || url == "" {
			http.Error(w, "`file_name` and `url` required", http.StatusLocked)
			return
		}

		go func() {
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

			// append download object to list
			Downloads = append(Downloads, download)

			for {
				select {
				case <-download.Cancel:
					log.Println("Download canceled.")
					index := IndexOf(Downloads, download)
					Downloads = append(Downloads[:index], Downloads[index+1:]...)
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
						index := IndexOf(Downloads, download)
						Downloads = append(Downloads[:index], Downloads[index+1:]...)
						close(download.Cancel)
						return
					}
				}
			}

		}()
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Task Added To Queue"))
	})

	// create a ServeMux to handle cancel downloads
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		url := r.FormValue("url")
		if url == "" {
			http.Error(w, "`url` field required to stop download progress", http.StatusBadRequest)
			return
		}

		for _, i := range Downloads {
			if i.Url == url {
				i.Cancel <- true // canceling download
				close(i.Cancel)  // close cancel channel

				err := os.Remove(dir + "/" + i.Fname) // delete uncompleted file
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			}
		}
	})

	// create a ServeMux to handle send download status
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		type ResponseCreator struct {
			Size            int64   `json:"size"`
			DownloadedBytes int64   `json:"downloaded"`
			Fname           string  `json:"fname"`
			Percentage      float32 `json:"percentage"`
			Speed           string  `json:"speed"`
			Url             string  `json:"url"`
		}
		var arr = []*ResponseCreator{}
		for _, item := range Downloads {
			arr = append(arr, &ResponseCreator{
				Size:            item.Size,
				DownloadedBytes: item.DownloadedSize,
				Fname:           item.Fname,
				Percentage:      item.Percentage(),
				Speed:           item.Speed(),
				Url:             item.Url,
			})
		}

		responseData, err := json.Marshal(arr)
		if err != nil {
			http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
			return
		}

		// Set the content type and write the JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(responseData)

		if err != nil {
			http.Error(w, "Failed to write JSON response", http.StatusInternalServerError)
			return
		}
	})

	mux.Handle("/", handler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start the server
	log.Printf("Server started on :8080...")
	err := server.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
