package beacon

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spacemeshos/fixed"
	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/go-spacemesh/activation"
	"github.com/spacemeshos/go-spacemesh/beacon/weakcoin"
	"github.com/spacemeshos/go-spacemesh/common/types"
	"github.com/spacemeshos/go-spacemesh/common/util"
	"github.com/spacemeshos/go-spacemesh/datastore"
	"github.com/spacemeshos/go-spacemesh/log/logtest"
	"github.com/spacemeshos/go-spacemesh/p2p"
	"github.com/spacemeshos/go-spacemesh/p2p/pubsub"
	pubsubmocks "github.com/spacemeshos/go-spacemesh/p2p/pubsub/mocks"
	"github.com/spacemeshos/go-spacemesh/signing"
	"github.com/spacemeshos/go-spacemesh/sql"
	"github.com/spacemeshos/go-spacemesh/sql/atxs"
	"github.com/spacemeshos/go-spacemesh/system/mocks"
)

const (
	numATXs = 10
)

func coinValueMock(tb testing.TB, value bool) coin {
	ctrl := gomock.NewController(tb)
	coinMock := NewMockcoin(ctrl)
	coinMock.EXPECT().StartEpoch(
		gomock.Any(),
		gomock.AssignableToTypeOf(types.EpochID(0)),
		gomock.AssignableToTypeOf(weakcoin.UnitAllowances{}),
	).AnyTimes()
	coinMock.EXPECT().FinishEpoch(gomock.Any(), gomock.AssignableToTypeOf(types.EpochID(0))).AnyTimes()
	nonce := types.VRFPostIndex(0)
	coinMock.EXPECT().StartRound(gomock.Any(),
		gomock.AssignableToTypeOf(types.RoundID(0)),
		gomock.AssignableToTypeOf(&nonce),
	).AnyTimes().Return(nil)
	coinMock.EXPECT().FinishRound(gomock.Any()).AnyTimes()
	coinMock.EXPECT().Get(
		gomock.Any(),
		gomock.AssignableToTypeOf(types.EpochID(0)),
		gomock.AssignableToTypeOf(types.RoundID(0)),
	).AnyTimes().Return(value)
	return coinMock
}

