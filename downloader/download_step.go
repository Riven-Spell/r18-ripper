package downloader

import (
	"fmt"
	"github.com/virepri/r18-ripper/playlist"
	"io/ioutil"
	"net/http"
	"time"
)

// expects a chunklistreturn, puts out a blob
func DownloadStep(_ interface{}, chunk playlist.ChunkListReturn, cfg PipelineConfig) interface{} {
	retryWait := time.Minute / 4
	retryDownload:
	resp, err := http.Get(chunk.Url)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			fmt.Println("Chunk", chunk.ChunkID, "is being invisibly rate limited (HTTP 404). Waiting...")
		} else {
			fmt.Printf("Chunk %d is throwing an error: HTTP %d. This is assumed to be invisible rate limiting for now.", chunk.ChunkID, resp.StatusCode)
		}
		time.Sleep(retryWait)
		retryWait *= 2
		goto retryDownload
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	return body
}
