package fetch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/go-spacemesh/codec"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/datastore"
	"github.com/spacemeshos/go-spacemesh/fetch/mocks"
	"github.com/spacemeshos/go-spacemesh/log/logtest"
	"github.com/spacemeshos/go-spacemesh/p2p"
	"github.com/spacemeshos/go-spacemesh/sql"
)

type testFetch struct {
	*Fetch
	mh      *mocks.Mockhost
	mMalS   *mocks.Mockrequester
	mAtxS   *mocks.Mockrequester
	mLyrS   *mocks.Mockrequester
	mOpnS   *mocks.Mockrequester
	mHashS  *mocks.Mockrequester
	mMHashS *mocks.Mockrequester

	mMesh      *mocks.MockmeshProvider
	mMalH      *mocks.MockmalfeasanceHandler
	mAtxH      *mocks.MockatxHandler
	mBallotH   *mocks.MockballotHandler
	mBlocksH   *mocks.MockblockHandler
	mProposalH *mocks.MockproposalHandler
	method     int
	mTxH       *mocks.MocktxHandler
	mPoetH     *mocks.MockpoetHandler
}

func createFetch(tb testing.TB) *testFetch {
	ctrl := gomock.NewController(tb)
	tf := &testFetch{
		mh:         mocks.NewMockhost(ctrl),
		mMalS:      mocks.NewMockrequester(ctrl),
		mAtxS:      mocks.NewMockrequester(ctrl),
		mLyrS:      mocks.NewMockrequester(ctrl),
		mOpnS:      mocks.NewMockrequester(ctrl),
		mHashS:     mocks.NewMockrequester(ctrl),
		mMHashS:    mocks.NewMockrequester(ctrl),
		mMalH:      mocks.NewMockmalfeasanceHandler(ctrl),
		mAtxH:      mocks.NewMockatxHandler(ctrl),
		mBallotH:   mocks.NewMockballotHandler(ctrl),
		mBlocksH:   mocks.NewMockblockHandler(ctrl),
		mProposalH: mocks.NewMockproposalHandler(ctrl),
		mTxH:       mocks.NewMocktxHandler(ctrl),
		mPoetH:     mocks.NewMockpoetHandler(ctrl),
	}
	cfg := Config{
		time.Millisecond * time.Duration(2000), // make sure we never hit the batch timeout
		3,
		3,
		1000,
		time.Second * time.Duration(3),
		3,
	}
	lg := logtest.New(tb)
	tf.Fetch = NewFetch(datastore.NewCachedDB(sql.InMemory(), lg), tf.mMesh, nil, nil,
		WithContext(context.TODO()),
		WithConfig(cfg),
		WithLogger(lg),
		WithMalfeasanceHandler(tf.mMalH),
		WithATXHandler(tf.mAtxH),
		WithBallotHandler(tf.mBallotH),
		WithBlockHandler(tf.mBlocksH),
		WithProposalHandler(tf.mProposalH),
		WithTXHandler(tf.mTxH),
		WithPoetHandler(tf.mPoetH),
		withServers(map[string]requester{
			malProtocol:      tf.mMalS,
			atxProtocol:      tf.mAtxS,
			lyrDataProtocol:  tf.mLyrS,
			lyrOpnsProtocol:  tf.mOpnS,
			hashProtocol:     tf.mHashS,
			meshHashProtocol: tf.mMHashS,
		}),
		withHost(tf.mh))

	return tf
}

func goodReceiver(context.Context, p2p.Peer, []byte) error {
	return nil
}

func badReceiver(context.Context, p2p.Peer, []byte) error {
	return errors.New("bad receiver")
}

func TestFetch_GetHash(t *testing.T) {
	f := createFetch(t)
	f.mh.EXPECT().Close()
	f.Start()
	defer f.Stop()
	h1 := types.RandomHash()
	hint := datastore.POETDB
	hint2 := datastore.BallotDB

	// test hash aggregation
	p0, err := f.getHash(context.TODO(), h1, hint, goodReceiver)
	require.NoError(t, err)
	p1, err := f.getHash(context.TODO(), h1, hint, goodReceiver)
	require.NoError(t, err)
	require.Equal(t, p0.completed, p1.completed)

	h2 := types.RandomHash()
	p2, err := f.getHash(context.TODO(), h2, hint2, goodReceiver)
	require.NoError(t, err)
	require.NotEqual(t, p1.completed, p2.completed)
}

