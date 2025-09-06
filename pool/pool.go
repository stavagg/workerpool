package pool

import (
	"errors"
	"sync"
)

type state int

const (
	stateRunning state = iota
	stateStopping
	stateStopped
)

var ErrPoolStopped = errors.New("pool stopped")
var ErrQueueFull = errors.New("queue is full")

type Pool interface {
	Submit(func()) error
	Stop() error
}
type pool struct {
	tasks chan func()

	mu sync.Mutex
	st state

	wg   sync.WaitGroup
	once sync.Once

	postHook func(any)
}

func New(numWorkers, queueSize int, hook ...func(any)) (Pool, error) {
	if numWorkers <= 0 {
		return nil, errors.New("numWorkers must be > 0")
	}
	if queueSize < 0 {
		return nil, errors.New("queueSize must be >= 0")
	}

	p := &pool{
		tasks:    make(chan func(), queueSize),
		st:       stateRunning,
		postHook: nil,
	}

	if len(hook) > 0 {
		p.postHook = hook[0]
	}

	for i := 0; i < numWorkers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p, nil
}

func (p *pool) Submit(task func()) error {
	if task == nil {
		return errors.New("task is nil")
	}

	p.mu.Lock()
	stopped := p.st != stateRunning
	p.mu.Unlock()
	if stopped {
		return ErrPoolStopped
	}

	select {
	case p.tasks <- task:
		return nil
	default:
		return ErrQueueFull
	}

	return nil
}
