// Package types defines the types used by go-spacemesh consensus algorithms and structs
package types

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/spacemeshos/go-spacemesh/common/util"
	"github.com/spacemeshos/go-spacemesh/log"
)

const (
	// LayerIDSize in bytes.
	LayerIDSize = 4

	// GenesisBeacon is the hex value of the beacon used during genesis.
	GenesisBeacon = "0xaeebad4a796fcc2e15dc4c6061b45ed9b373f26adfc798ca7d2d8cc58182718e" // sha256("genesis")

	// genesisBallotIDHex is the genesis ballot ID in hex.
	genesisBallotIDHex = "0x7a68f37b1a1903c9b9d428c3bdb0a8188c6b7d888ce63166dc97a9826105f417"

	// genesisBlockIDHex is the genesis block ID in hex.
	genesisBlockIDHex = "0xbab7a6d8efcb406d121199f93cc7997cf9a97ea83262e72548fe9389a9ac88ae"
)

var (
	genesisLayer   *Layer
	layersPerEpoch uint32
	// effectiveGenesis marks when actual proposals would start being created in the network. It takes into account
	// the first genesis epoch and the following epoch in which ATXs are published.
	effectiveGenesis uint32

	// EmptyLayerHash is the layer hash for an empty layer.
	EmptyLayerHash = Hash32{}

	// GenesisBallotID is the BallotID for the genesis ballot.
	GenesisBallotID = BallotID(HexToHash32(genesisBallotIDHex).ToHash20())
	// GenesisBlockID is the BlockID for the genesis block.
	GenesisBlockID = BlockID(HexToHash32(genesisBlockIDHex).ToHash20())
)

// SetLayersPerEpoch sets global parameter of layers per epoch, all conversions from layer to epoch use this param.
func SetLayersPerEpoch(layers uint32) {
	atomic.StoreUint32(&layersPerEpoch, layers)
	atomic.StoreUint32(&effectiveGenesis, layers*2-1)
}

// GetLayersPerEpoch returns number of layers per epoch.
func GetLayersPerEpoch() uint32 {
	return atomic.LoadUint32(&layersPerEpoch)
}

// GenesisLayer returns the genesis layer.
func GenesisLayer() *Layer {
	if genesisLayer == nil {
		InitGenesisData()
	}
	return genesisLayer
}

// InitGenesisData generate the genesis data.
func InitGenesisData() {
	p := &Proposal{
		InnerProposal: InnerProposal{
			Ballot: Ballot{
				InnerBallot: InnerBallot{
					LayerIndex: GetEffectiveGenesis(),
					EpochData: &EpochData{
						Beacon: HexToBeacon(GenesisBeacon),
					},
				},
				ballotID: GenesisBallotID,
			},
		},
		proposalID: ProposalID(GenesisBlockID),
	}
	genesisLayer = NewLayer(GetEffectiveGenesis())
	genesisLayer.AddProposal(p)
}

// GetEffectiveGenesis returns when actual proposals would be created.
func GetEffectiveGenesis() LayerID {
	return NewLayerID(atomic.LoadUint32(&effectiveGenesis))
}

// NewLayerID creates LayerID from uint32.
func NewLayerID(value uint32) LayerID {
	return LayerID{Value: value}
}

// LayerID is representing a layer number. Zero value is safe to use, and means 0.
// Internally it is a simple wrapper over uint32 and should be considered immutable
// the same way as any integer.
type LayerID struct {
	// NOTE(dshulyak) it is made public for compatibility with encoding library.
	// Don't modify it directly, as it will likely to be made private in the future.
	Value uint32
}

// GetEpoch returns the epoch number of this LayerID.
func (l LayerID) GetEpoch() EpochID {
	return EpochID(l.Value / GetLayersPerEpoch())
}

