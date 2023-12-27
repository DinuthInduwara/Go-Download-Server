package utils

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/google/uuid"
)

func DownloadYTDLP(url, dir string, Downloads map[string]*DownloadFile) {
	cmd := exec.Command("yt-dlp", url, "-s", "--print-json")

	// Capture the command's output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error running yt-dlp:", err)
		return
	}

	// Write the JSON output to a temporary file
	jsonFile, err := writeJSON(output)
	if err != nil {
		log.Println("Error occurred:", err)
		return
	}
	defer os.Remove(jsonFile) // Delete the temporary JSON file

	// Load file info from the JSON
	info, err := getInfo(jsonFile)
	if err != nil {
		log.Println("Error loading data:", err)
		return
	}

	// Run yt-dlp to download the video based on the JSON file
	cmd = exec.Command("yt-dlp", "--load-info-json", jsonFile, "-o", "-", "-q")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Error creating stdout pipe:", err)
		return
	}

	// starting command
	if err := cmd.Start(); err != nil {
		log.Println("Error starting the command:", err)
		return
	}

	// create output file
	file, err := os.Create(path.Join(dir, info.Fname))
	if err != nil {
		log.Println("Error creating the output file:", err)
		return
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	info.Started = time.Now()
	defer delete(Downloads, info.Url)
	Downloads[url] = info
	for {
		select {
		case <-info.CancelChan:
			log.Println("Download canceled.", info.Fname)
			return
		default:
			n, err := stdout.Read(buffer)
			if err != nil && err != io.EOF {
				log.Println("Error reading from stdout:", err)
				return
			}
			if n == 0 {
				return
			}

			_, err = file.Write(buffer[:n])
			info.DownloadedSize += int64(n)

			if err != nil {
				log.Println("Error writing to the output file:", err)
				return
			}
		}
	}

}

func writeJSON(jsonBytes []byte) (string, error) {
	var jsonData interface{}
	err := json.Unmarshal(jsonBytes, &jsonData)
	if err != nil {
		return "", err
	}

	// Save JSON data to the specified file
	fname := "info" + uuid.New().String() + ".json"
	file, err := os.Create(fname)
	if err != nil {
		return "", err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ") // Indent with four spaces

	if err := encoder.Encode(jsonData); err != nil {
		os.Remove(fname)
		return "", err
	}

	return fname, nil
}

func getInfo(jsonFile string) (*DownloadFile, error) {
	file, err := os.Open(jsonFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data DownloadFile
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, err
	}
	data.CancelChan = make(chan bool)

	return &data, nil
}
