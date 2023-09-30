package integration

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "net/http/pprof"

	"github.com/stretchr/testify/require"
	accessContract "go.dedis.ch/dela/contracts/access"
	"go.dedis.ch/dela/core/txn"
	"go.dedis.ch/dela/core/txn/signed"
	"go.dedis.ch/dela/crypto/bls"
	"go.dedis.ch/dela/crypto/loader"
)

func init() {
	rand.Seed(0)
}

func Test_Event_Throughput(t *testing.T) {
	batchSizes := []int{100}
	numDKGs := []int{64}
	withGrpc := false
	numNodes := 32

	for _, batchSize := range batchSizes {
		for _, numDKG := range numDKGs {
			t.Run(fmt.Sprintf("batch size %d num dkg %d", batchSize, numDKG),
				eventThroughputScenario(batchSize, numDKG, numNodes, withGrpc))
		}
	}
}

func eventThroughputScenario(batchSize, numDKG, numNodes int, withGrpc bool) func(t *testing.T) {
	return func(t *testing.T) {

		require.Greater(t, numDKG, 0)
		require.Greater(t, numNodes, 0)
		require.GreaterOrEqual(t, numDKG, numNodes)

		to := time.Second * 10 // transaction inclusion timeout

		// setting up the blockchain

		dir, err := os.MkdirTemp("", "dela-integration-test")
		require.NoError(t, err)

		t.Logf("using temps dir %s", dir)

		defer os.RemoveAll(dir)

		nodes := make([]dela, numNodes)

		for i := range nodes {
			nodes[i] = newDelaNode(t, filepath.Join(dir, fmt.Sprintf("node%d", i)), 0)
		}

		nodes[0].Setup(nodes[1:]...)

		l := loader.NewFileLoader(filepath.Join(dir, "private.key"))

		// creating a new client/signer
		signerdata, err := l.LoadOrCreate(newKeyGenerator())
		require.NoError(t, err)

		signer, err := bls.NewSignerFromBytes(signerdata)
		require.NoError(t, err)

		pubKey := signer.GetPublicKey()
		cred := accessContract.NewCreds(aKey[:])

		for _, node := range nodes {
			node.GetAccessService().Grant(node.(cosiDelaNode).GetAccessStore(), cred, pubKey)
		}

		manager := signed.NewManager(signer, &txClient{})

		pubKeyBuf, err := signer.GetPublicKey().MarshalBinary()
		require.NoError(t, err)
		pubKeyByte, err := signer.GetPublicKey().MarshalText()
		require.NoError(t, err)
		pubKeyString := string(pubKeyByte)

		// Giving access to value contract
		args := []txn.Arg{
			{Key: "go.dedis.ch/dela.ContractArg", Value: []byte("go.dedis.ch/dela.Access")},
			{Key: "access:grant_id", Value: []byte(hex.EncodeToString(valueAccessKey[:]))},
			{Key: "access:grant_contract", Value: []byte("go.dedis.ch/dela.Value")},
			{Key: "access:grant_command", Value: []byte("all")},
			{Key: "access:identity", Value: []byte(base64.StdEncoding.EncodeToString(pubKeyBuf))},
			{Key: "access:command", Value: []byte("GRANT")},
		}

		// waiting for the confirmation of the transaction
		err = addAndWait(t, to, manager, nodes[0].(cosiDelaNode), args...)
		require.NoError(t, err)

		// Giving access to event contract
		args = []txn.Arg{
			{Key: "go.dedis.ch/dela.ContractArg", Value: []byte("go.dedis.ch/dela.Access")},
			{Key: "access:grant_id", Value: []byte(hex.EncodeToString(valueAccessKey[:]))},
			{Key: "access:grant_contract", Value: []byte("go.dedis.ch/dela.Event")},
			{Key: "access:grant_command", Value: []byte("all")},
			{Key: "access:identity", Value: []byte(base64.StdEncoding.EncodeToString(pubKeyBuf))},
			{Key: "access:command", Value: []byte("GRANT")},
		}
		err = addAndWait(t, to, manager, nodes[0].(cosiDelaNode), args...)
		require.NoError(t, err)

		// Initialize Event
		name, numTickets, price, maxResalePrice, resaleRoyalty := "Event Name", "5000", "50", "50", "10"
		args = getInitCommandArgs(pubKeyString, name, numTickets, price, maxResalePrice, resaleRoyalty)
		err = addAndWait(t, to, manager, nodes[0].(cosiDelaNode), args...)
		require.NoError(t, err)

		fmt.Println("Buy Transactions")
		start := time.Now()
		for i := 0; i < batchSize; i++ {
			payment := "50"
			numTickets = "1"
			args = getBuyCommandArgs(pubKeyString, numTickets, payment)
			err = addAndWait(t, to, manager, nodes[0].(cosiDelaNode), args...)
			require.NoError(t, err)
		}
		buyTime := time.Since(start)
		t.Logf("Buy transactions: %s", buyTime)

		fmt.Println("Average Time Taken")
		fmt.Println("Buy Transactions: ", float64(buyTime.Milliseconds())/float64(batchSize))
	}
}
