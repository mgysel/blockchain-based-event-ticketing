// Package auction implements an F3B sealed-bid auction contract
package auctionF3B

import (
	"fmt"
	"io"
	"strconv"

	"go.dedis.ch/dela"
	"go.dedis.ch/dela/contracts/value"
	"go.dedis.ch/dela/core/access"
	"go.dedis.ch/dela/core/execution"
	"go.dedis.ch/dela/core/execution/native"
	"go.dedis.ch/dela/core/store"
	"golang.org/x/xerrors"
)

// Argument ENUM
const (
	BidLengthKey     string = "auction:bid_length"
	RevealLengthKey         = "auction:reveal_length"
	OwnerKey                = "auction:owner"
	HighestBidderKey        = "auction:highest_bidder"
	HighestBidKey           = "auction:highest_bid"
	BlockNumberKey          = "auction:block_number"
	ListDelimiter           = ";"
	BankKey                 = "bank"
	BidKey                  = "bid"
	DepositKey              = "deposit"
	RevealBidKey            = "reveal:bid"
	RevealNonceKey          = "reveal:nonce"
)

// commands defines the commands of the auction contract. This interface helps in
// testing the contract.
type commands interface {
	init(snap store.Snapshot, step execution.Step) error
	bid(snap store.Snapshot, step execution.Step) error
	selectWinner(snap store.Snapshot, step execution.Step) error
}

const (
	// ContractName is the name of the contract.
	ContractName = "go.dedis.ch/dela.AuctionF3B"

	// InitBidLengthArg is the argument's name in the transaction that contains the
	// bid length.
	InitBidLengthArg = "value:initBidLength"

	// BidArg is the argument's name in the transaction that contains the
	// Hash(bid, nonce).
	BidArg = "value:bid"

	// CmdArg is the argument's name to indicate the kind of command we want to
	// run on the contract. Should be one of the Command type.
	CmdArg = "value:command"

	// credentialAllCommand defines the credential command that is allowed to
	// perform all commands.
	credentialAllCommand = "all"
)

// Command defines a type of command for the auction contract
type Command string

const (
	// CmdBid defines the command to initialize bid length
	CmdInit Command = "INIT"

	// CmdBid defines the command to make a bid
	CmdBid Command = "BID"

	// CmdWinner defines the command to select the auction winner
	CmdSelectWinner Command = "SELECTWINNER"
)

// NewCreds creates new credentials for an auction contract execution. We might
// want to use in the future a separate credential for each command.
func NewCreds(id []byte) access.Credential {
	return access.NewContractCreds(id, ContractName, credentialAllCommand)
}

// RegisterContract registers the auction contract to the given execution service.
func RegisterContract(exec *native.Service, c Contract) {
	exec.Set(ContractName, c)
}

// Contract is a smart contract that allows for sealed-bid auctions
//
// - implements native.Contract
type Contract struct {
	// store is used to store/retrieve data
	store value.Contract

	// access is the access control service managing this smart contract
	access access.Service

	// accessKey is the access identifier allowed to use this smart contract
	accessKey []byte

	// cmd provides the commands that can be executed by this smart contract
	cmd commands

	// printer is the output used by the READ and LIST commands
	printer io.Writer
}

// NewContract creates a new Auction contract
func NewContract(aKey []byte, srvc access.Service) Contract {
	// Create new contract
	contract := Contract{
		store:     value.NewContract(aKey, srvc),
		access:    srvc,
		accessKey: aKey,
		printer:   infoLog{},
	}

	contract.cmd = auctionCommand{Contract: &contract}

	return contract
}

// Execute implements native.Contract. It runs the appropriate command.
func (c Contract) Execute(snap store.Snapshot, step execution.Step) error {
	creds := NewCreds(c.accessKey)

	err := c.access.Match(snap, creds, step.Current.GetIdentity())
	if err != nil {
		return xerrors.Errorf("identity not authorized: %v (%v)",
			step.Current.GetIdentity(), err)
	}

	cmd := step.Current.GetArg(CmdArg)
	if len(cmd) == 0 {
		return xerrors.Errorf("'%s' not found in tx arg", CmdArg)
	}

	switch Command(cmd) {
	case CmdInit:
		err := c.cmd.init(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to INIT: %v", err)
		}
	case CmdBid:
		err := c.cmd.bid(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to BID: %v", err)
		}
	case CmdSelectWinner:
		err := c.cmd.selectWinner(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to SELECTWINNER: %v", err)
		}
	default:
		return xerrors.Errorf("unknown command: %s", cmd)
	}

	return nil
}

// auctionCommand implements the commands of the auction contract
//
// - implements commands
type auctionCommand struct {
	*Contract
}

