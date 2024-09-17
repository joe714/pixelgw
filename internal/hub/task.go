package hub

// We want certain configuration tasks to run in the context of the hub
// or channel's main goroutine.
type task struct {
	fn     func() error
	done   chan bool
	result error
}

func (t *task) run() {
	err := t.fn()
	if err != nil {
		t.result = err
	}
	t.done <- true
}

func RunTask(q chan *task, fn func() error) error {
	t := task{
		fn:     fn,
		done:   make(chan bool),
		result: nil,
	}
	q <- &t
	<-t.done
	return t.result
}