func newPublisher(tb testing.TB) pubsub.Publisher {
	tb.Helper()
	ctrl := gomock.NewController(tb)

	publisher := pubsubmocks.NewMockPublisher(ctrl)
	publisher.EXPECT().
		Publish(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()
	return publisher
}

type testProtocolDriver struct {
	*ProtocolDriver
	ctrl          *gomock.Controller
	cdb           *datastore.CachedDB
	mClock        *MocklayerClock
	mSync         *mocks.MockSyncStateProvider
	mSigner       *MockvrfSigner
	mVerifier     *MockvrfVerifier
	mNonceFetcher *MocknonceFetcher
}

func setUpProtocolDriver(tb testing.TB) *testProtocolDriver {
	return newTestDriver(tb, UnitTestConfig(), newPublisher(tb))
}

func newTestDriver(tb testing.TB, cfg Config, p pubsub.Publisher) *testProtocolDriver {
	ctrl := gomock.NewController(tb)
	tpd := &testProtocolDriver{
		ctrl:          ctrl,
		mClock:        NewMocklayerClock(ctrl),
		mSync:         mocks.NewMockSyncStateProvider(ctrl),
		mSigner:       NewMockvrfSigner(ctrl),
		mVerifier:     NewMockvrfVerifier(ctrl),
		mNonceFetcher: NewMocknonceFetcher(ctrl),
	}
	edSgn, err := signing.NewEdSigner()
	require.NoError(tb, err)
	extractor, err := signing.NewPubKeyExtractor()
	require.NoError(tb, err)
	minerID := edSgn.NodeID()
	lg := logtest.New(tb).WithName(minerID.ShortString())

	tpd.mSigner.EXPECT().Sign(gomock.Any()).AnyTimes().Return([]byte{})
	tpd.mVerifier.EXPECT().Verify(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(true)
	tpd.mNonceFetcher.EXPECT().VRFNonce(gomock.Any(), gomock.Any()).AnyTimes().Return(types.VRFPostIndex(1), nil)

	tpd.cdb = datastore.NewCachedDB(sql.InMemory(), lg)
	tpd.ProtocolDriver = New(minerID, p, edSgn, extractor, tpd.mSigner, tpd.mVerifier, tpd.cdb, tpd.mClock,
		WithConfig(cfg),
		WithLogger(lg),
		withWeakCoin(coinValueMock(tb, true)),
		withNonceFetcher(tpd.mNonceFetcher),
	)
	tpd.ProtocolDriver.SetSyncState(tpd.mSync)
	tpd.ProtocolDriver.setMetricsRegistry(prometheus.NewPedanticRegistry())
	return tpd
}

func createATX(tb testing.TB, db *datastore.CachedDB, lid types.LayerID, sig *signing.EdSigner, numUnits uint32) {
	nodeID := sig.NodeID()
	nonce := types.VRFPostIndex(1)
	atx := types.NewActivationTx(
		types.NIPostChallenge{PubLayerID: lid},
		&nodeID,
		types.Address{},
		nil,
		numUnits,
		nil,
		&nonce,
	)

	atx.SetEffectiveNumUnits(numUnits)
	atx.SetReceived(time.Now().Add(-1 * time.Second))
	require.NoError(tb, activation.SignAndFinalizeAtx(sig, atx))
	vAtx, err := atx.Verify(0, 1)
	require.NoError(tb, err)
	require.NoError(tb, atxs.Add(db, vAtx))
}

func createRandomATXs(tb testing.TB, db *datastore.CachedDB, lid types.LayerID, num int) {
	for i := 0; i < num; i++ {
		sig, err := signing.NewEdSigner()
		require.NoError(tb, err)
		createATX(tb, db, lid, sig, 1)
	}
}

func TestMain(m *testing.M) {
	types.SetLayersPerEpoch(3)

	res := m.Run()
	os.Exit(res)
}

func TestBeacon_MultipleNodes(t *testing.T) {
	numNodes := 5
	testNodes := make([]*testProtocolDriver, 0, numNodes)
	publisher := pubsubmocks.NewMockPublisher(gomock.NewController(t))
	publisher.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, protocol string, data []byte) error {
			for _, node := range testNodes {
				switch protocol {
				case pubsub.BeaconProposalProtocol:
					require.NoError(t, node.handleProposal(ctx, p2p.Peer(node.nodeID.ShortString()), data, time.Now()))
				case pubsub.BeaconFirstVotesProtocol:
					require.NoError(t, node.handleFirstVotes(ctx, p2p.Peer(node.nodeID.ShortString()), data))
				case pubsub.BeaconFollowingVotesProtocol:
					require.NoError(t, node.handleFollowingVotes(ctx, p2p.Peer(node.nodeID.ShortString()), data, time.Now()))
				case pubsub.BeaconWeakCoinProtocol:
				}
			}
			return nil
		}).AnyTimes()

	atxPublishLid := types.NewLayerID(types.GetLayersPerEpoch()*2 - 1)
	current := atxPublishLid.Add(1)
	dbs := make([]*datastore.CachedDB, 0, numNodes)
	cfg := NodeSimUnitTestConfig()
	now := time.Now()
	for i := 0; i < numNodes; i++ {
		node := newTestDriver(t, cfg, publisher)
		node.mSync.EXPECT().IsSynced(gomock.Any()).Return(true).AnyTimes()
		node.mClock.EXPECT().CurrentLayer().Return(current).AnyTimes()
		node.mClock.EXPECT().LayerToTime(current).Return(now).AnyTimes()
		testNodes = append(testNodes, node)
		dbs = append(dbs, node.cdb)

		require.ErrorIs(t, node.onNewEpoch(context.Background(), types.EpochID(0)), errGenesis)
		require.ErrorIs(t, node.onNewEpoch(context.Background(), types.EpochID(1)), errGenesis)
		got, err := node.GetBeacon(types.EpochID(1))
		require.NoError(t, err)
		require.EqualValues(t, got, types.HexToBeacon(types.BootstrapBeacon))
		got, err = node.GetBeacon(types.EpochID(2))
		require.NoError(t, err)
		require.EqualValues(t, got, types.HexToBeacon(types.BootstrapBeacon))
	}
	for i, node := range testNodes {
		if i == 0 {
			// make the first node non-smeshing node
			continue
		}
		for _, db := range dbs {
			createATX(t, db, atxPublishLid, node.edSigner, 1)
		}
	}
	var wg sync.WaitGroup
	for _, node := range testNodes {
		wg.Add(1)
		go func(testNode *testProtocolDriver) {
			require.NoError(t, testNode.onNewEpoch(context.Background(), types.EpochID(2)))
			wg.Done()
		}(node)
	}
	wg.Wait()
	beacons := make(map[types.Beacon]struct{})
	for _, node := range testNodes {
		got, err := node.GetBeacon(types.EpochID(3))
		require.NoError(t, err)
		require.NotEqual(t, types.EmptyBeacon, got)
		beacons[got] = struct{}{}
	}
	require.Len(t, beacons, 1)
}