// Add layers to the layer. Panics on wraparound.
func (l LayerID) Add(layers uint32) LayerID {
	nl := l.Value + layers
	if nl < l.Value {
		panic("layer_id wraparound")
	}
	l.Value = nl
	return l
}

// Sub layers from the layer. Panics on wraparound.
func (l LayerID) Sub(layers uint32) LayerID {
	if layers > l.Value {
		panic("layer_id wraparound")
	}
	l.Value -= layers
	return l
}

// OrdinalInEpoch returns layer ordinal in epoch.
func (l LayerID) OrdinalInEpoch() uint32 {
	return l.Value % GetLayersPerEpoch()
}

// FirstInEpoch returns whether this LayerID is first in epoch.
func (l LayerID) FirstInEpoch() bool {
	return l.OrdinalInEpoch() == 0
}

// Mul layer by the layers. Panics on wraparound.
func (l LayerID) Mul(layers uint32) LayerID {
	if l.Value == 0 {
		return l
	}
	nl := l.Value * layers
	if nl/l.Value != layers {
		panic("layer_id wraparound")
	}
	l.Value = nl
	return l
}

// Uint32 returns the LayerID as a uint32.
func (l LayerID) Uint32() uint32 {
	return l.Value
}

// Before returns true if this layer is lower than the other.
func (l LayerID) Before(other LayerID) bool {
	return l.Value < other.Value
}

// After returns true if this layer is higher than the other.
func (l LayerID) After(other LayerID) bool {
	return l.Value > other.Value
}

// Difference returns the difference between current and other layer.
func (l LayerID) Difference(other LayerID) uint32 {
	if other.Value > l.Value {
		panic(fmt.Sprintf("other (%d) must be before or equal to this layer (%d)", other.Value, l.Value))
	}
	return l.Value - other.Value
}

// Field returns a log field. Implements the LoggableField interface.
func (l LayerID) Field() log.Field { return log.Uint32("layer_id", l.Value) }

// String returns string representation of the layer id numeric value.
func (l LayerID) String() string {
	return strconv.FormatUint(uint64(l.Value), 10)
}

// NodeID contains a miner's two public keys.
type NodeID struct {
	// Key is the miner's Edwards public key
	Key string

	// VRFPublicKey is the miner's public key used for VRF.
	VRFPublicKey []byte
}

// String returns a string representation of the NodeID, for logging purposes.
// It implements the Stringer interface.
func (id NodeID) String() string {
	return id.Key + string(id.VRFPublicKey)
}

// ToBytes returns the byte representation of the Edwards public key.
func (id NodeID) ToBytes() []byte {
	return util.Hex2Bytes(id.String())
}

// ShortString returns a the first 5 characters of the ID, for logging purposes.
func (id NodeID) ShortString() string {
	name := id.Key
	return Shorten(name, 5)
}

// BytesToNodeID deserializes a byte slice into a NodeID
// TODO: length of the input will be made exact when the NodeID is compressed into
// one single key (https://github.com/spacemeshos/go-spacemesh/issues/2269)
func BytesToNodeID(b []byte) (*NodeID, error) {
	if len(b) < 32 {
		return nil, fmt.Errorf("invalid input length, input too short")
	}
	if len(b) > 64 {
		return nil, fmt.Errorf("invalid input length, input too long")
	}

	pubKey := b[0:32]
	vrfKey := b[32:]
	return &NodeID{
		Key:          util.Bytes2Hex(pubKey),
		VRFPublicKey: []byte(util.Bytes2Hex(vrfKey)),
	}, nil
}

// StringToNodeID deserializes a string into a NodeID
// TODO: length of the input will be made exact when the NodeID is compressed into
// one single key (https://github.com/spacemeshos/go-spacemesh/issues/2269)
func StringToNodeID(s string) (*NodeID, error) {
	strLen := len(s)
	if strLen < 64 {
		return nil, fmt.Errorf("invalid length, input too short")
	}
	if strLen > 128 {
		return nil, fmt.Errorf("invalid length, input too long")
	}
	// portion of the string corresponding to the Edwards public key
	pubKey := s[:64]
	vrfKey := s[64:]
	return &NodeID{
		Key:          pubKey,
		VRFPublicKey: []byte(vrfKey),
	}, nil
}

