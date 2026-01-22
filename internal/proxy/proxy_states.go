package proxy

// crashLoop sets state to FAILED and records the error message.
func (p *proxy) crashLoop(err error) {
	p.state = StateFailed
	p.lastError = err.Error()
	if p.lastRestartReason == "" {
		p.lastRestartReason = "crash_loop"
	}
	if p.logger != nil {
		p.logger.Error("crash loop detected", err)
	}
}