func TestBeacon_NoProposals(t *testing.T) {
	numNodes := 5
	testNodes := make([]*testProtocolDriver, 0, numNodes)
	publisher := pubsubmocks.NewMockPublisher(gomock.NewController(t))
	publisher.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	atxPublishLid := types.NewLayerID(types.GetLayersPerEpoch()*2 - 1)
	current := atxPublishLid.Add(1)
	dbs := make([]*datastore.CachedDB, 0, numNodes)
	cfg := NodeSimUnitTestConfig()
	now := time.Now()
	for i := 0; i < numNodes; i++ {
		node := newTestDriver(t, cfg, publisher)
		node.mSync.EXPECT().IsSynced(gomock.Any()).Return(true).AnyTimes()
		node.mClock.EXPECT().CurrentLayer().Return(current).AnyTimes()
		node.mClock.EXPECT().LayerToTime(current).Return(now).AnyTimes()
		testNodes = append(testNodes, node)
		dbs = append(dbs, node.cdb)

		require.ErrorIs(t, node.onNewEpoch(context.Background(), types.EpochID(0)), errGenesis)
		require.ErrorIs(t, node.onNewEpoch(context.Background(), types.EpochID(1)), errGenesis)
		got, err := node.GetBeacon(types.EpochID(1))
		require.NoError(t, err)
		require.EqualValues(t, got, types.HexToBeacon(types.BootstrapBeacon))
		got, err = node.GetBeacon(types.EpochID(2))
		require.NoError(t, err)
		require.EqualValues(t, got, types.HexToBeacon(types.BootstrapBeacon))
	}
	for _, node := range testNodes {
		for _, db := range dbs {
			createATX(t, db, atxPublishLid, node.edSigner, 1)
		}
	}
	var wg sync.WaitGroup
	for _, node := range testNodes {
		wg.Add(1)
		go func(testNode *testProtocolDriver) {
			require.NoError(t, testNode.onNewEpoch(context.Background(), types.EpochID(2)))
			wg.Done()
		}(node)
	}
	wg.Wait()
	for _, node := range testNodes {
		got, err := node.GetBeacon(types.EpochID(3))
		require.Error(t, err)
		require.Equal(t, types.EmptyBeacon, got)
	}
}

func TestBeaconNotSynced(t *testing.T) {
	tpd := setUpProtocolDriver(t)
	tpd.mSync.EXPECT().IsSynced(gomock.Any()).Return(false).AnyTimes()
	require.ErrorIs(t, tpd.onNewEpoch(context.Background(), types.EpochID(0)), errGenesis)
	require.ErrorIs(t, tpd.onNewEpoch(context.Background(), types.EpochID(1)), errGenesis)
	require.ErrorIs(t, tpd.onNewEpoch(context.Background(), types.EpochID(2)), errNodeNotSynced)

	got, err := tpd.GetBeacon(types.EpochID(2))
	require.NoError(t, err)
	require.EqualValues(t, got, types.HexToBeacon(types.BootstrapBeacon))
	got, err = tpd.GetBeacon(types.EpochID(3))
	require.Equal(t, errBeaconNotCalculated, err)
	require.Equal(t, types.EmptyBeacon, got)
}

func TestBeaconNotSynced_ReleaseMemory(t *testing.T) {
	tpd := setUpProtocolDriver(t)
	tpd.config.BeaconSyncWeightUnits = 1
	tpd.mSync.EXPECT().IsSynced(gomock.Any()).Return(false).AnyTimes()
	start := types.EpochID(2)
	end := start + numEpochsToKeep + 10
	for eid := start; eid <= end; eid++ {
		b := types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
			Layer: start.FirstLayer(),
		})
		b.EligibilityProofs = []types.VotingEligibility{{J: 1}}
		tpd.ReportBeaconFromBallot(eid, &b, types.RandomBeacon(), fixed.New64(1))
		require.ErrorIs(t, tpd.onNewEpoch(context.Background(), eid), errNodeNotSynced)
	}
	require.Len(t, tpd.beacons, numEpochsToKeep)
	require.Len(t, tpd.ballotsBeacons, numEpochsToKeep)
}

func TestBeaconNoATXInPreviousEpoch(t *testing.T) {
	tpd := setUpProtocolDriver(t)
	tpd.mSync.EXPECT().IsSynced(gomock.Any()).Return(true).AnyTimes()
	require.ErrorIs(t, tpd.onNewEpoch(context.Background(), types.EpochID(0)), errGenesis)
	require.ErrorIs(t, tpd.onNewEpoch(context.Background(), types.EpochID(1)), errGenesis)
	lid := types.NewLayerID(types.GetLayersPerEpoch()*2 - 1)
	createRandomATXs(t, tpd.cdb, lid, numATXs)
	require.NoError(t, tpd.onNewEpoch(context.Background(), types.EpochID(2)))

	got, err := tpd.GetBeacon(types.EpochID(2))
	require.NoError(t, err)
	require.EqualValues(t, got, types.HexToBeacon(types.BootstrapBeacon))
	got, err = tpd.GetBeacon(types.EpochID(3))
	require.Equal(t, errBeaconNotCalculated, err)
	require.Equal(t, types.EmptyBeacon, got)
}

