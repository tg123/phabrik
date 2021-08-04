package federation

import "github.com/tg123/phabrik/serialization"

type NodeIdRange struct {
	Begin NodeID
	End   NodeID
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
