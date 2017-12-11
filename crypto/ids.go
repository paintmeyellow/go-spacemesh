package crypto

import (
	"fmt"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	"strings"
)

type Identifier interface {
	String() string
	Bytes() []byte
	PeerId() peer.ID
	Pretty() string
	PublicKey() (PublicKeylike, error)
}

// An Id is derived from a public key
// Used to for node ids and account ids and may be used by other types
type Id struct {
	peer.ID
}

func (id Id) String() string {
	return peer.IDB58Encode(id.ID)
}

func (id Id) Bytes() []byte {
	return []byte(id.ID)
}

func (id Id) PeerId() peer.ID {
	return id.ID
}

// get public key from key
// untested - dragons!!
func (id Id) PublicKey() (PublicKeylike, error) {
	pubKey, err := id.ID.ExtractEd25519PublicKey()
	if err != nil {
		return nil, err
	}
	return &PublicKey{pubKey}, nil
}

func (id Id) Pretty() string {

	pid := id.String()

	// get rid of mh first 2 chars which are always Qm
	if strings.HasPrefix(pid, "Qm") {
		pid = pid[2:]
	}

	maxRunes := 6
	if len(pid) < maxRunes {
		maxRunes = len(pid)
	}
	return fmt.Sprintf("<ID %s>", pid[:maxRunes])
}

// create a new ID from a b58 encoded string
func NewIdentifier(b58 string) (Identifier, error) {
	id, err := peer.IDB58Decode(b58)
	return &Id{id}, err
}