func TestBeaconWithMetrics(t *testing.T) {
	tpd := setUpProtocolDriver(t)
	tpd.mSync.EXPECT().IsSynced(gomock.Any()).Return(true).AnyTimes()

	gLayer := types.GetEffectiveGenesis()
	tpd.mClock.EXPECT().CurrentLayer().Return(gLayer).Times(2)
	tpd.mClock.EXPECT().AwaitLayer(gLayer.Add(1)).Return(nil).Times(1)
	tpd.mClock.EXPECT().LayerToTime((gLayer.GetEpoch() + 1).FirstLayer()).Return(time.Now()).Times(1)
	tpd.Start(context.Background())

	epoch3Beacon := types.HexToBeacon("0xaf1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262")
	epoch := types.EpochID(3)
	for i := types.EpochID(2); i < epoch; i++ {
		lid := i.FirstLayer().Sub(1)
		createATX(t, tpd.cdb, lid, tpd.edSigner, 199)
		createRandomATXs(t, tpd.cdb, lid, numATXs-1)
	}
	finalLayer := types.NewLayerID(types.GetLayersPerEpoch() * uint32(epoch))
	beacon1 := types.RandomBeacon()
	beacon2 := types.RandomBeacon()
	for layer := gLayer.Add(1); layer.Before(finalLayer); layer = layer.Add(1) {
		tpd.mClock.EXPECT().CurrentLayer().Return(layer).AnyTimes()
		if layer.FirstInEpoch() {
			require.NoError(t, tpd.onNewEpoch(context.Background(), layer.GetEpoch()))
		}
		thisEpoch := layer.GetEpoch()
		b := types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
			Layer: thisEpoch.FirstLayer(),
		})
		b.EligibilityProofs = []types.VotingEligibility{{J: 1}}
		tpd.recordBeacon(thisEpoch, &b, beacon1, fixed.New64(1))
		b = types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
			Layer: thisEpoch.FirstLayer(),
		})
		b.EligibilityProofs = []types.VotingEligibility{{J: 1}}
		tpd.recordBeacon(thisEpoch, &b, beacon2, fixed.New64(1))

		numCalculated := 0
		numObserved := 0
		numObservedWeight := 0
		allMetrics, err := prometheus.DefaultGatherer.Gather()
		require.NoError(t, err)
		for _, m := range allMetrics {
			switch *m.Name {
			case "spacemesh_beacons_beacon_calculated_weight":
				require.Equal(t, 1, len(m.Metric))
				numCalculated++
				beaconStr := epoch3Beacon.ShortString()
				expected := fmt.Sprintf("label:<name:\"beacon\" value:\"%s\" > label:<name:\"epoch\" value:\"%d\" > counter:<value:%d > ", beaconStr, thisEpoch+1, 0)
				require.Equal(t, expected, m.Metric[0].String())
			case "spacemesh_beacons_beacon_observed_total":
				require.Equal(t, 2, len(m.Metric))
				numObserved = numObserved + 2
				count := layer.OrdinalInEpoch() + 1
				expected := []string{
					fmt.Sprintf("label:<name:\"beacon\" value:\"%s\" > label:<name:\"epoch\" value:\"%d\" > counter:<value:%d > ", beacon1.ShortString(), thisEpoch, count),
					fmt.Sprintf("label:<name:\"beacon\" value:\"%s\" > label:<name:\"epoch\" value:\"%d\" > counter:<value:%d > ", beacon2.ShortString(), thisEpoch, count),
				}
				for _, subM := range m.Metric {
					require.Contains(t, expected, subM.String())
				}
			case "spacemesh_beacons_beacon_observed_weight":
				require.Equal(t, 2, len(m.Metric))
				numObservedWeight = numObservedWeight + 2
				weight := layer.OrdinalInEpoch() + 1
				expected := []string{
					fmt.Sprintf("label:<name:\"beacon\" value:\"%s\" > label:<name:\"epoch\" value:\"%d\" > counter:<value:%d > ", beacon1.ShortString(), thisEpoch, weight),
					fmt.Sprintf("label:<name:\"beacon\" value:\"%s\" > label:<name:\"epoch\" value:\"%d\" > counter:<value:%d > ", beacon2.ShortString(), thisEpoch, weight),
				}
				for _, subM := range m.Metric {
					require.Contains(t, expected, subM.String())
				}
			}
		}
		require.Equal(t, 0, numCalculated, layer)
		require.Equal(t, 2, numObserved, layer)
		require.Equal(t, 2, numObservedWeight, layer)
	}

	tpd.Close()
}

func TestBeacon_BeaconsWithDatabase(t *testing.T) {
	t.Parallel()

	pd := &ProtocolDriver{
		logger:  logtest.New(t).WithName("Beacon"),
		beacons: make(map[types.EpochID]types.Beacon),
		cdb:     datastore.NewCachedDB(sql.InMemory(), logtest.New(t)),
	}
	epoch3 := types.EpochID(3)
	beacon2 := types.RandomBeacon()
	epoch5 := types.EpochID(5)
	beacon4 := types.RandomBeacon()
	err := pd.setBeacon(epoch3, beacon2)
	require.NoError(t, err)
	err = pd.setBeacon(epoch5, beacon4)
	require.NoError(t, err)

	got, err := pd.GetBeacon(epoch3)
	require.NoError(t, err)
	require.Equal(t, beacon2, got)

	got, err = pd.GetBeacon(epoch5)
	require.NoError(t, err)
	require.Equal(t, beacon4, got)

	got, err = pd.GetBeacon(epoch5 - 1)
	require.Equal(t, errBeaconNotCalculated, err)
	require.Equal(t, types.EmptyBeacon, got)

	// clear out the in-memory map
	// the database should still give us values
	pd.mu.Lock()
	pd.beacons = make(map[types.EpochID]types.Beacon)
	pd.mu.Unlock()

	got, err = pd.GetBeacon(epoch3)
	require.NoError(t, err)
	require.Equal(t, beacon2, got)

	got, err = pd.GetBeacon(epoch5)
	require.NoError(t, err)
	require.Equal(t, beacon4, got)

	got, err = pd.GetBeacon(epoch5 - 1)
	require.Equal(t, errBeaconNotCalculated, err)
	require.Equal(t, types.EmptyBeacon, got)
}