// Field returns a log field. Implements the LoggableField interface.
func (id NodeID) Field() log.Field { return log.String("node_id", id.Key) }

// Layer contains a list of proposals and their corresponding LayerID.
type Layer struct {
	proposals []*Proposal
	index     LayerID
}

// Field returns a log field. Implements the LoggableField interface.
func (l *Layer) Field() log.Field {
	return log.String("layer",
		fmt.Sprintf("layerhash %s layernum %d numblocks %d", l.Hash().String(), l.index, len(l.proposals)))
}

// Index returns the layer's ID.
func (l *Layer) Index() LayerID {
	return l.index
}

// Proposals returns the list of Proposal in this layer.
func (l *Layer) Proposals() []*Proposal {
	return l.proposals
}

// Blocks returns the list of Block in this layer.
func (l *Layer) Blocks() []*Block {
	blocks := make([]*Block, 0, len(l.proposals))
	for _, p := range l.proposals {
		blocks = append(blocks, (*Block)(p))
	}
	return blocks
}

// BlocksIDs returns the list of IDs for blocks in this layer.
func (l *Layer) BlocksIDs() []BlockID {
	ids := make([]BlockID, 0, len(l.proposals))
	for _, p := range l.proposals {
		ids = append(ids, BlockID(p.ID()))
	}
	return ids
}

// Ballots returns the list of ballots in this layer.
func (l *Layer) Ballots() []*Ballot {
	ballots := make([]*Ballot, 0, len(l.proposals))
	for _, p := range l.proposals {
		ballots = append(ballots, &p.Ballot)
	}
	return ballots
}

// ProposalsIDs returns the list of IDs for proposals in this layer.
func (l *Layer) ProposalsIDs() []ProposalID {
	ids := make([]ProposalID, len(l.proposals))
	for i := range l.proposals {
		ids[i] = l.proposals[i].ID()
	}
	return ids
}

// Hash returns the 32-byte sha256 sum of the block IDs of both contextually valid and invalid proposals in this layer,
// sorted in lexicographic order.
func (l Layer) Hash() Hash32 {
	if len(l.proposals) == 0 {
		return EmptyLayerHash
	}
	return CalcProposalsHash32(SortProposalIDs(ToProposalIDs(l.proposals)), nil)
}

// AddProposal adds a proposal to this layer. Panics if the proposal's index doesn't match the layer.
func (l *Layer) AddProposal(p *Proposal) {
	if p.LayerIndex != l.index {
		log.Panic("add proposal with wrong layer number act %v exp %v", p.LayerIndex, l.index)
	}
	l.proposals = append(l.proposals, p)
}

// AddBlock adds a block to this layer. Panics if the block's index doesn't match the layer.
func (l *Layer) AddBlock(b *Block) {
	l.AddProposal((*Proposal)(b))
}

// SetProposals sets the list of proposals for the layer without validation.
func (l *Layer) SetProposals(proposals []*Proposal) {
	l.proposals = proposals
}

// SetBlocks sets the list of blocks for the layer without validation.
func (l *Layer) SetBlocks(blocks []*Block) {
	l.proposals = ToProposals(blocks)
}

// NewExistingLayer returns a new layer with the given list of blocks without validation.
func NewExistingLayer(idx LayerID, blocks []*Block) *Layer {
	return &Layer{
		proposals: ToProposals(blocks),
		index:     idx,
	}
}

// NewLayer returns a layer with no proposals.
func NewLayer(layerIndex LayerID) *Layer {
	return &Layer{
		index:     layerIndex,
		proposals: make([]*Proposal, 0, 10),
	}
}