// ############################################################################
// ############################ INIT FUNCTIONS ################################
// ############################################################################

// init implements commands. It performs the INIT command
// Auction owner initialises bid_length, reveal_length, maximum_bid
func (c auctionCommand) init(snap store.Snapshot, step execution.Step) error {
	// Obtain public key from this txn
	pub_key, err := step.Current.GetIdentity().MarshalText()
	if err != nil {
		return xerrors.Errorf("public key not found in tx arg")
	}

	// Obtain initBidLength from auctionCommand
	initBidLength := step.Current.GetArg(InitBidLengthArg)
	if len(initBidLength) == 0 {
		return xerrors.Errorf("'%s' not found in tx arg", InitBidLengthArg)
	}

	// Store pub_key as auction owner
	//   (auction:owner)
	keyAuctionOwner := []byte(OwnerKey)
	valAuctionOwner := pub_key
	err = snap.Set(keyAuctionOwner, valAuctionOwner)
	if err != nil {
		return xerrors.Errorf("failed to set owner: %v", err)
	}
	// Store bid length arg
	// 	(auction:bid_length)
	keyBidLength := []byte(BidLengthKey)
	valBidLength := initBidLength
	err = snap.Set(keyBidLength, valBidLength)
	if err != nil {
		return xerrors.Errorf("failed to set bid_length: %v", err)
	}

	// Initialise highest bidder
	err = snap.Set([]byte(HighestBidderKey), []byte("-1"))
	if err != nil {
		return xerrors.Errorf("failed to set highest bidder: %v", err)
	}
	// Initialise highest bid
	err = snap.Set([]byte(HighestBidKey), []byte("-1"))
	if err != nil {
		return xerrors.Errorf("failed to set highest bid: %v", err)
	}
	// Initialise block number
	err = snap.Set([]byte(BlockNumberKey), []byte("0"))
	if err != nil {
		return xerrors.Errorf("failed to set block number: %v", err)
	}

	dela.Logger.Info().Str("contract", ContractName).Msgf("setting %x=%s", keyBidLength, valBidLength)

	return nil
}

// ############################################################################
// ############################# BID FUNCTIONS ################################
// ############################################################################

// bid implements commands. It performs the BID command
// User bids Hash(bid, nonce)
func (c auctionCommand) bid(snap store.Snapshot, step execution.Step) error {
	// Check if bid period
	isBidPeriod, err := isValidBidPeriod(snap)
	if err != nil {
		return err
	}
	if !isBidPeriod {
		return xerrors.Errorf("Not valid bid period")
	}

	// Obtain public key from this txn
	pub_key, err := step.Current.GetIdentity().MarshalText()
	if err != nil {
		return xerrors.Errorf("public key not found in tx argument")
	}

	// Check bid from auctionCommand
	bid := step.Current.GetArg(BidArg)
	if len(bid) == 0 {
		return xerrors.Errorf("'%s' not found in tx arg", BidArg)
	}

	// Handle new bid
	// Updates highest bidder/bid and returns deposit
	err = handleNewBid(snap, bid, pub_key)
	if err != nil {
		return xerrors.Errorf("Failed to handle new bid: %s", err)
	}

	// Increment block number
	err = incBlockNumber(snap)
	if err != nil {
		return err
	}

	dela.Logger.Info().Str("contract", ContractName).Msgf("setting %v=%v", pub_key, bid)

	return nil
}

// handleNewBid handles a new bid
// 1. Checks if highest bidder
// 2. If highest bidder, updates bid and returns deposit of old bidder
// 3. If not highest bidder, returns deposit
func handleNewBid(snap store.Snapshot, bid []byte, pk []byte) error {
	isHighest, err := isHighestBidder(snap, bid)
	if err != nil {
		return xerrors.Errorf("Cannot determine highest bidder: %s", err)
	}

	if isHighest {
		// Get old highest bidder
		oldHighestBidder, err := getHighestBidder(snap)
		if err != nil {
			return xerrors.Errorf("Failed to get old highest bidder: %s", err)
		}
		oldHighestBid, err := getHighestBid(snap)
		if err != nil {
			return xerrors.Errorf("Failed to get old highest bid: %s", err)
		}
		// Update highest bid/bidder
		err = setHighestBid(snap, bid)
		if err != nil {
			return xerrors.Errorf("Failed to set new highest bid: %s", err)
		}
		err = setHighestBidder(snap, pk)
		if err != nil {
			return xerrors.Errorf("Failed to set new highest bidder: %s", err)
		}

		// Refund old highest bid
		oldHighestBidInt, err := byteToInt(oldHighestBid)
		if err != nil {
			return xerrors.Errorf("Failed to convert old highest bid to integer: %s", err)
		}
		err = refundDeposit(snap, oldHighestBidder, oldHighestBidInt)
	}

	return nil
}

