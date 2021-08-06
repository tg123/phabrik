package transport

import (
	"context"
	"fmt"
	"sync"
)

type RequestTable struct {
	table sync.Map
}

type PendingRequest struct {
	parent *RequestTable
	id     MessageId
	ch     chan *ByteArrayMessage
	close  sync.Once
}

func (r *PendingRequest) Close() error {
	pr, ok := r.parent.table.LoadAndDelete(r.id)
	if !ok {
		return nil
	}

	pr.(*PendingRequest).close.Do(func() {
		ch := pr.(*PendingRequest).ch
		close(ch)
	})

	return nil
}

func (r *PendingRequest) Wait(ctx context.Context) (*ByteArrayMessage, error) {
	defer r.Close()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case reply := <-r.ch:
		if reply == nil {
			return nil, fmt.Errorf("operation cancelled")
		}
		return reply, nil
	}
}

func (r *RequestTable) Close() error {
	r.table.Range(func(key, value interface{}) bool {
		value.(*PendingRequest).Close()
		return true
	})

	return nil
}

func (r *RequestTable) Put(msg *Message) *PendingRequest {
	id := msg.Headers.Id

	p := &PendingRequest{
		parent: r,
		id:     id,
		ch:     make(chan *ByteArrayMessage),
	}
	r.table.Store(id, p)
	return p
}

func (r *RequestTable) Feed(msg *ByteArrayMessage) bool {
	id := msg.Headers.RelatesTo
	if !id.IsEmpty() {
		pr, ok := r.table.LoadAndDelete(id)

		if ok {
			pr.(*PendingRequest).ch <- msg
			return true
		}
	}

	return false
}
