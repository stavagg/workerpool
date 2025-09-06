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

	// Задача с паникой
	_ = p.Submit(func() { panic("boom") })

	// Нормальные задачи после паники
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

func TestBackpressure(t *testing.T) {
	// Очередь размера 1 и один воркер: вторая отправка должна блокироваться,
	// пока воркер не возьмёт первую задачу.
	p, _ := New(1, 1)
	started := make(chan struct{})
	release := make(chan struct{})

	// Первая задача держит воркера
	if err := p.Submit(func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatal(err)
	}

	// Вторая задача должна заблокироваться до освобождения места
	blockedDone := make(chan struct{})
	go func() {
		_ = p.Submit(func() {})
		close(blockedDone)
	}()

	// Убедимся, что реально блокируется
	select {
	case <-blockedDone:
		t.Fatal("second submit should be blocked")
	case <-time.After(100 * time.Millisecond):
		// ок, блокируется
	}

	// Освобождаем воркера — теперь вторая задача должна пройти
	close(release)
	select {
	case <-blockedDone:
		// ок
	case <-time.After(1 * time.Second):
		t.Fatal("second submit did not complete after release")
	}

	_ = p.Stop()
}
