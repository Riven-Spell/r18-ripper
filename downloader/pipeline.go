package downloader

import (
	"github.com/virepri/r18-ripper/playlist"
	"sync"
	"sync/atomic"
)

type PipelineConfig struct {
	WorkingDirectory string
	ChunkStatus      *sync.Map // map of int to atomic uint64
	ChunkWG          *sync.WaitGroup
	ChunkCount		 *uint64
	// the first step should always expect a ChunkListReturn input.
	// the last step output does not matter.
	Steps []StepRunner
}

// performs the step
type StepRunner func(input interface{}, chunk playlist.ChunkListReturn, config PipelineConfig) interface{}

// should be ran as a goroutine
func PipelineExecutor(cfg PipelineConfig, input chan playlist.ChunkListReturn) {
	for k := range cfg.Steps {
		var count uint64

		cfg.ChunkStatus.Store(k, &count)
	}

	for v := range input {
		var acc interface{}
		for k, f := range cfg.Steps {
			statusLoc, _ := cfg.ChunkStatus.Load(k)
			atomic.AddUint64(statusLoc.(*uint64), 1)
			acc = f(acc, v, cfg)

			if err, ok := acc.(error); ok {
				// todo: better pipeline error handling
				panic(err)
			}

			atomic.StoreUint64(statusLoc.(*uint64), atomic.LoadUint64(statusLoc.(*uint64)) - 1)
		}

		FinishStep(cfg)
	}
}

// Finishes the pipeline
func FinishStep(cfg PipelineConfig) interface{} {
	atomic.AddUint64(cfg.ChunkCount, 1)
	cfg.ChunkWG.Done()

	return nil
}