func getBid(snap store.Snapshot, bidder []byte) ([]byte, error) {
	// Get reveal bid from database
	bidKey := []byte(fmt.Sprintf("%s:%s", string(bidder), BidKey))
	bid, err := snap.Get(bidKey)
	if err != nil {
		return []byte{}, xerrors.Errorf("failed to get reveal bid", err)
	}

	return bid, nil
}

// storeBid stores a bid from a bidder
func storeBid(snap store.Snapshot, bidder []byte, bid []byte) error {
	key := []byte(fmt.Sprintf("%s:%s", string(bidder), "bid"))
	val := bid
	err := snap.Set(key, val)
	if err != nil {
		return xerrors.Errorf("failed to set bid: %v", err)
	}

	return nil
}

// Compares a bid to the highest bid
// Returns true if this bid is higher than the highest bid, false otherwise
func isHighestBidder(snap store.Snapshot, bid []byte) (bool, error) {
	// Get highestBid in integer format
	highestBidBytes, err := getHighestBid(snap)
	if err != nil {
		xerrors.Errorf("failed to get highest bid: %v", err)
	}
	highestBid, err := byteToInt(highestBidBytes)
	if err != nil {
		xerrors.Errorf("failed to convert highest bid to int: %v", err)
	}

	// Get this bid in integer format
	thisBid, err := byteToInt(bid)
	if err != nil {
		xerrors.Errorf("failed to convert bid to int: %v", err)
	}

	// If this bid is greater than the highest bid, return true
	if thisBid > highestBid {
		return true, nil
	}
	return false, nil
}

// refundDeposit refunds a deposit from a bidder
// bid:deposit:bid# = deposit
func refundDeposit(snap store.Snapshot, bidder []byte, deposit int) error {
	// Add deposit to user bank
	bankKey := []byte(fmt.Sprintf("bank:%s", string(bidder)))
	balanceByte, err := snap.Get(bankKey)
	if err != nil || len(balanceByte) == 0 {
		balanceByte = []byte("0")
	}
	balance, err := byteToInt(balanceByte)
	if err != nil {
		return xerrors.Errorf("failed to convert user deposit to int: %v", err)
	}
	newBalance := balance + deposit
	newBalanceByte := []byte(strconv.Itoa(newBalance))
	err = snap.Set(bankKey, []byte(newBalanceByte))
	if err != nil {
		return xerrors.Errorf("failed to set new user balance: %v", err)
	}

	return nil
}

// Gets the highest bid in []byte format]
func getHighestBid(snap store.Snapshot) ([]byte, error) {
	// Get highest bid from database
	highestBidKey := []byte(fmt.Sprintf(HighestBidKey))
	highestBid, err := snap.Get(highestBidKey)
	if err != nil {
		return []byte{}, xerrors.Errorf("failed to get highest bid", err)
	}

	return highestBid, nil
}

// Sets the highest bid
func setHighestBid(snap store.Snapshot, bid []byte) error {
	// Get highest bid from database
	highestBidKey := []byte(fmt.Sprintf(HighestBidKey))
	err := snap.Set(highestBidKey, bid)
	if err != nil {
		return xerrors.Errorf("failed to set highest bid", err)
	}

	return nil
}

// Gets the highest bidder
func getHighestBidder(snap store.Snapshot) ([]byte, error) {
	// Get highest bid from database
	highestBidderKey := []byte(fmt.Sprintf(HighestBidderKey))
	highestBidder, err := snap.Get(highestBidderKey)
	if err != nil {
		return []byte{}, xerrors.Errorf("failed to get highest bid", err)
	}

	return highestBidder, nil
}

// Sets the highest bidder
func setHighestBidder(snap store.Snapshot, pk []byte) error {
	// Get highest bid from database
	highestBidderKey := []byte(fmt.Sprintf(HighestBidderKey))
	err := snap.Set(highestBidderKey, pk)
	if err != nil {
		return xerrors.Errorf("failed to set highest bidder", err)
	}

	return nil
}

// ############################################################################
// ######################## SELECTWINNER FUNCTIONS ############################
// ############################################################################

