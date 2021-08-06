package federation

import (
	"context"
	"log"
	"time"

	"github.com/tg123/phabrik/transport"
)

func (s *SiteNode) Bootstrap(ctx context.Context) error {
	if s.phase >= NodePhaseJoining {
		return nil
	}

	for {
		for _, seed := range s.seedNodes {
			if err := s.votePing(seed.Id); err != nil {
				log.Printf("send vote ping to %v failed %v", seed.Address, err)
			}
		}

		select {
		case <-s.phaseChanged:
			if s.phase >= NodePhaseJoining {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * 30): // TODO config
			// continue
		}
	}

}

func (s *SiteNode) votePing(id NodeID) error {
	return s.SendOneWay(id, &transport.Message{
		Headers: transport.MessageHeaders{
			Actor:  transport.MessageActorTypeFederation,
			Action: "VotePing",
		},
	})
}
