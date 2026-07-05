package realtime

import (
	"context"

	"github.com/boxify/api-go/internal/domain/types"
)

type ForwardOptions struct {
	StopEvents map[string]struct{}
}

func Forward(ctx context.Context, sub Subscription, out chan<- types.Event, opts ForwardOptions) error {
	defer close(out)
	defer sub.Close(context.Background())

	stopEvents := opts.StopEvents
	if len(stopEvents) == 0 {
		stopEvents = map[string]struct{}{
			types.EventTypeDone: {},
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-sub.Events():
			if !ok {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- event:
			}
			if _, stop := stopEvents[event.EventName()]; stop {
				return nil
			}
		}
	}
}
