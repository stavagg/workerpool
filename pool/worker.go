package pool

import (
	"log"
	"runtime/debug"
)

func (p *pool) worker() {

	defer p.wg.Done()
	for task := range p.tasks {
		var pan any
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("pool: task panic recovered: %v\n%s", r, debug.Stack())
				}
			}()
			task()
		}()
		if h := p.postHook; h != nil {
			h(pan)
		}
	}
}
