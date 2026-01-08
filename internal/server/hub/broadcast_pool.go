package hub

import (
	"log"
	"sync"
	"sync/atomic"
)

type broadcastJob struct {
	message []byte
	clients []*Client
}

type BroadcastPool struct {
	workers     []chan *broadcastJob
	workerCount int
	nextWorker  uint32
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

func (bp *BroadcastPool) Submit(job *broadcastJob) {
	workerIdx := atomic.AddUint32(&bp.nextWorker, 1) % uint32(bp.workerCount)

	select {
	case bp.workers[workerIdx] <- job:
	default:
		log.Println("Warning: Broadcast worker pool full, dropping message")
	}
}

func (bp *BroadcastPool) Shutdown() {
	for _, workerChan := range bp.workers {
		close(workerChan)
	}

	bp.wg.Wait()
}
