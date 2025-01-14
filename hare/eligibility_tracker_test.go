package hare_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/spacemeshos/go-spacemesh/hare"
	"github.com/spacemeshos/go-spacemesh/signing"
)

func TestEligibilityTracker(t *testing.T) {
	const (
		totalNodes        = 5
		count      uint16 = 2
	)
	et := hare.NewEligibilityTracker(totalNodes)
	rounds := []uint32{0, 1, 2, 3, 4}
	pubKeys := map[string]bool{}
	for i := 0; i < totalNodes; i++ {
		honest := false
		if i%2 == 0 {
			honest = true
		}
		sig, err := signing.NewEdSigner()
		require.NoError(t, err)
		pubKeys[string(sig.PublicKey().Bytes())] = honest
		for _, r := range rounds {
			et.Track(sig.PublicKey().Bytes(), r, count, honest)
		}
	}
	for _, r := range rounds {
		total := 0
		good := 0
		et.ForEach(r, func(k string, cred *hare.Cred) {
			total++
			honest, ok := pubKeys[k]
			require.True(t, ok)
			require.Equal(t, honest, cred.Honest)
			require.Equal(t, count, cred.Count)
			if cred.Honest {
				good++
			}
		})
		require.Equal(t, totalNodes, total)
		require.Equal(t, 3, good)
	}

	// update everyone to be honest have no effect
	for key := range pubKeys {
		for _, r := range rounds {
			et.Track([]byte(key), r, count, true)
		}
	}
	for _, r := range rounds {
		total := 0
		good := 0
		et.ForEach(r, func(k string, cred *hare.Cred) {
			total++
			honest, ok := pubKeys[k]
			require.True(t, ok)
			require.Equal(t, honest, cred.Honest)
			require.Equal(t, count, cred.Count)
			if cred.Honest {
				good++
			}
		})
		require.Equal(t, totalNodes, total)
		require.Equal(t, 3, good)
	}

	// update everyone to be dishonest will update the tracker
	for key := range pubKeys {
		for _, r := range rounds {
			et.Track([]byte(key), r, count, false)
		}
	}
	for _, r := range rounds {
		total := 0
		good := 0
		et.ForEach(r, func(k string, cred *hare.Cred) {
			total++
			_, ok := pubKeys[k]
			require.True(t, ok)
			require.Equal(t, false, cred.Honest)
			require.Equal(t, count, cred.Count)
			if cred.Honest {
				good++
			}
		})
		require.Equal(t, totalNodes, total)
		require.Equal(t, 0, good)
	}
}
