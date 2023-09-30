package auction

import (
	"fmt"
	"strings"
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
	reveal_length := "2"

	// Check error when no bid_length, or reveal_length
	err := cmd.init(fake.NewBadSnapshot(), makeStep(t, signer, InitBidLengthArg, bid_length, InitRevealLengthArg, reveal_length))
	require.EqualError(t, err, fake.Err("failed to set owner"))
	err = cmd.init(fake.NewSnapshot(), makeStep(t, signer, InitRevealLengthArg, reveal_length))
	require.EqualError(t, err, "'value:initBidLength' not found in tx arg")
	err = cmd.init(fake.NewSnapshot(), makeStep(t, signer, InitBidLengthArg, bid_length))
	require.EqualError(t, err, "'value:initRevealLength' not found in tx arg")

	// Correct init
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitBidLengthArg, bid_length, InitRevealLengthArg, reveal_length)
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

	// Check store for (auction:reveal_length)
	key = []byte("auction:reveal_length")
	val, err = snap.Get(key)
	val_res = string(val)
	require.Equal(t, reveal_length, val_res)

	// Check store for (auction:bidders)
	key = []byte("auction:bidders")
	val, err = snap.Get(key)
	val_res = string(val)
	require.Equal(t, val_res, "")

	// Check store for (auction:highest_bidder)
	key = []byte("auction:highest_bidder")
	val, err = snap.Get(key)
	val_res = string(val)
	require.Equal(t, val_res, "-1")
}

func TestCommand_Bid(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "2"
	reveal_length := "2"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitBidLengthArg, bid_length, InitRevealLengthArg, reveal_length)
	err := cmd.init(snap, step)

	// Bid Deposit, Hash(Bid, Nonce)
	bidDeposit := "1"
	bidBid := []byte("1")
	bidNonce := []byte("Nonce")
	bidByte, err := contract.HashReveal(bidBid, bidNonce)
	bid := string(bidByte)
	step = makeStep(t, signer, BidArg, bid, BidDepositArg, bidDeposit)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// Check store for (bid:bid:0, Hash(Bid, Nonce))
	pub_key, err := step.Current.GetIdentity().MarshalText()
	key := []byte(fmt.Sprintf("%s:%s:%s", "bid", "bid", "0"))
	bidResByte, err := snap.Get(key)
	bidRes := string(bidResByte)
	require.Equal(t, bid, bidRes)

	// Check store for (bid:deposit:0, deposit)
	key = []byte(fmt.Sprintf("%s:%s:%s", "bid", "deposit", "0"))
	bidDepositResByte, err := snap.Get(key)
	bidDepositRes := string(bidDepositResByte)
	require.Equal(t, bidDeposit, bidDepositRes)

	// Check store for (auction:bidders, pub_key)
	key = []byte("auction:bidders")
	val, err := snap.Get(key)
	val_res := string(val)
	require.Equal(t, fmt.Sprintf("%s;", string(pub_key)), val_res)
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
	reveal_length := "1"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitBidLengthArg, bid_length, InitRevealLengthArg, reveal_length)
	err := cmd.init(snap, step)

	// Bid Hash(Bid, Nonce)
	bidDeposit := "1"
	bidBid := []byte("1")
	bidNonce := []byte("Nonce")
	bidByte, err := contract.HashReveal(bidBid, bidNonce)
	bid := string(bidByte)
	step = makeStep(t, signer1, BidArg, bid, BidDepositArg, bidDeposit)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// Second bid should give an error
	bidDeposit = "2"
	bidBid = []byte("2")
	bidNonce = []byte("Nonce")
	bidByte, err = contract.HashReveal(bidBid, bidNonce)
	bid = string(bidByte)
	step = makeStep(t, signer2, BidArg, bid, BidDepositArg, bidDeposit)
	err = cmd.bid(snap, step)
	require.EqualError(t, err, "Not valid bid period")
}

