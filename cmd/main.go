package main

import (
	"fmt"
	"time"

	"example.com/pool-demo/pool"
)

func main() {
	p, err := pool.New(2, 4)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 5; i++ {
		n := i
		if err := p.Submit(func() {
			fmt.Println("task", n)
			time.Sleep(50 * time.Millisecond)
		}); err != nil {
			fmt.Println("submit error:", err)
		}
	}

	_ = p.Stop()
}
