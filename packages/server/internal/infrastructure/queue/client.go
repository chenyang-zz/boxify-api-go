package queue

import (
	"context"

	"github.com/boxify/api-go/internal/domain/types"
)

type EnqueueOptions struct {
	Queue    types.QueueName
	MaxRetry *int
}

type EnqueueOption func(*EnqueueOptions)

type TaskInfo struct {
	ID    string
	Name  types.TaskName
	Queue types.QueueName
}

type Producer interface {
	Enqueue(ctx context.Context, task *types.Task, opts ...EnqueueOption) (*TaskInfo, error)
	Close() error
}

type Handler interface {
	HandleTask(ctx context.Context, task *types.Task) error
}

type HandlerFunc func(ctx context.Context, task *types.Task) error

func (f HandlerFunc) HandleTask(ctx context.Context, task *types.Task) error {
	return f(ctx, task)
}

type Router interface {
	Handle(name types.TaskName, handler Handler)
}

func WithQueue(queue types.QueueName) EnqueueOption {
	return func(opts *EnqueueOptions) {
		opts.Queue = queue
	}
}

func WithMaxRetry(maxRetry int) EnqueueOption {
	return func(opts *EnqueueOptions) {
		opts.MaxRetry = &maxRetry
	}
}

func NewEnqueueOptions(task *types.Task, opts ...EnqueueOption) *EnqueueOptions {
	options := &EnqueueOptions{}
	if task != nil {
		options.Queue = task.Queue
	}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	return options
}