func TestFetch_RequestHashBatchFromPeers(t *testing.T) {
	tt := []struct {
		name       string
		nErr, vErr error
	}{
		{
			name: "request batch",
		},
		{
			name: "request batch network failure",
			nErr: errors.New("network failure"),
		},
		{
			name: "request batch validation failure",
			vErr: errors.New("validation failure"),
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := createFetch(t)
			f.cfg.MaxRetriesForRequest = 0
			f.cfg.MaxRetriesForPeer = 0
			peer := p2p.Peer("buddy")
			f.mh.EXPECT().GetPeers().Return([]p2p.Peer{peer})

			hsh0 := types.RandomHash()
			res0 := ResponseMessage{
				Hash: hsh0,
				Data: []byte("a"),
			}
			hsh1 := types.RandomHash()
			res1 := ResponseMessage{
				Hash: hsh1,
				Data: []byte("b"),
			}
			f.mHashS.EXPECT().Request(gomock.Any(), peer, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ p2p.Peer, req []byte, okFunc func([]byte), _ func(error)) error {
					if tc.nErr != nil {
						return tc.nErr
					}
					var rb RequestBatch
					err := codec.Decode(req, &rb)
					require.NoError(t, err)
					resBatch := ResponseBatch{
						ID:        rb.ID,
						Responses: []ResponseMessage{res0, res1},
					}
					bts, err := codec.Encode(&resBatch)
					require.NoError(t, err)
					okFunc(bts)
					return nil
				})

			var p0, p1 []*promise
			// query each hash twice
			receiver := goodReceiver
			if tc.vErr != nil {
				receiver = badReceiver
			}
			for i := 0; i < 2; i++ {
				p, err := f.getHash(context.TODO(), hsh0, datastore.ProposalDB, receiver)
				require.NoError(t, err)
				p0 = append(p0, p)
				p, err = f.getHash(context.TODO(), hsh1, datastore.BlockDB, receiver)
				require.NoError(t, err)
				p1 = append(p1, p)
			}
			require.Equal(t, p0[0], p0[1])
			require.Equal(t, p1[0], p1[1])

			f.requestHashBatchFromPeers()

			for _, p := range append(p0, p1...) {
				<-p.completed
				if tc.nErr != nil || tc.vErr != nil {
					require.Error(t, p.err)
				} else {
					require.NoError(t, p.err)
				}
			}
		})
	}
}

func TestFetch_GetHash_StartStopSanity(t *testing.T) {
	f := createFetch(t)
	f.mh.EXPECT().Close()
	f.Start()
	f.Stop()
}

func TestFetch_Loop_BatchRequestMax(t *testing.T) {
	f := createFetch(t)
	f.cfg.BatchTimeout = 1
	f.cfg.MaxRetriesForPeer = 2
	f.cfg.BatchSize = 2
	peer := p2p.Peer("buddy")
	f.mh.EXPECT().GetPeers().Return([]p2p.Peer{peer})

	h1 := types.RandomHash()
	h2 := types.RandomHash()
	h3 := types.RandomHash()
	f.mHashS.EXPECT().Request(gomock.Any(), peer, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ p2p.Peer, req []byte, okFunc func([]byte), _ func(error)) error {
			var rb RequestBatch
			err := codec.Decode(req, &rb)
			require.NoError(t, err)
			resps := make([]ResponseMessage, 0, len(rb.Requests))
			for _, r := range rb.Requests {
				resps = append(resps, ResponseMessage{
					Hash: r.Hash,
					Data: []byte("a"),
				})
			}
			resBatch := ResponseBatch{
				ID:        rb.ID,
				Responses: resps,
			}
			bts, err := codec.Encode(&resBatch)
			require.NoError(t, err)
			okFunc(bts)
			return nil
		}).Times(2) // 3 requests with batch size 2 -> 2 sends

	hint := datastore.POETDB

	f.mh.EXPECT().Close()
	defer f.Stop()
	f.Start()
	p1, err := f.getHash(context.TODO(), h1, hint, goodReceiver)
	require.NoError(t, err)
	p2, err := f.getHash(context.TODO(), h2, hint, goodReceiver)
	require.NoError(t, err)
	p3, err := f.getHash(context.TODO(), h3, hint, goodReceiver)
	require.NoError(t, err)
	for _, p := range []*promise{p1, p2, p3} {
		<-p.completed
		require.NoError(t, p.err)
	}
}

func TestFetch_GetRandomPeer(t *testing.T) {
	myPeers := make([]p2p.Peer, 1000)
	for i := 0; i < len(myPeers); i++ {
		myPeers[i] = p2p.Peer(types.RandomBytes(20))
	}
	allTheSame := true
	for i := 0; i < 20; i++ {
		peer1 := randomPeer(myPeers)
		peer2 := randomPeer(myPeers)
		if peer1 != peer2 {
			allTheSame = false
		}
	}
	assert.False(t, allTheSame)
}
