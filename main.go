package main

import (
	"fmt"
	"github.com/virepri/r18-ripper/downloader"
	"github.com/virepri/r18-ripper/playlist"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: r18rip <chunklist> <destination>")
		return
	}

	playlistURI, err := url.Parse(os.Args[1])

	if err != nil {
		fmt.Println("Supplied URI was invalid:", err.Error())
		return
	}

	plHandler := &playlist.Handler{
		URI: *playlistURI,
	}

	chunks, count, err := plHandler.GetChunks()

	if err != nil {
		panic(err)
	}

	os.MkdirAll(os.Args[2], os.ModePerm | os.ModeDir)

	absPath, err := filepath.Abs(os.Args[2])

	if err != nil {
		panic(err)
	}

	err = plHandler.HLSPlaylist.WritePlaylistToFile(path.Join(absPath, "chunklist.m3u8"))

	if err != nil {
		panic(err)
	}

	var chunkCount uint64

	pConfig := downloader.PipelineConfig{
		WorkingDirectory: os.Args[2],
		ChunkCount: 	  &chunkCount,
		ChunkWG:          &sync.WaitGroup{},
		ChunkStatus:      &sync.Map{},
		Steps: []downloader.StepRunner{downloader.DownloadStep, downloader.WriteStep},
	}

	pConfig.ChunkWG.Add(count)

	for i := runtime.NumCPU(); i > 0; i -- {
		go downloader.PipelineExecutor(pConfig, chunks)
	}

	go func() {
		lastChunk := uint64(0)

		for {
			time.Sleep(time.Second / 2)

			cc := atomic.LoadUint64(pConfig.ChunkCount)
			if cc != lastChunk {
				lastChunk = cc
				fmt.Printf("Completed %d/%d chunks\n", cc, count)

				for k, v := range pConfig.Steps {
					fName := runtime.FuncForPC(reflect.ValueOf(v).Pointer()).Name()
					statusptr, _ := pConfig.ChunkStatus.Load(k)
					fmt.Printf("Chunks in %s: %d\n", fName[strings.LastIndex(fName, ".")+1:], atomic.LoadUint64(statusptr.(*uint64)))
				}

				fmt.Println()
			}
		}
	}()

	pConfig.ChunkWG.Wait()

	fmt.Printf("\nAll chunks have finished downloading. The playlist is available as %s. Stick it through ffmpeg to combine into a single container of your choice. Enjoy your wank!", path.Join(os.Args[2], "chunklist.m3u8"))
}
