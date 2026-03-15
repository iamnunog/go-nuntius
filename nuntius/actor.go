package nuntius

import (
	"context"
	"fmt"
)

type Receiver[T any] interface {
	Receive(ctx context.Context, msg T)
}

type Sender[T any] interface {
	Tell(ctx context.Context, msg T) error
	Ask(ctx context.Context, msg T) (any, error)
}

type Broadcaster[T any] interface {
	TellAny(ctx context.Context, msg T) error
}

type Actor[T any] interface {
	Receiver[T]
}

type ActorRef[T any] struct {
	mailbox chan T
	path    string
	sys     *System
}

func (a *ActorRef[T]) Tell(ctx context.Context, msg T) error {
	a.sys.addMsg() // Track before sending
	select {
	case a.mailbox <- msg:
		fmt.Printf("[Tell] Sending to %s: %v\n", a.path, msg)
		return nil
	case <-ctx.Done():
		a.sys.doneMsg() // Untrack if cancelled
		return ctx.Err()
	}
}

func (a *ActorRef[T]) Ask(ctx context.Context, msg T) (any, error) {
	err := a.Tell(ctx, msg)
	return nil, err
}

func TellAny[T any](ctx context.Context, actors []*ActorRef[T], msg T) error {
	for _, ref := range actors {
		if err := ref.Tell(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}
