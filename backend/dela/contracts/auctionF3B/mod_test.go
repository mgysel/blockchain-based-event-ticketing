package auctionF3B

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/core/access"
	"go.dedis.ch/dela/core/execution"
	"go.dedis.ch/dela/core/execution/native"
	"go.dedis.ch/dela/core/store"
	"go.dedis.ch/dela/core/txn"
	"go.dedis.ch/dela/core/txn/signed"
	"go.dedis.ch/dela/crypto"
	"go.dedis.ch/dela/crypto/bls"
	"go.dedis.ch/dela/internal/testing/fake"
)

func TestRegisterContract(t *testing.T) {
	RegisterContract(native.NewExecution(), Contract{})
}

func TestExecuteSuccess(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := fake.NewSigner()

	contract.cmd = fakeCmd{}
	err := contract.Execute(fakeStore{}, makeStep(t, signer, CmdArg, "BID"))
	require.NoError(t, err)
}

func TestCommand_Init(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()
	pk, _ := signer.GetPublicKey().MarshalText()

	cmd := auctionCommand{
		Contract: &contract,
	}

	bid_length := "2"

	// Check error when no bid_length, or reveal_length
	err := cmd.init(fake.NewBadSnapshot(), makeStep(t, signer, InitBidLengthArg, bid_length))
	require.EqualError(t, err, fake.Err("failed to set owner"))
	err = cmd.init(fake.NewSnapshot(), makeStep(t, signer))
	require.EqualError(t, err, "'value:initBidLength' not found in tx arg")

	// Correct init
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitBidLengthArg, bid_length)
	err = cmd.init(snap, step)
	require.NoError(t, err)

	// Check store for (auction:owner)
	key := []byte("auction:owner")
	val, err := snap.Get(key)
	val_res := string(val)
	require.Equal(t, string(pk), val_res)

	// Check store for (auction:block_number)
	key = []byte("auction:block_number")
	val, err = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "0", val_res)

	// Check store for (auction:bid_length)
	key = []byte("auction:bid_length")
	val, err = snap.Get(key)
	val_res = string(val)
	require.Equal(t, bid_length, val_res)

	// Check store for (auction:highest_bidder)
	key = []byte("auction:highest_bidder")
	val, err = snap.Get(key)
	val_res = string(val)
	require.Equal(t, val_res, "-1")
}

func TestCommand_Bid(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()
	pkByte, err := signer.GetPublicKey().MarshalText()
	pk := string(pkByte)

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "2"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitBidLengthArg, bid_length)
	err = cmd.init(snap, step)

	// Bid with no error
	bid := "1"
	step = makeStep(t, signer, BidArg, bid)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// Check store for (auction:highest_bid, 1)
	key := []byte("auction:highest_bid")
	bidResByte, err := snap.Get(key)
	bidRes := string(bidResByte)
	require.Equal(t, bidRes, bid)

	// Check store for (auction:highestBidder, pk)
	key = []byte("auction:highest_bidder")
	bidderResByte, err := snap.Get(key)
	bidderRes := string(bidderResByte)
	require.Equal(t, bidderRes, pk)
}

func TestCommand_Bid_NotPeriod(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := fake.NewSigner()
	signer2 := fake.NewSigner()

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "1"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitBidLengthArg, bid_length)
	err := cmd.init(snap, step)

	// Bid Hash(Bid, Nonce) with no error
	bid := "1"
	step = makeStep(t, signer1, BidArg, bid)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// Second bid should give an error
	bid = "2"
	step = makeStep(t, signer2, BidArg, bid)
	err = cmd.bid(snap, step)
	require.EqualError(t, err, "Not valid bid period")
}