func TestCommand_RevealInputs(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := fake.NewSigner()

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "1"
	reveal_length := "2"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitBidLengthArg, bid_length, InitRevealLengthArg, reveal_length)
	err := cmd.init(snap, step)

	// Make bid
	bidDeposit := "1"
	bidBid := []byte("1")
	bidNonce := []byte("Nonce")
	bidByte, err := contract.HashReveal(bidBid, bidNonce)
	bid := string(bidByte)
	step = makeStep(t, signer, BidArg, bid, BidDepositArg, bidDeposit)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// RevealBid Errors
	err = cmd.reveal(snap, makeStep(t, signer))
	require.EqualError(t, err, "'value:revealBid' not found in tx arg")
	err = cmd.reveal(snap, makeStep(t, signer, RevealBidArg, "Bid"))
	require.EqualError(t, err, "'value:revealNonce' not found in tx arg")
	err = cmd.reveal(snap, makeStep(t, signer, RevealNonceArg, "Nonce"))
	require.EqualError(t, err, "'value:revealBid' not found in tx arg")

	// RevalNonce Errors
	err = cmd.reveal(snap, makeStep(t, signer, RevealBidArg, "Bid"))
	require.EqualError(t, err, "'value:revealNonce' not found in tx arg")
}

func TestCommand_RevealOneBid(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "1"
	reveal_length := "2"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitBidLengthArg, bid_length, InitRevealLengthArg, reveal_length)
	err := cmd.init(snap, step)

	// Make bid
	bidDeposit := "1"
	bidBid := "1"
	bidNonce := "Nonce"
	bid, err := contract.HashReveal([]byte(bidBid), []byte(bidNonce))
	step = makeStep(t, signer, BidArg, string(bid), BidDepositArg, bidDeposit)
	err = cmd.bid(snap, step)
	require.NoError(t, err)

	// Reveal Bid, Nonce with no error
	// Submit Reveal
	revealBid := "1"
	revealNonce := "Nonce"
	step = makeStep(t, signer, RevealBidArg, revealBid, RevealNonceArg, revealNonce)
	// Check store for (pub_key:reveal:bid, bid)
	pub_key, err := step.Current.GetIdentity().MarshalText()
	// Check Reveal Storage
	err = cmd.reveal(snap, step)
	require.NoError(t, err)

	// Check reveal set correctly
	key := []byte(fmt.Sprintf("reveal:bid:0"))
	val, err := snap.Get(key)
	revealBidRes := string(val)
	require.Equal(t, revealBid, revealBidRes)

	// Check revealers list set correctly
	key = []byte("auction:revealers")
	val, err = snap.Get(key)
	require.Equal(t, string(pub_key), strings.Split(string(val), ";")[0])
}

func TestCommand_RevealMultipleBids(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := bls.NewSigner()
	signer2 := bls.NewSigner()

	cmd := auctionCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	bid_length := "2"
	reveal_length := "2"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitBidLengthArg, bid_length, InitRevealLengthArg, reveal_length)
	err := cmd.init(snap, step)

	// Make first bid
	bidDeposit := "1"
	bidBid := "1"
	bidNonce := "Nonce"
	bid, err := contract.HashReveal([]byte(bidBid), []byte(bidNonce))
	step = makeStep(t, signer1, BidArg, string(bid), BidDepositArg, bidDeposit)
	err = cmd.bid(snap, step)

	// Make second bid
	bidDeposit = "2"
	bidBid = "2"
	bidNonce = "Nonce"
	bid, err = contract.HashReveal([]byte(bidBid), []byte(bidNonce))
	step = makeStep(t, signer2, BidArg, string(bid), BidDepositArg, string(bidDeposit))
	err = cmd.bid(snap, step)

	// First reveal
	revealBid := "1"
	revealNonce := "Nonce"
	step = makeStep(t, signer1, RevealBidArg, revealBid, RevealNonceArg, revealNonce)
	// Check Reveal Storage
	err = cmd.reveal(snap, step)
	require.NoError(t, err)
	// Check store for (pub_key:reveal:bid, bid)
	key := []byte(fmt.Sprintf("reveal:bid:0"))
	val, err := snap.Get(key)
	revealBidRes := string(val)
	require.Equal(t, revealBid, revealBidRes)

	// Second reveal
	revealBid = "2"
	revealNonce = "Nonce"
	step = makeStep(t, signer2, RevealBidArg, revealBid, RevealNonceArg, revealNonce)
	// Check Reveal Storage
	err = cmd.reveal(snap, step)
	require.NoError(t, err)
	// Check store for (pub_key:reveal:bid, bid)
	key = []byte(fmt.Sprintf("reveal:bid:1"))
	val, err = snap.Get(key)
	revealBidRes = string(val)
	require.Equal(t, revealBid, revealBidRes)

	// Check revealers list
	signer1_pk, err := signer1.GetPublicKey().MarshalText()
	signer2_pk, err := signer2.GetPublicKey().MarshalText()
	key = []byte("auction:revealers")
	val, err = snap.Get(key)
	revealers_list := strings.Split(string(val), ";")
	require.Equal(t, string(signer1_pk), revealers_list[0])
	require.Equal(t, string(signer2_pk), revealers_list[1])
}

