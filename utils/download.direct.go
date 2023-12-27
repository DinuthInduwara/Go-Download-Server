package utils

func DoDownload(Downloads map[string]*DownloadFile, download *DownloadFile) {
	Downloads[download.Url] = download
	if !download.Resume() {
		delete(Downloads, download.Url)
	}

}