func TestBeacon_BeaconsWithDatabaseFailure(t *testing.T) {
	t.Parallel()

	pd := &ProtocolDriver{
		logger:  logtest.New(t).WithName("Beacon"),
		beacons: make(map[types.EpochID]types.Beacon),
		cdb:     datastore.NewCachedDB(sql.InMemory(), logtest.New(t)),
	}
	epoch := types.EpochID(3)

	got, errGet := pd.getPersistedBeacon(epoch)
	require.Equal(t, types.EmptyBeacon, got)
	require.ErrorIs(t, errGet, sql.ErrNotFound)
}

func TestBeacon_BeaconsCleanupOldEpoch(t *testing.T) {
	t.Parallel()

	pd := &ProtocolDriver{
		logger:         logtest.New(t).WithName("Beacon"),
		cdb:            datastore.NewCachedDB(sql.InMemory(), logtest.New(t)),
		beacons:        make(map[types.EpochID]types.Beacon),
		ballotsBeacons: make(map[types.EpochID]map[types.Beacon]*beaconWeight),
	}

	epoch := types.EpochID(5)
	for i := 0; i < numEpochsToKeep; i++ {
		e := epoch + types.EpochID(i)
		err := pd.setBeacon(e, types.RandomBeacon())
		require.NoError(t, err)
		b := types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
			Layer: e.FirstLayer(),
		})
		b.EligibilityProofs = []types.VotingEligibility{{J: 1}}
		pd.ReportBeaconFromBallot(e, &b, types.RandomBeacon(), fixed.New64(1))
		pd.cleanupEpoch(e)
		require.Equal(t, i+1, len(pd.beacons))
		require.Equal(t, i+1, len(pd.ballotsBeacons))
	}
	require.Equal(t, numEpochsToKeep, len(pd.beacons))
	require.Equal(t, numEpochsToKeep, len(pd.ballotsBeacons))

	epoch = epoch + numEpochsToKeep
	err := pd.setBeacon(epoch, types.RandomBeacon())
	require.NoError(t, err)
	b := types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
		Layer: epoch.FirstLayer(),
	})
	b.EligibilityProofs = []types.VotingEligibility{{J: 1}}
	pd.recordBeacon(epoch, &b, types.RandomBeacon(), fixed.New64(1))
	require.Equal(t, numEpochsToKeep+1, len(pd.beacons))
	require.Equal(t, numEpochsToKeep+1, len(pd.ballotsBeacons))
	pd.cleanupEpoch(epoch)
	require.Equal(t, numEpochsToKeep, len(pd.beacons))
	require.Equal(t, numEpochsToKeep, len(pd.ballotsBeacons))
}

func TestBeacon_ReportBeaconFromBallot(t *testing.T) {
	t.Parallel()

	beacon1 := types.RandomBeacon()
	beacon2 := types.RandomBeacon()
	beacon3 := types.RandomBeacon()
	tt := []struct {
		name          string
		majority      bool
		beacon        types.Beacon
		beaconBallots map[types.Beacon][]fixed.Fixed
	}{
		{
			name: "majority",
			beaconBallots: map[types.Beacon][]fixed.Fixed{
				beacon1: {fixed.New64(1), fixed.New64(1)},
				beacon2: {fixed.New64(1)},
				beacon3: {fixed.Div64(1, 10)},
			},
			majority: true,
			beacon:   beacon1,
		},
		{
			name: "plurality",
			beaconBallots: map[types.Beacon][]fixed.Fixed{
				beacon1: {fixed.New64(1)},
				beacon2: {fixed.Div64(11, 10)},
				beacon3: {fixed.Div64(3, 10), fixed.Div64(7, 10)},
			},
			beacon: beacon2,
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// making sure the math in test arguments are correct
			total := fixed.New64(0)
			beaconWeights := make(map[types.Beacon]fixed.Fixed)
			for beacon, weights := range tc.beaconBallots {
				bweight := fixed.New64(0)
				for _, w := range weights {
					total = total.Add(w)
					bweight = bweight.Add(w)
				}
				beaconWeights[beacon] = bweight
			}
			maxWeight := fixed.New64(0)
			majorityWeight := total.Div(fixed.New64(2))
			var found bool
			for _, weight := range beaconWeights {
				if weight.GreaterThan(majorityWeight) {
					found = true
				}
				if weight.GreaterThan(maxWeight) {
					maxWeight = weight
				}
			}
			if tc.majority {
				require.True(t, found)
			} else {
				require.Greater(t, maxWeight.Float(), 0.0)
			}

			pd := &ProtocolDriver{
				logger:         logtest.New(t).WithName("Beacon"),
				config:         UnitTestConfig(),
				cdb:            datastore.NewCachedDB(sql.InMemory(), logtest.New(t)),
				beacons:        make(map[types.EpochID]types.Beacon),
				ballotsBeacons: make(map[types.EpochID]map[types.Beacon]*beaconWeight),
			}
			pd.config.BeaconSyncWeightUnits = 4

			epoch := types.EpochID(3)
			for beacon, weights := range tc.beaconBallots {
				for _, w := range weights {
					b := types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
						Layer: epoch.FirstLayer(),
					})
					b.EligibilityProofs = []types.VotingEligibility{{J: 1}}
					pd.ReportBeaconFromBallot(epoch, &b, beacon, w)
				}
			}
			got, err := pd.GetBeacon(epoch)
			require.NoError(t, err)
			require.Equal(t, tc.beacon, got)
		})
	}
}

