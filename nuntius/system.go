package nuntius

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type System struct {
	name      string
	transport Transport
	executer  Executer

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Track pending messages
	pending int64
	done    chan struct{}
	mu      sync.Mutex
}

type systemBuilder func() *System

func (b systemBuilder) WithTransport(t Transport) systemBuilder {
	return func() *System {
		s := b()
		s.transport = t
		return s
	}
}

func (b systemBuilder) WithExecuter(e Executer) systemBuilder {
	return func() *System {
		s := b()
		s.executer = e
		return s
	}
}

func NewSystem(name string) systemBuilder {
	return func() *System {
		ctx, cancel := context.WithCancel(context.Background())
		return &System{
			name:   name,
			ctx:    ctx,
			cancel: cancel,
			done:   make(chan struct{}),
		}
	}
}

func (s *System) addMsg() {
	atomic.AddInt64(&s.pending, 1)
}

func (s *System) doneMsg() {
	if atomic.AddInt64(&s.pending, -1) == 0 {
		s.mu.Lock()
		if s.done != nil {
			close(s.done)
			s.done = nil // Prevent double-close
		}
		s.mu.Unlock()
	}
}

func (s *System) StopWithTimeout(timeout time.Duration) {
	if atomic.LoadInt64(&s.pending) > 0 {
		select {
		case <-s.done:
		case <-time.After(timeout):
			log.Println("Shutdown timeout, forcing exit")
		}
	}
	s.cancel()
	s.wg.Wait()
}

func (s *System) Stop() {
	// Wait for all pending messages to complete
	pending := atomic.LoadInt64(&s.pending)
	if pending > 0 {
		<-s.done
	}

	s.cancel()
	s.wg.Wait()
}

// Spawn creates a new actor using the factory and returns an ActorRef
func Spawn[T any](s *System, factory func() Actor[T], path string) *ActorRef[T] {
	mailbox := make(chan T, 32)
	actor := factory()

	ref := &ActorRef[T]{mailbox: mailbox, path: path, sys: s}

	s.wg.Go(func() {
		defer s.wg.Done()

		defer func() {
			if r := recover(); r != nil {
				log.Printf("Actor panic: %v", r)
				s.doneMsg()
			}
		}()

		for {
			select {
			case msg, ok := <-mailbox:
				if !ok {
					return
				}
				fmt.Printf("[nuntius] Message to %s: %v\n", path, msg)
				actor.Receive(s.ctx, msg)
				s.doneMsg() // Signal message processed
			case <-s.ctx.Done():
				return
			}
		}
	})

	return ref
}