func TestCommand_HighestBidder_AuctionNotOver(t *testing.T) {
	auctionContract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()

	cmd := auctionCommand{
		Contract: &auctionContract,
	}

	// Initialize smart contract
	bid_length := "1"
	reveal_length := "2"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitBidLengthArg, bid_length, InitRevealLengthArg, reveal_length)
	err := cmd.init(snap, step)

	// Check auction not over error
	step = makeStep(t, signer)
	err = cmd.selectWinner(snap, step)
	require.EqualError(t, err, "Auction is not over")
}

func TestCommand_HighestBidder(t *testing.T) {
	auctionContract := NewContract([]byte{}, fakeAccess{})
	signer1 := bls.NewSigner()
	signer2 := bls.NewSigner()

	auctionCmd := auctionCommand{
		Contract: &auctionContract,
	}

	// Initialize smart contract
	bid_length := "2"
	reveal_length := "2"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitBidLengthArg, bid_length, InitRevealLengthArg, reveal_length)
	err := auctionCmd.init(snap, step)

	// Bid 1
	bidDeposit := "2"
	bidBid := "1"
	bidNonce := "Nonce"
	bidBytes, err := auctionContract.HashReveal([]byte(bidBid), []byte(bidNonce))
	bid := string(bidBytes)
	step = makeStep(t, signer1, BidArg, bid, BidDepositArg, bidDeposit)
	err = auctionCmd.bid(snap, step)
	require.NoError(t, err)

	// Bid 2
	bidDeposit = "2"
	bidBid = "2"
	bidNonce = "Nonce"
	bidBytes, err = auctionContract.HashReveal([]byte(bidBid), []byte(bidNonce))
	bid = string(bidBytes)
	step = makeStep(t, signer2, BidArg, bid, BidDepositArg, bidDeposit)
	err = auctionCmd.bid(snap, step)
	require.NoError(t, err)

	// Reveal 1
	revealBid := "1"
	revealNonce := "Nonce"
	step = makeStep(t, signer1, RevealBidArg, revealBid, RevealNonceArg, revealNonce)
	err = auctionCmd.reveal(snap, step)
	require.NoError(t, err)

	// Reveal 2
	revealBid = "2"
	revealNonce = "Nonce"
	step = makeStep(t, signer2, RevealBidArg, revealBid, RevealNonceArg, revealNonce)
	err = auctionCmd.reveal(snap, step)
	require.NoError(t, err)

	// Check non-contract owner cannot call
	step = makeStep(t, signer2)
	err = auctionCmd.selectWinner(snap, step)
	require.EqualError(t, err, "selectWinner not called by contract owner")

	// Check no errors
	step = makeStep(t, signer1)
	err = auctionCmd.selectWinner(snap, step)
	require.NoError(t, err)

	// Check correct highest bidder
	key := []byte("auction:highest_bidder")
	highestBidder, err := snap.Get(key)
	pk_two, err := signer2.GetPublicKey().MarshalText()
	require.Equal(t, highestBidder, pk_two)

	// Check correct highest bid
	key = []byte("auction:highest_bid")
	highestBid, err := snap.Get(key)
	require.Equal(t, string(highestBid), "2")
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
