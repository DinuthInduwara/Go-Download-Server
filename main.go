package main

import (
	"DinuthInduwara/GoMirrorServer/utils"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var Downloads = make(map[string]*utils.DownloadFile)
var Encrypting = make(map[string]*utils.CryptFile)

func main() {
	// Specify the directory you want to serve files from
	dir := "./static"

	// Create a ServeMux to handle custom routes
	mux := mux.NewRouter()
	mux.Use(loggingMiddleware)

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

	// resume direct download task
	mux.HandleFunc("/resume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		url := r.FormValue("url")
		if download, done := Downloads[url]; done {
			go utils.DoDownload(Downloads, download)
			w.Write([]byte("Task Resumed"))
			return
		}
		w.Write([]byte("No Task Resumed"))
	})

	// Pause direct downloads
	mux.HandleFunc("/pause", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		url := r.FormValue("url")
		if download, ok := Downloads[url]; ok {
			message := download.Pause()
			w.Write([]byte(message))
		}
		w.Write([]byte("No Download Task To Pause"))

	})

	// create direct download task
	mux.HandleFunc("/direct-download", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		url := r.FormValue("url")
		fname := r.FormValue("file_name")

		if fname == "" {
			parts := strings.Split(url, "/")
			fname = parts[len(parts)-1]
		}

		download := utils.NewDownloader(url, dir, fname)

		go utils.DoDownload(Downloads, download)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Task Added To Queue"))
	})

	// create a ServeMux to handle cancel downloads
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		url := r.FormValue("url")
		if url == "" {
			http.Error(w, "`url` field required to stop download progress", http.StatusBadRequest)
			return
		}

		if task, done := Downloads[url]; done {
			task.CancelChan <- true
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
			Speed           float64 `json:"speed"`
			Url             string  `json:"url"`
			Paused          bool    `json:"paused"`
		}
		type crypting struct {
			FSize       int64  `json:"fsize"`
			Fname       string `json:"filename"`
			Mode        string `json:"mode"`
			CryptedSize int64  `json:"cryptedSize"`
			Percentage  int    `json:"percentage"`
		}

		var downloadArr = []*ResponseCreator{}
		for _, item := range Downloads {
			downloadArr = append(downloadArr, &ResponseCreator{
				Size:            item.Size,
				DownloadedBytes: item.DownloadedSize,
				Fname:           item.Fname,
				Speed:           item.Speed(),
				Url:             item.Url,
				Paused:          item.IsPaused(),
			})
		}

		var cryptingArr = []*crypting{}
		for _, item := range Encrypting {
			cryptingArr = append(cryptingArr, &crypting{
				FSize:       item.FSize,
				Fname:       item.Fname,
				Mode:        item.Task,
				CryptedSize: item.CryperdSize,
				Percentage:  item.Percentage(),
			})
		}

		combinedData := make(map[string]interface{})
		combinedData["downloads"] = downloadArr
		combinedData["crypting"] = cryptingArr
		responseData, err := json.Marshal(combinedData)
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

	// create a ServeMux to handle Yt-dlp downloads
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

	// create a ServeMux to handle encrypt files
	mux.HandleFunc("/sys", func(w http.ResponseWriter, r *http.Request) {
		rclone_tasks := func() int {
			req, _ := http.NewRequest("POST", "http://127.0.0.1:5572/core/stats", nil)
			client := &http.Client{}
			response, err := client.Do(req)

			if err != nil {
				log.Fatal("Error GET RCLONE Tasks request: ", err)
				return 0
			}
			defer response.Body.Close()
			type FileTransferResponse struct {
				Transferring []interface{} `json:"transferring"`
			}
			// Read response body
			body, err := io.ReadAll(response.Body)
			if err != nil {
				log.Fatal("Error reading response body: ", err)
				return 0
			}

			// Parse JSON response
			var fileTransferResponse FileTransferResponse
			err = json.Unmarshal(body, &fileTransferResponse)
			if err != nil {
				log.Fatal("Error parsing JSON: ", err)
				return 0
			}
			return len(fileTransferResponse.Transferring)
		}()

		type ServerStatus struct {
			Cpu           int     `json:"cpu"`
			MemTot        uint64  `json:"mem_total"`
			MemUse        uint64  `json:"mem_used"`
			DiskUsed      uint64  `json:"disk_used"`
			DiskTotal     uint64  `json:"disk_total"`
			DownloadFSize int64   `json:"down_size"`
			DownloadSpeed float64 `json:"down_speed"`
			UploadSpeed   float64 `json:"up_speed"`
			NetUsage      uint64  `json:"net_usage"`
			FolderCount   int     `json:"folder_count"`
			FileCount     int     `json:"file_count"`
			Downloads     int     `json:"download_tasks"`
			RcloneTrans   int     `json:"rclone_tasks"`
		}
		disk := utils.Disk()
		down, up := utils.NetworkSpeed(time.Second)
		files, folders := utils.CountFilesAndFolders(dir)
		res := ServerStatus{
			Cpu:           utils.CpuCount(),
			MemTot:        utils.Memory().Total,
			MemUse:        utils.Memory().Used,
			DiskUsed:      disk.Used,
			DiskTotal:     disk.Total,
			DownloadFSize: utils.FolderSize(dir),
			DownloadSpeed: down,
			UploadSpeed:   up,
			NetUsage:      utils.NetUsageStats(),
			FolderCount:   folders,
			FileCount:     files,
			Downloads:     len(Downloads),
			RcloneTrans:   rclone_tasks,
		}
		responseData, err := json.Marshal(res)
		if err != nil {
			http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseData)
	})

	// Register the file server at the "/fs" route
	mux.PathPrefix("/fs/").Handler(http.StripPrefix("/fs/", http.FileServer(http.Dir(dir))))

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

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log the request information
		log.Printf(
			"[%s] %s %s %s",
			r.Method,
			r.RemoteAddr,
			r.URL.Path,
			time.Since(start),
		)

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}
