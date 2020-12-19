package downloader

import (
	"github.com/virepri/r18-ripper/playlist"
	"io/ioutil"
	"net/url"
	"path"
)

// expects a blob, outputs nothing-- is the end of the pipeline.
func WriteStep(input interface{}, chunk playlist.ChunkListReturn, cfg PipelineConfig) interface{} {
	uri, _ := url.Parse(chunk.Url)
	fname := path.Base(uri.Path)

	body := input.([]byte)

	fName := path.Join(cfg.WorkingDirectory, fname)

	err := ioutil.WriteFile(fName, body, 0644)

	return err
}
