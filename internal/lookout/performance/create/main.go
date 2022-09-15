package main

import (
	"github.com/G-Research/armada/internal/common/compress"
	"github.com/G-Research/armada/internal/lookout/performance"
	"math/rand"
)

func main() {
	db := performance.OpenDb()

	jobs := performance.CreateJobs(performance.VgisSpec)
	rand.Shuffle(len(jobs), func(i, j int) { jobs[i], jobs[j] = jobs[j], jobs[i] })

	compressor, err := compress.NewZlibCompressor(1024)
	if err != nil {
		panic(err)
	}

	err = performance.SaveBatched(db, jobs, compressor, 10000)
	performance.PanicOnErr(err)
}
