package pool

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubmitExecutesTasks(t *testing.T) {
	p, err := New(4, 8)
	if err != nil {
		t.Fatal(err)
	}

	var c int64
	const n = 100
	done := make(chan struct{})

	go func() {
		for i := 0; i < n; i++ {
			if err := p.Submit(func() { atomic.AddInt64(&c, 1) }); err != nil {
				t.Fatalf("unexpected submit error: %v", err)
			}
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("submit timed out")
	}

	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt64(&c); got != n {
		t.Fatalf("want %d, got %d", n, got)
	}
}

func TestErrorAfterStop(t *testing.T) {
	p, _ := New(1, 1)
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := p.Submit(func() {}); !errors.Is(err, ErrPoolStopped) {
		t.Fatalf("want ErrPoolStopped, got %v", err)
	}
}

func TestPanicDoesNotKillWorker(t *testing.T) {
	p, _ := New(2, 4)
	done := make(chan struct{})

	_ = p.Submit(func() { panic("boom") })

	const n = 10
	var c int64
	for i := 0; i < n; i++ {
		if err := p.Submit(func() { atomic.AddInt64(&c, 1) }); err != nil {
			t.Fatalf("submit err: %v", err)
		}
	}

	go func() {
		_ = p.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("stop timed out")
	}
	if atomic.LoadInt64(&c) != n {
		t.Fatalf("worker died after panic; want %d done, got %d", n, c)
	}
}

func TestQueueFullError(t *testing.T) {
	p, err := New(1, 1)
	if err != nil {
		t.Fatal(err)
	}

	release := make(chan struct{})
	if err := p.Submit(func() {
		<-release
	}); err != nil {
		t.Fatal(err)
	}

	if err := p.Submit(func() {}); err != nil {
		t.Fatalf("unexpected error for buffered task: %v", err)
	}

	if err := p.Submit(func() {}); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("want ErrQueueFull, got %v", err)
	}

	close(release)
	if err := p.Stop(); err != nil {
		t.Fatal(err)
	}
}