// selectWinner implements commands. It performs the SELECTWINNER command
// Auction SC searches for the highest reveal, ensures it matches bid, and selects that winner
func (c auctionCommand) selectWinner(snap store.Snapshot, step execution.Step) error {
	// Ensure tx pk is contract owner
	pub_key, err := step.Current.GetIdentity().MarshalText()
	if err != nil {
		return xerrors.Errorf("Could not obtain pk from tx")
	}
	isOwner, err := isAuctionOwner(snap, pub_key)
	if !isOwner {
		return xerrors.Errorf("selectWinner not called by contract owner")
	}

	// Ensure auction is complete
	isAuctionOver, err := isAuctionOver(snap)
	if !isAuctionOver {
		return xerrors.Errorf("Auction is not over")
	}

	// Get highest bidder, highest bid
	highestBidder, err := getHighestBidder(snap)
	if err != nil {
		return xerrors.Errorf("Could not get highest bidder")
	}
	highestBid, err := getHighestBid(snap)
	if err != nil {
		return xerrors.Errorf("Could not get highest bid")
	}

	output := "Highest Bidder: " + string(highestBidder) + ", Highest Bid: " + string(highestBid)
	fmt.Fprint(c.printer, output)

	// dela.Logger.Info().Str("contract", ContractName).Msgf("setting Highest Bidder=%s", highestBidder)

	return nil
}

// ############################################################################
// ########################## HELPER FUNCTIONS ################################
// ############################################################################

// Gets bid length
// Used to determine if in bidding period
func getBidLength(snap store.Snapshot) (int, error) {
	// Get Bid Length
	key := []byte(BidLengthKey)
	bidLengthByte, err := snap.Get(key)
	if err != nil {
		return -1, xerrors.Errorf("failed to get bid length", err)
	}

	// Convert Bid Length to int
	bidLength, err := strconv.Atoi(string(bidLengthByte))
	if err != nil {
		return -1, xerrors.Errorf("failed to convert bid_length to int: %v. Error: %v", err)
	}

	return bidLength, nil
}

// Gets the current block number
// Block number is defined as the number of bid or reveal tx that have taken place
func getBlockNumber(snap store.Snapshot) (int, error) {
	// Get block number from database
	key := []byte(BlockNumberKey)
	blockNumberByte, err := snap.Get(key)
	if err != nil {
		return -1, xerrors.Errorf("failed to get block_number", err)
	}

	// Convert block number to int
	// Convert reveal to integer
	blockNumber, err := strconv.Atoi(string(blockNumberByte))
	if err != nil {
		return -1, xerrors.Errorf("failed to convert block_number to int: %v. Error: %v", err)
	}

	return blockNumber, nil
}

// Used to increment block number after bid or reveal tx
func incBlockNumber(snap store.Snapshot) error {
	// Get block number from database
	blockNumber, err := getBlockNumber(snap)
	if err != nil {
		return xerrors.Errorf("failed to get block number: %v. Error: %v", err)
	}

	// Inc blockNumber
	blockNumber = blockNumber + 1
	err = snap.Set([]byte(BlockNumberKey), []byte(strconv.Itoa(blockNumber)))
	if err != nil {
		return xerrors.Errorf("failed to set increment block number: %v", err)
	}

	return nil
}

// Checks if contract is in bidding period
func isValidBidPeriod(snap store.Snapshot) (bool, error) {
	blockNumber, err := getBlockNumber(snap)
	if err != nil {
		return false, err
	}

	bidPeriod, err := getBidLength(snap)
	if err != nil {
		return false, err
	}

	if blockNumber < bidPeriod {
		return true, nil
	}

	return false, nil
}

// Checks if auction is over
// Used to determine if auction owner can selectWinner
func isAuctionOver(snap store.Snapshot) (bool, error) {
	blockNumber, err := getBlockNumber(snap)
	if err != nil {
		return false, err
	}

	// Get bid and reveal periods
	bidPeriod, err := getBidLength(snap)
	if err != nil {
		return false, err
	}

	// Check if blockNumber between bid and reveal periods
	if blockNumber >= bidPeriod {
		return true, nil
	}

	return false, nil
}

func isAuctionOwner(snap store.Snapshot, pk []byte) (bool, error) {
	// Get auction owner
	owner, err := snap.Get([]byte(OwnerKey))
	if err != nil {
		return false, xerrors.Errorf("owner not found in store")
	}

	// Check if pk matches auction owner
	if string(owner) == string(pk) {
		return true, nil
	} else {
		return false, nil
	}
}

// Converts a byte array to an integer
// Returns error if cannot be converted
func byteToInt(Byte []byte) (int, error) {
	deposit, err := strconv.Atoi(string(Byte))
	if err != nil {
		return -1, xerrors.Errorf("Failed to convert Byte Array to int", err)
	}

	return deposit, nil
}

// infoLog defines an output using zerolog
//
// - implements io.writer
type infoLog struct{}

func (h infoLog) Write(p []byte) (int, error) {
	dela.Logger.Info().Msg(string(p))

	return len(p), nil
}