func TestBeacon_ReportBeaconFromBallot_SameBallot(t *testing.T) {
	t.Parallel()

	pd := &ProtocolDriver{
		logger:         logtest.New(t).WithName("Beacon"),
		config:         UnitTestConfig(),
		cdb:            datastore.NewCachedDB(sql.InMemory(), logtest.New(t)),
		beacons:        make(map[types.EpochID]types.Beacon),
		ballotsBeacons: make(map[types.EpochID]map[types.Beacon]*beaconWeight),
	}
	pd.config.BeaconSyncWeightUnits = 2

	epoch := types.EpochID(3)
	beacon1 := types.RandomBeacon()
	beacon2 := types.RandomBeacon()

	b1 := types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
		Layer: epoch.FirstLayer(),
	})
	b1.EligibilityProofs = []types.VotingEligibility{{J: 1}}
	pd.ReportBeaconFromBallot(epoch, &b1, beacon1, fixed.New64(1))
	pd.ReportBeaconFromBallot(epoch, &b1, beacon1, fixed.New64(1))
	// same ballotID does not count twice
	got, err := pd.GetBeacon(epoch)
	require.Equal(t, errBeaconNotCalculated, err)
	require.Equal(t, types.EmptyBeacon, got)

	b2 := types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
		Layer: epoch.FirstLayer(),
	})
	b2.EligibilityProofs = []types.VotingEligibility{{J: 1}}
	pd.ReportBeaconFromBallot(epoch, &b2, beacon2, fixed.New64(2))
	got, err = pd.GetBeacon(epoch)
	require.NoError(t, err)
	require.Equal(t, beacon2, got)
}

func TestBeacon_ensureEpochHasBeacon_BeaconAlreadyCalculated(t *testing.T) {
	t.Parallel()

	epoch := types.EpochID(3)
	beacon := types.RandomBeacon()
	beaconFromBallots := types.RandomBeacon()
	pd := &ProtocolDriver{
		logger: logtest.New(t).WithName("Beacon"),
		config: UnitTestConfig(),
		beacons: map[types.EpochID]types.Beacon{
			epoch: beacon,
		},
		ballotsBeacons: make(map[types.EpochID]map[types.Beacon]*beaconWeight),
	}
	pd.config.BeaconSyncWeightUnits = 2

	got, err := pd.GetBeacon(epoch)
	require.NoError(t, err)
	require.Equal(t, beacon, got)

	b1 := types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
		Layer: epoch.FirstLayer(),
	})
	b1.EligibilityProofs = []types.VotingEligibility{{J: 1}}
	pd.ReportBeaconFromBallot(epoch, &b1, beaconFromBallots, fixed.New64(1))
	b2 := types.NewExistingBallot(types.RandomBallotID(), nil, types.EmptyNodeID, types.BallotMetadata{
		Layer: epoch.FirstLayer(),
	})
	b2.EligibilityProofs = []types.VotingEligibility{{J: 1}}
	pd.ReportBeaconFromBallot(epoch, &b2, beaconFromBallots, fixed.New64(1))

	// should not change the beacon value
	got, err = pd.GetBeacon(epoch)
	require.NoError(t, err)
	require.Equal(t, beacon, got)
}

