package federation

import (
	"context"
	"fmt"

	"github.com/tg123/phabrik/common"
	"github.com/tg123/phabrik/serialization"
	"github.com/tg123/phabrik/transport"
)

func (s *SiteNode) Join(ctx context.Context) error {
	msg := &transport.Message{
		Headers: transport.MessageHeaders{
			Actor:  transport.MessageActorTypeFederation,
			Action: "NeighborhoodQueryRequest",
		},
	}

	msg.Body = &struct {
		Time common.StopwatchTime
	}{
		1,
	}

	reply, err := s.Route(ctx, s.instance.Id, msg)
	if err != nil {
		return err
	}

	var global GlobalLease

	serialization.Unmarshal(reply.Body, &global)

	fmt.Println("reply", global)

	return nil
}
