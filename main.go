package main

import (
	"context"
	"fmt"

	"github.com/iamnunog/go-nuntius/nuntius"
)

type CounterActor struct {
	count int
}

func (c *CounterActor) Receive(ctx context.Context, msg int) {
	c.count += msg
	fmt.Println("Counter:", c.count)
}

func main() {
	sys := nuntius.NewSystem("example")()

	counter1 := nuntius.Spawn(
		sys,
		func() nuntius.Actor[int] {
			return &CounterActor{}
		},
		"/user/counter1",
	)

	counter2 := nuntius.Spawn(
		sys,
		func() nuntius.Actor[int] {
			return &CounterActor{}
		},
		"/user/counter2",
	)

	counter1.Tell(context.Background(), 5)
	counter2.Tell(context.Background(), 10)
	nuntius.TellAny(
		context.Background(),
		[]*nuntius.ActorRef[int]{counter1, counter2},
		3,
	)

	sys.Stop()
}