func TestBeacon_findMajorityBeacon(t *testing.T) {
	t.Parallel()

	beacon1 := types.RandomBeacon()
	beacon2 := types.RandomBeacon()
	beacon3 := types.RandomBeacon()

	beaconFromBallots := map[types.Beacon]*beaconWeight{
		beacon1: {
			ballots:        map[types.BallotID]struct{}{types.RandomBallotID(): {}, types.RandomBallotID(): {}},
			totalWeight:    fixed.New64(1),
			numEligibility: 2,
		},
		beacon2: {
			ballots:        map[types.BallotID]struct{}{types.RandomBallotID(): {}},
			totalWeight:    fixed.New64(3),
			numEligibility: 1,
		},
		beacon3: {
			ballots:        map[types.BallotID]struct{}{types.RandomBallotID(): {}},
			totalWeight:    fixed.New64(1),
			numEligibility: 1,
		},
	}
	epoch := types.EpochID(3)
	pd := &ProtocolDriver{
		logger:         logtest.New(t).WithName("Beacon"),
		config:         UnitTestConfig(),
		beacons:        make(map[types.EpochID]types.Beacon),
		ballotsBeacons: map[types.EpochID]map[types.Beacon]*beaconWeight{epoch: beaconFromBallots},
	}
	pd.config.BeaconSyncWeightUnits = 4
	got := pd.findMajorityBeacon(epoch)
	require.Equal(t, beacon2, got)
}

func TestBeacon_findMajorityBeacon_plurality(t *testing.T) {
	t.Parallel()

	beacon1 := types.RandomBeacon()
	beacon2 := types.RandomBeacon()
	beacon3 := types.RandomBeacon()

	beaconFromBallots := map[types.Beacon]*beaconWeight{
		beacon1: {
			ballots:        map[types.BallotID]struct{}{types.RandomBallotID(): {}, types.RandomBallotID(): {}},
			totalWeight:    fixed.New64(1),
			numEligibility: 2,
		},
		beacon2: {
			ballots:        map[types.BallotID]struct{}{types.RandomBallotID(): {}},
			totalWeight:    fixed.DivUint64(11, 10),
			numEligibility: 1,
		},
		beacon3: {
			ballots:        map[types.BallotID]struct{}{types.RandomBallotID(): {}},
			totalWeight:    fixed.New64(1),
			numEligibility: 1,
		},
	}
	epoch := types.EpochID(3)
	pd := &ProtocolDriver{
		logger:         logtest.New(t).WithName("Beacon"),
		config:         UnitTestConfig(),
		beacons:        make(map[types.EpochID]types.Beacon),
		ballotsBeacons: map[types.EpochID]map[types.Beacon]*beaconWeight{epoch: beaconFromBallots},
	}
	pd.config.BeaconSyncWeightUnits = 4
	got := pd.findMajorityBeacon(epoch)
	require.Equal(t, beacon2, got)
}

func TestBeacon_findMajorityBeacon_NotEnoughBallots(t *testing.T) {
	t.Parallel()

	beacon1 := types.RandomBeacon()
	beacon2 := types.RandomBeacon()
	beacon3 := types.RandomBeacon()

	beaconFromBallots := map[types.Beacon]*beaconWeight{
		beacon1: {
			ballots:        map[types.BallotID]struct{}{types.RandomBallotID(): {}, types.RandomBallotID(): {}},
			totalWeight:    fixed.New64(1),
			numEligibility: 2,
		},
		beacon2: {
			ballots:        map[types.BallotID]struct{}{types.RandomBallotID(): {}},
			totalWeight:    fixed.New64(3),
			numEligibility: 1,
		},
		beacon3: {
			ballots:        map[types.BallotID]struct{}{types.RandomBallotID(): {}},
			totalWeight:    fixed.New64(1),
			numEligibility: 1,
		},
	}
	epoch := types.EpochID(3)
	pd := &ProtocolDriver{
		logger:         logtest.New(t).WithName("Beacon"),
		config:         UnitTestConfig(),
		beacons:        make(map[types.EpochID]types.Beacon),
		ballotsBeacons: map[types.EpochID]map[types.Beacon]*beaconWeight{epoch: beaconFromBallots},
	}
	pd.config.BeaconSyncWeightUnits = 5
	got := pd.findMajorityBeacon(epoch)
	require.Equal(t, types.EmptyBeacon, got)
}

func TestBeacon_findMajorityBeacon_NoBeacon(t *testing.T) {
	t.Parallel()

	pd := &ProtocolDriver{
		logger:         logtest.New(t).WithName("Beacon"),
		config:         UnitTestConfig(),
		beacons:        make(map[types.EpochID]types.Beacon),
		ballotsBeacons: make(map[types.EpochID]map[types.Beacon]*beaconWeight),
	}
	epoch := types.EpochID(3)
	got := pd.findMajorityBeacon(epoch)
	require.Equal(t, types.EmptyBeacon, got)
}

func TestBeacon_setBeacon(t *testing.T) {
	t.Parallel()

	tpd := setUpProtocolDriver(t)
	epoch := types.EpochID(5)
	beacon := types.RandomBeacon()
	require.NoError(t, tpd.setBeacon(epoch, beacon))

	// saving it again won't cause error
	require.NoError(t, tpd.setBeacon(epoch, beacon))
	// but saving a different one will
	require.ErrorIs(t, tpd.setBeacon(epoch, types.RandomBeacon()), errDifferentBeacon)
}

