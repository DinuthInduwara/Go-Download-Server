package main

import (
	"DinuthInduwara/GoMirrorServer/utils"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var Downloads = make(map[string]*utils.DownloadFile)

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

		go utils.DownloadDirect(dir, url, fname, Downloads)
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

		if task := Downloads[url]; task != nil {
			task.Cancel <- true
			w.Write([]byte("Task Cancelled..."))
			return
		}

		http.Error(w, "No Downloading Task", http.StatusBadRequest)

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

	mux.HandleFunc("/yt-dlp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		url := r.FormValue("url")
		if url == "" {
			http.Error(w, "`url` required", http.StatusLocked)
			return
		}

		go utils.DownloadYTDLP(url, dir, Downloads)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Task Added To Queue"))
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
