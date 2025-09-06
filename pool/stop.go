package pool

func (p *pool) Stop() error {

	p.once.Do(func() {
		p.mu.Lock()
		if p.st == stateRunning {
			p.st = stateStopping
		}
		p.mu.Unlock()

		close(p.tasks)

		p.wg.Wait()

		p.mu.Lock()
		p.st = stateStopped
		p.mu.Unlock()
	})
	return nil
}
