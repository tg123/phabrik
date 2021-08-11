package federation

import (
	"context"
	"log"
	"sync"
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

func (s *SiteNode) Discover(ctx context.Context) ([]PartnerNodeInfo, error) {

	discovered := make(map[NodeID]PartnerNodeInfo)

	for _, seed := range s.seedNodes {
		discovered[seed.Id] = PartnerNodeInfo{
			Instance: NodeInstance{
				Id: seed.Id,
			},
			Address: seed.Address,
		}
	}

	round := 5

	for {
		var wg sync.WaitGroup
		wg.Add(len(discovered))

		for _, partner := range discovered {
			go func(p *PartnerNodeInfo) {
				defer wg.Done()
				if err := s.votePing(p.Instance.Id); err != nil {
					log.Printf("send vote ping to %v failed %v", p.Address, err)
				}
			}(&partner)
		}

		wg.Wait()

		time.Sleep(1 * time.Second) // wait for voteping reply

		found := false
		for _, partner := range s.KnownPartnerNodes(func(PartnerNodeInfo) bool { return true }) {

			if partner.Instance.Id == s.instance.Id {
				continue
			}

			p := discovered[partner.Instance.Id]

			if partner.Instance.InstanceId > p.Instance.InstanceId {
				found = true
				discovered[partner.Instance.Id] = partner
			}
		}

		if !found {
			round -= 1
		}

		if round < 0 {
			break
		}
	}

	parteners := make([]PartnerNodeInfo, 0, len(discovered))
	for _, partner := range discovered {
		parteners = append(parteners, partner)
	}

	return parteners, nil
}

func (s *SiteNode) votePing(id NodeID) error {
	return s.SendOneWay(id, &transport.Message{
		Headers: transport.MessageHeaders{
			Actor:  transport.MessageActorTypeFederation,
			Action: "VotePing",
		},
	})
}
