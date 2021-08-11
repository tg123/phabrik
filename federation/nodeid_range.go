package federation

import (
	"github.com/tg123/phabrik/serialization"
)

type NodeIdRange struct {
	Begin NodeID
	End   NodeID
}

func (r *NodeIdRange) Contains(id NodeID) bool {
	s := uint128(r.Begin)
	e := uint128(r.End)
	i := uint128(id)

	if s.cmp(e) > 0 {

		if s.cmp(i) <= 0 && u128max.cmp(e) <= 0 {
			return true
		}

		if u128zero.cmp(i) <= 0 && i.cmp(e) <= 0 {
			return true
		}

		return false
	}

	return s.cmp(i) <= 0 && i.cmp(e) <= 0
}

var _ serialization.CustomMarshaler = (*NodeIdRange)(nil)

func (r *NodeIdRange) Marshal(s serialization.Encoder) error {
	if err := r.Begin.Marshal(s); err != nil {
		return err
	}

	if err := r.End.Marshal(s); err != nil {
		return err
	}

	return nil
}

func (r *NodeIdRange) Unmarshal(meta serialization.FabricSerializationType, s serialization.Decoder) error {
	if err := r.Begin.Unmarshal(meta, s); err != nil {
		return err
	}

	meta, err := s.ReadTypeMeta()
	if err != nil {
		return err
	}

	if err := r.End.Unmarshal(meta, s); err != nil {
		return err
	}

	return nil
}
