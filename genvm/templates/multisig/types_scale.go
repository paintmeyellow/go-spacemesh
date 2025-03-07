// Code generated by github.com/spacemeshos/go-scale/scalegen. DO NOT EDIT.

// nolint
package multisig

import (
	"github.com/spacemeshos/go-scale"
	"github.com/spacemeshos/go-spacemesh/common/types"
)

func (t *SpawnArguments) EncodeScale(enc *scale.Encoder) (total int, err error) {
	{
		n, err := scale.EncodeStructSlice(enc, t.PublicKeys)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func (t *SpawnArguments) DecodeScale(dec *scale.Decoder) (total int, err error) {
	{
		field, n, err := scale.DecodeStructSlice[types.Hash32](dec)
		if err != nil {
			return total, err
		}
		total += n
		t.PublicKeys = field
	}
	return total, nil
}

func (t *Part) EncodeScale(enc *scale.Encoder) (total int, err error) {
	{
		n, err := scale.EncodeCompact8(enc, uint8(t.Ref))
		if err != nil {
			return total, err
		}
		total += n
	}
	{
		n, err := scale.EncodeByteArray(enc, t.Sig[:])
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func (t *Part) DecodeScale(dec *scale.Decoder) (total int, err error) {
	{
		field, n, err := scale.DecodeCompact8(dec)
		if err != nil {
			return total, err
		}
		total += n
		t.Ref = uint8(field)
	}
	{
		n, err := scale.DecodeByteArray(dec, t.Sig[:])
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}