func TestCommand_Multiple_Bidders(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := fake.NewSigner()
	signer2 := fake.NewSigner()
	pk2, err := signer2.GetPublicKey().MarshalText()

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "2"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitBidLengthArg, bid_length)
	err = cmd.init(snap, step)

	// First Bid
	bid1 := "1"
	step = makeStep(t, signer1, BidArg, bid1)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// Second Bid
	bid2 := "2"
	step = makeStep(t, signer2, BidArg, bid2)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// Highest Bidder and Bid should be second bidder
	// Check store for (auction:highest_bid, 2)
	key := []byte("auction:highest_bid")
	bidResByte, err := snap.Get(key)
	bidRes := string(bidResByte)
	require.Equal(t, bidRes, bid2)

	// Check store for (auction:highestBidder, pk)
	key = []byte("auction:highest_bidder")
	bidderResByte, err := snap.Get(key)
	bidderRes := string(bidderResByte)
	require.Equal(t, bidderRes, string(pk2))
}

func TestCommand_HighestBidder_AuctionNotOver(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "1"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitBidLengthArg, bid_length)
	err := cmd.init(snap, step)

	// Check auction not over error
	step = makeStep(t, signer)
	err = cmd.selectWinner(snap, step)
	require.EqualError(t, err, "Auction is not over")
}

func TestCommand_HighestBidder_NotOwner(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := bls.NewSigner()
	signer2 := bls.NewSigner()

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "1"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitBidLengthArg, bid_length)
	err := cmd.init(snap, step)

	// Check auction not owner error
	step = makeStep(t, signer2)
	err = cmd.selectWinner(snap, step)
	require.EqualError(t, err, "selectWinner not called by contract owner")
}

func TestCommand_HighestBidder(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := bls.NewSigner()
	signer2 := bls.NewSigner()
	pk1, err := signer1.GetPublicKey().MarshalText()

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "2"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitBidLengthArg, bid_length)
	err = cmd.init(snap, step)

	// First Bid
	bid1 := "2"
	step = makeStep(t, signer1, BidArg, bid1)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// Second Bid
	bid2 := "1"
	step = makeStep(t, signer2, BidArg, bid2)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// Check no error
	step = makeStep(t, signer1)
	err = cmd.selectWinner(snap, step)
	require.NoError(t, err)

	// Highest Bidder and Bid should be second bidder
	// Check store for (auction:highest_bid, 2)
	key := []byte("auction:highest_bid")
	bidResByte, err := snap.Get(key)
	bidRes := string(bidResByte)
	require.Equal(t, bidRes, bid1)

	// Check store for (auction:highestBidder, pk)
	key = []byte("auction:highest_bidder")
	bidderResByte, err := snap.Get(key)
	bidderRes := string(bidderResByte)
	require.Equal(t, bidderRes, string(pk1))
}

func TestInfoLog(t *testing.T) {
	log := infoLog{}

	n, err := log.Write([]byte{0b0, 0b1})
	require.NoError(t, err)
	require.Equal(t, 2, n)
}

// -----------------------------------------------------------------------------
// Utility functions

func makeStep(t *testing.T, signer crypto.Signer, args ...string) execution.Step {
	return execution.Step{Current: makeTx(t, signer, args...)}
}

func makeTx(t *testing.T, signer crypto.Signer, args ...string) txn.Transaction {
	// t.Log("INSIDE MAKE TX")
	// pub_key, err := signer.GetPublicKey().MarshalBinary()
	// t.Log("SIGNER PUBLIC KEY: ", string(pub_key))
	options := []signed.TransactionOption{}
	for i := 0; i < len(args)-1; i += 2 {
		options = append(options, signed.WithArg(args[i], []byte(args[i+1])))
	}

	tx, err := signed.NewTransaction(0, signer.GetPublicKey(), options...)
	require.NoError(t, err)

	return tx
}

type fakeAccess struct {
	access.Service

	err error
}

func (srvc fakeAccess) Match(store.Readable, access.Credential, ...access.Identity) error {
	return srvc.err
}

func (srvc fakeAccess) Grant(store.Snapshot, access.Credential, ...access.Identity) error {
	return srvc.err
}

type fakeStore struct {
	store.Snapshot
}

func (s fakeStore) Get(key []byte) ([]byte, error) {
	return nil, nil
}

func (s fakeStore) Set(key, value []byte) error {
	return nil
}

type fakeCmd struct {
	err error
}

func (c fakeCmd) init(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) bid(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) reveal(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) selectWinner(snap store.Snapshot, step execution.Step) error {
	return c.err
}