func TestBeacon_atxThresholdFraction(t *testing.T) {
	t.Parallel()

	kappa := 40
	q := big.NewRat(1, 3)
	tt := []struct {
		name      string
		w         int
		threshold string
	}{
		{
			name:      "0 atxs",
			w:         0,
			threshold: "0",
		},
		{
			name:      "30 atxs",
			w:         30,
			threshold: "0.75",
		},
		{
			name:      "10000 atxs",
			w:         10000,
			threshold: "0.004150246906",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			threshold := atxThresholdFraction(kappa, q, tc.w)
			expected, ok := new(big.Float).SetString(tc.threshold)
			require.True(t, ok)
			require.Equal(t, expected.String(), threshold.String())
		})
	}
}

func TestBeacon_atxThreshold(t *testing.T) {
	t.Parallel()

	kappa := 40
	q := big.NewRat(1, 3)
	tt := []struct {
		name      string
		w         int
		threshold string
	}{
		{
			name:      "Case 1",
			w:         60,
			threshold: "2281220308811097609320585802850145662446614253624279965289596258949637583604338693252956405658685699889321154786797203655344352360687718999126330659861107094125997337180132475041437096123301888",
		},
		{
			name:      "Case 2",
			w:         10_000,
			threshold: "18935255055005106377502632398712282551719911452308460382048488311892953261334543784347262759720917437038480763542475179136852475093285227949507665240470812709909012324673914264266057279602688",
		},
		{
			name:      "Case 3",
			w:         100_000,
			threshold: "1897071198136899971649143041158774510550610452467254281004680932854399011006090657229005189686141691549362111655061764931498490692502427712970984851624895957066181188995976870137803158061056",
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			threshold := atxThreshold(kappa, q, tc.w)
			expected, ok := new(big.Int).SetString(tc.threshold, 10)
			require.True(t, ok)
			require.Equal(t, expected, threshold)
		})
	}
}

func TestBeacon_proposalPassesEligibilityThreshold(t *testing.T) {
	cfg := Config{Kappa: 40, Q: big.NewRat(1, 3)}
	tt := []struct {
		name string
		w    int
	}{
		{
			name: "100K atxs",
			w:    100_000,
		},
		{
			name: "10K atxs",
			w:    10_000,
		},
		{
			name: "30 atxs",
			w:    30,
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger := logtest.New(t).WithName("proposal checker")
			checker := createProposalChecker(logger, cfg, tc.w)
			numEligible := 0
			for i := 0; i < tc.w; i++ {
				signer, err := signing.NewEdSigner()
				require.NoError(t, err)
				vrfSigner, err := signer.VRFSigner()
				require.NoError(t, err)
				proposal := buildSignedProposal(context.Background(), logtest.New(t), vrfSigner, 3, types.VRFPostIndex(1))
				if checker.IsProposalEligible(proposal) {
					numEligible++
				}
			}
			require.NotZero(t, numEligible)
		})
	}
}

func TestBeacon_buildProposal(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name   string
		epoch  types.EpochID
		result string
	}{
		{
			name:   "Case 1",
			epoch:  13110,
			result: string(util.Hex2Bytes("0404d9cc")),
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := buildProposal(logtest.New(t), tc.epoch, types.VRFPostIndex(1))
			require.Equal(t, tc.result, string(result))
		})
	}
}

func TestBeacon_getSignedProposal(t *testing.T) {
	t.Parallel()

	edSgn, err := signing.NewEdSigner()
	require.NoError(t, err)
	vrfSigner, err := edSgn.VRFSigner()
	require.NoError(t, err)

	tt := []struct {
		name   string
		epoch  types.EpochID
		result []byte
	}{
		{
			name:   "Case 1",
			epoch:  1,
			result: vrfSigner.Sign(util.Hex2Bytes("040404")),
		},
		{
			name:   "Case 2",
			epoch:  2,
			result: vrfSigner.Sign(util.Hex2Bytes("040408")),
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := buildSignedProposal(context.Background(), logtest.New(t), vrfSigner, tc.epoch, types.VRFPostIndex(1))
			require.Equal(t, string(tc.result), string(result))
		})
	}
}

func TestBeacon_signAndExtractED(t *testing.T) {
	r := require.New(t)

	signer, err := signing.NewEdSigner()
	r.NoError(err)
	extractor, err := signing.NewPubKeyExtractor()
	r.NoError(err)

	message := []byte{1, 2, 3, 4}

	signature := signer.Sign(message)
	extractedPK, err := extractor.Extract(message, signature)
	r.NoError(err)

	r.Equal(signer.PublicKey().String(), extractedPK.String())
}

func TestBeacon_calcBeacon(t *testing.T) {
	set := proposalSet{
		"0x1": {},
		"0x2": {},
		"0x4": {},
		"0x5": {},
	}

	beacon := calcBeacon(logtest.New(t), set)
	expected := types.HexToBeacon("0x98f88210")
	require.EqualValues(t, expected, beacon)
}
