# File Server and Download Manager

## Table of Contents

- [Introduction](#introduction)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation and Usage](#installation-and-usage)
- [Endpoints](#endpoints)
  - [Serve Static Files](#serve-static-files)
  - [Delete Files](#delete-files)
  - [Rename Files](#rename-files)
  - [Download Files](#download-files)
  - [Cancel Downloads](#cancel-downloads)
  - [Get Download Status](#get-download-status)
  - [Yt-Dlp Support](#Yt-Dlp Support)
- [Notes](#notes)
- [License](#license)

## Introduction

This Go code provides a simple web server for serving static files from a specified directory and includes a download manager with the ability to add, cancel, and monitor file downloads.

## Features

1. **File Server:** You can serve static files from a directory of your choice.
2. **Delete Files:** You can delete files from the server.
3. **Rename Files:** You can rename files on the server.
4. **Download Manager:** You can add, cancel, and monitor file downloads.
5. **Status Endpoint:** You can check the status of ongoing downloads, including the download progress, download speed, and percentage completion.

## Prerequisites

Make sure you have Go installed on your system.

## Installation and Usage

1. Clone this repository to your local machine:

   ```bash
   git clone <repository-url>
2. Navigate to the project directory:

   ```bash
   cd <project-directory>
3. Build and run the code:

   ```bash
   go run main.go
# Endpoints

## Serve Static Files
The server serves static files from the `./static` directory. You can access these files by visiting http://localhost:8080/fs<file-url>


## Delete Files
To delete a file, send a DELETE request to the `/delete` endpoint with the file parameter set to the file you want to delete.

Example:

    curl -X DELETE http://localhost:8080/delete?file=<file-name>

## Download Files
To add a file for download, send a POST request to the `/download` endpoint with the file_name and url parameters. This will add the file to the download queue.

Example:

    curl -X POST -d "file_name=<file-name>&url=<file-url>" http://localhost:8080/download

## Cancel Downloads
To cancel an ongoing download, send a PUT request to the `/cancel` endpoint with the url parameter set to the URL of the file you want to cancel.

Example:

    curl -X PUT -d "url=<file-url>" http://localhost:8080/cancel

## Get Download Status
You can check the status of ongoing downloads by sending a GET request to the `/status` endpoint. This will return a JSON response with details about the ongoing downloads, including file size, downloaded bytes, percentage completion, download speed, file name, and URL.

Example:

    curl http://localhost:8080/status


## Yt-Dlp Support
Example:

    curl -X POST -d "url=<file-url>" http://localhost:8080/yt-dlp


## Notes
- The server uses a default directory of ./static for serving files. You can change this directory in the main function.

- The download manager can be useful for adding and monitoring downloads of large files.

- The code provides basic error handling, but you may want to enhance it for production use.

- Remember to adjust the server's address and port as needed in the code (:8080 in this example).

# License
This code is provided under the MIT License.