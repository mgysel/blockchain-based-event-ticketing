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
	"go.dedis.ch/dela/crypto"
	"go.dedis.ch/dela/crypto/bls"
	"go.dedis.ch/dela/crypto/ed25519"
	"go.dedis.ch/dela/crypto/loader"
	"go.dedis.ch/dela/dkg"
	"go.dedis.ch/dela/dkg/pedersen"
	"go.dedis.ch/dela/internal/testing/fake"

	"go.dedis.ch/dela/mino"

	"go.dedis.ch/kyber/v3"
)

func init() {
	rand.Seed(0)
}

func Test_Identification(t *testing.T) {
	batchSizes := []int{15}
	numDKGs := []int{2}
	withGrpc := false

	for _, batchSize := range batchSizes {
		for _, numDKG := range numDKGs {
			t.Run(fmt.Sprintf("batch size %d num dkg %d", batchSize, numDKG),
				identificationScenario(batchSize, numDKG, 2, withGrpc))
		}
	}
}

func identificationScenario(batchSize, numDKG, numNodes int, withGrpc bool) func(t *testing.T) {
	return func(t *testing.T) {

		require.Greater(t, numDKG, 0)
		require.Greater(t, numNodes, 0)
		require.GreaterOrEqual(t, numDKG, numNodes)

		to := time.Second * 10 // transaction inclusion timeout

		// set up the dkg
		minosBuilder := getMinoch
		if withGrpc {
			minosBuilder = getMinogRPCs
		}

		minos := minosBuilder(t, numDKG)
		dkgs := make([]dkg.DKG, numDKG)
		addrs := make([]mino.Address, numDKG)

		// initializing the addresses
		for i, mino := range minos {
			addrs[i] = mino.GetAddress()
		}

		pubkeys := make([]kyber.Point, len(minos))
		bdnPubkeys := make([]kyber.Point, len(minos))

		for i, mino := range minos {
			dkg, pubkey, bdnPubkey := pedersen.NewPedersen(mino)
			dkgs[i] = dkg
			pubkeys[i] = pubkey
			bdnPubkeys[i] = bdnPubkey
		}

		actors := make([]dkg.Actor, numDKG)
		for i := 0; i < numDKG; i++ {
			actor, err := dkgs[i].Listen()
			require.NoError(t, err)
			actors[i] = actor
		}

		fakeAuthority := NewAuthority(addrs, pubkeys)
		fakeAuthorityBls := NewAuthorityBls(addrs, bdnPubkeys)

		start := time.Now()
		_, err := actors[0].Setup(fakeAuthority, fakeAuthorityBls, numDKG)
		require.NoError(t, err)

		setupTime := time.Since(start)
		fmt.Println("setup done in ", setupTime)
		t.Logf("setup done in %s", setupTime)

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

		// sending the grant transaction to the blockchain
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

		// generate idHashes used for credentials
		idHashes := make([]string, batchSize)
		for i := range idHashes {
			idHashes[i] = fmt.Sprintf("%d", i)
			require.NoError(t, err)
		}

		// Issue Master Credential
		fmt.Println("Issuing Master Credential")
		mcs := make([][]byte, 0)
		mss := make([][][]byte, 0)
		start = time.Now()
		for i := 0; i < batchSize; i++ {
			mc, ms, err := actors[0].IssueMasterCredential(idHashes[i])
			require.NoError(t, err)
			mcs = append(mcs, mc)
			mss = append(mss, ms)
		}
		mcTime := time.Since(start)
		t.Logf("issue master credential: %s", mcTime)

		// Issue Event Credential
		fmt.Println("Issuing Event Credential")
		eventName := "Event"
		ecs := make([][]byte, 0)
		ess := make([][][]byte, 0)
		start = time.Now()
		for i := 0; i < batchSize; i++ {
			ec, es, err := actors[0].IssueEventCredential(idHashes[i], eventName, mcs[i], mss[i])
			require.NoError(t, err)
			ecs = append(ecs, ec)
			ess = append(ess, es)
		}
		ecTime := time.Since(start)
		t.Logf("issue event credential: %s", ecTime)

		// Verify Event Credential
		fmt.Println("Verifying Event Credential")
		start = time.Now()
		for i := 0; i < batchSize; i++ {
			_, err := actors[0].VerifyEventCredential(idHashes[i], eventName, ecs[i], ess[i])
			require.NoError(t, err)
		}
		veTime := time.Since(start)
		t.Logf("verify event credential: %s", veTime)

		fmt.Println("Average Time Taken")
		fmt.Println("Issue Master Credential: ", float64(mcTime.Milliseconds())/float64(batchSize))
		fmt.Println("Issue Event Credential: ", float64(ecTime.Milliseconds())/float64(batchSize))
		fmt.Println("Verify Event Credential: ", float64(veTime.Milliseconds())/float64(batchSize))
	}
}

