package main

import (
	"sync"
)

type Counter struct {
	mu sync.Mutex
	n  int64
}

func (c *Counter) Inc() {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
}

func (c *Counter) Value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
