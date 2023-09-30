package pedersen

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/sign/bdn"
)

func TestBdn_Signatures(t *testing.T) {
	num_nodes := 3
	sks := make([]kyber.Scalar, num_nodes)
	pks := make([]kyber.Point, num_nodes)

	// Create sk/pk pairs
	for i := 0; i < num_nodes; i++ {
		// sks[i], pks[i] = bdn.NewKeyPair(bdn.NewSuite(), bdn.NewZeroReader())
		sks[i], pks[i] = bdn.NewKeyPair(pairing.NewSuiteBn256(), suite.RandomStream())
	}
	t.Log("*** Creating sk/pk pairs")
	t.Log("sks:", sks)
	t.Log("pks:", pks)

	msg := []byte("Hello World!")

	// Create individual signatures
	t.Logf("*** Creating individual signatures")
	sigs := make([][]byte, num_nodes)
	for i := 0; i < num_nodes; i++ {
		sigs[i], _ = bdn.Sign(pairing.NewSuiteBn256(), sks[i], msg)
	}
	t.Logf("sigs: %v", sigs)

	// // Create aggregate signature
	// aggSigs := make([][]byte, num_nodes)
	// for i := 0; i < num_nodes; i++ {
	// 	thisAggSig, err := bls.AggregateSignatures(suite, sigs...)
	// 	aggSigs[i] = thisAggSig
	// 	t.Log("Error: ", err)
	// }

	// t.Log("*** Creating aggregate signatures")
	// t.Logf("aggSigs: %v", aggSigs)

	// Verify individual signatures
	msg = []byte("Hello World!")
	for i := 0; i < num_nodes; i++ {
		err := bdn.Verify(pairing.NewSuiteBn256(), pks[i], msg, sigs[i])
		require.NoError(t, err)
	}
}