// -----------------------------------------------------------------------------
// Utility functions

//
// Collective authority
//

// CollectiveAuthority is a fake implementation of the cosi.CollectiveAuthority
// interface.
type CollectiveAuthority struct {
	crypto.CollectiveAuthority
	addrs   []mino.Address
	pubkeys []kyber.Point
	signers []crypto.Signer
}

// NewAuthority returns a new collective authority of n members with new signers
// generated by g.
func NewAuthority(addrs []mino.Address, pubkeys []kyber.Point) CollectiveAuthority {
	fmt.Println("*** Inside NewAuthority")
	fmt.Println("pubkeys: ", pubkeys)
	signers := make([]crypto.Signer, len(pubkeys))
	for i, pubkey := range pubkeys {
		signers[i] = newFakeSigner(pubkey)
	}
	fmt.Println("signers: ", signers)

	return CollectiveAuthority{
		pubkeys: pubkeys,
		addrs:   addrs,
		signers: signers,
	}
}

// GetPublicKey implements cosi.CollectiveAuthority.
func (ca CollectiveAuthority) GetPublicKey(addr mino.Address) (crypto.PublicKey, int) {
	fmt.Println("*** Inside GetPublicKey")

	for i, address := range ca.addrs {
		if address.Equal(addr) {
			return ed25519.NewPublicKeyFromPoint(ca.pubkeys[i]), i
		}
	}
	return nil, -1
}

// Len implements mino.Players.
func (ca CollectiveAuthority) Len() int {
	return len(ca.pubkeys)
}

// AddressIterator implements mino.Players.
func (ca CollectiveAuthority) AddressIterator() mino.AddressIterator {
	return fake.NewAddressIterator(ca.addrs)
}

func (ca CollectiveAuthority) PublicKeyIterator() crypto.PublicKeyIterator {
	return fake.NewPublicKeyIterator(ca.signers)
}

func newFakeSigner(pubkey kyber.Point) fakeSigner {
	return fakeSigner{
		pubkey: pubkey,
	}
}

// fakeSigner is a fake signer
//
// - implements crypto.Signer
type fakeSigner struct {
	crypto.Signer
	pubkey kyber.Point
}

// GetPublicKey implements crypto.Signer
func (s fakeSigner) GetPublicKey() crypto.PublicKey {
	return ed25519.NewPublicKeyFromPoint(s.pubkey)
}

//
// Collective authority BLS
//

// CollectiveAuthorityBls is a fake implementation of the cosi.CollectiveAuthority
// interface.
type CollectiveAuthorityBls struct {
	crypto.CollectiveAuthority
	addrs   []mino.Address
	pubkeys []kyber.Point
	signers []crypto.Signer
}

// NewAuthority returns a new collective authority of n members with new signers
// generated by g.
func NewAuthorityBls(addrs []mino.Address, blsPubkeys []kyber.Point) CollectiveAuthorityBls {
	fmt.Println("*** Inside NewAuthorityBls")
	fmt.Println("Bls pub keys: ", blsPubkeys)

	signers := make([]crypto.Signer, len(addrs))
	for i, pubkey := range blsPubkeys {
		signers[i] = newFakeSignerBls(pubkey)
	}

	fmt.Println("Bls Signers: ", signers)

	return CollectiveAuthorityBls{
		pubkeys: blsPubkeys,
		addrs:   addrs,
		signers: signers,
	}
}

// GetPublicKey implements cosi.CollectiveAuthority.
func (ca CollectiveAuthorityBls) GetPublicKey(addr mino.Address) (crypto.PublicKey, int) {
	fmt.Println("*** Inside GetPublicKey")

	for i, address := range ca.addrs {
		if address.Equal(addr) {
			return bls.NewPublicKeyFromPoint(ca.pubkeys[i]), i
		}
	}
	return nil, -1
}

// Len implements mino.Players.
func (ca CollectiveAuthorityBls) Len() int {
	return len(ca.pubkeys)
}

// AddressIterator implements mino.Players.
func (ca CollectiveAuthorityBls) AddressIterator() mino.AddressIterator {
	return fake.NewAddressIterator(ca.addrs)
}

func (ca CollectiveAuthorityBls) PublicKeyIterator() crypto.PublicKeyIterator {
	return fake.NewPublicKeyIterator(ca.signers)
}

func newFakeSignerBls(pubkey kyber.Point) fakeSignerBls {
	return fakeSignerBls{
		pubkey: pubkey,
	}
}

// fakeSigner is a fake signer
//
// - implements crypto.Signer
type fakeSignerBls struct {
	crypto.Signer
	pubkey kyber.Point
}

// GetPublicKey implements crypto.Signer
func (s fakeSignerBls) GetPublicKey() crypto.PublicKey {
	return bls.NewPublicKeyFromPoint(s.pubkey)
}
