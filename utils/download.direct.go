package utils

func DoDownload(Downloads map[string]*DownloadFile, download *DownloadFile) {
	Downloads[download.Url] = download
	download.Resume()
	if download.IsCanceled() || download.Completed {
		delete(Downloads, download.Url)
	}

}
