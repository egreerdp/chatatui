package hub

import (
	"log/slog"
	"sync"
)

type broadcastJob struct {
	message []byte
	clients []*Client
}

type BroadcastPool struct {
	workers     []chan *broadcastJob
	workerCount int
	wg          sync.WaitGroup
}

func NewBroadcastPool(workerCount int) *BroadcastPool {
	bp := &BroadcastPool{
		workers:     make([]chan *broadcastJob, workerCount),
		workerCount: workerCount,
	}

	for i := range workerCount {
		bp.workers[i] = make(chan *broadcastJob, 16)
	}

	return bp
}

func (bp *BroadcastPool) Start() {
	for i := range bp.workerCount {
		bp.wg.Go(func() {
			bp.worker(bp.workers[i])
		})
	}
}

func (bp *BroadcastPool) worker(jobChan chan *broadcastJob) {
	for job := range jobChan {
		for _, client := range job.clients {
			client.SendRaw(job.message)
		}
	}
}

// Submit shards the job's client list across all workers so that a single
// large room fans out in parallel rather than being handled by one worker.
func (bp *BroadcastPool) Submit(job *broadcastJob) {
	if len(job.clients) == 0 {
		return
	}

	chunkSize := max((len(job.clients)+bp.workerCount-1)/bp.workerCount, 1)
	for i, workerIdx := 0, 0; i < len(job.clients); i, workerIdx = i+chunkSize, workerIdx+1 {
		end := min(i+chunkSize, len(job.clients))
		sub := &broadcastJob{
			message: job.message,
			clients: job.clients[i:end],
		}
		select {
		case bp.workers[workerIdx] <- sub:
		default:
			slog.Warn("broadcast worker pool full, dropping message", "worker_index", workerIdx)
		}
	}
}

func (bp *BroadcastPool) Shutdown() {
	for _, workerChan := range bp.workers {
		close(workerChan)
	}

	bp.wg.Wait()
}
