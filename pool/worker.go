package pool

import (
	"log"
	"runtime/debug"
)

func (p *pool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("pool: task panic recovered: %v\n%s", r, debug.Stack())
				}
			}()
			task()
		}()

	}
}
