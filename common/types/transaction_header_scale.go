// Code generated by github.com/spacemeshos/go-scale/scalegen. DO NOT EDIT.

package types

import (
	"github.com/spacemeshos/go-scale"
)

func (t *Nonce) EncodeScale(enc *scale.Encoder) (total int, err error) {
	if n, err := scale.EncodeCompact64(enc, uint64(t.Counter)); err != nil {
		return total, err
	} else {
		total += n
	}
	if n, err := scale.EncodeCompact8(enc, uint8(t.Bitfield)); err != nil {
		return total, err
	} else {
		total += n
	}
	return total, nil
}

func (t *Nonce) DecodeScale(dec *scale.Decoder) (total int, err error) {
	if field, n, err := scale.DecodeCompact64(dec); err != nil {
		return total, err
	} else {
		total += n
		t.Counter = uint64(field)
	}
	if field, n, err := scale.DecodeCompact8(dec); err != nil {
		return total, err
	} else {
		total += n
		t.Bitfield = uint8(field)
	}
	return total, nil
}