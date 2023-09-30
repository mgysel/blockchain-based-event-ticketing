// Package event implements a traditional sealed-bid auction contract
package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.dedis.ch/dela"
	"go.dedis.ch/dela/contracts/value"
	"go.dedis.ch/dela/core/access"
	"go.dedis.ch/dela/core/execution"
	"go.dedis.ch/dela/core/execution/native"
	"go.dedis.ch/dela/core/store"
	"go.dedis.ch/dela/crypto"
	"golang.org/x/xerrors"
)

// Argument ENUM
const (
	BlockNumberKey = "event:block_number"
	ListDelimiter  = ";"
	BankKey        = "bank"
	BidKey         = "bid"
	DepositKey     = "deposit"
	RevealBidKey   = "reveal:bid"
	RevealNonceKey = "reveal:nonce"

	TXCountKey = "event:txcount"

	OwnerKey                  = "event:owner"
	UserIndexKey              = "event:users"
	NameKey                   = "event:name"
	NumTicketsKey             = "event:num_tickets"
	PriceKey                  = "event:price"
	MaxResalePriceKey         = "event:max_resale_price"
	ResaleRoyaltyKey          = "event:resale_royalty"
	TicketOwnersKey           = "event:ticket_owners"
	TicketResellersKey        = "event:ticket_resellers"
	TicketRebuyersKey         = "event:ticket_rebuyers"
	BuyerTicketsKey           = "event:buyer_tickets"
	BuyerEventCredentialKey   = "event:buyer_event_credential"
	ResellerTicketsNumberKey  = "event:reseller_tickets_number"
	ResellerTicketsPriceKey   = "event:reseller_tickets_price"
	RebuyerTicketsNumberKey   = "event:rebuyer_tickets_number"
	RebuyerTicketsPriceKey    = "event:rebuyer_tickets_price"
	RebuyerEventCredentialKey = "event:rebuyer_event_credential"
	AttendeesKey              = "event:attendees"

	NumTicketsLeftKey = "event:num_tickets_left"

	EventBalanceKey = "event:balance"
)

// commands defines the commands of the event contract. This interface helps in
// testing the contract.
type commands interface {
	init(snap store.Snapshot, step execution.Step) error
	buy(snap store.Snapshot, step execution.Step) error
	resell(snap store.Snapshot, step execution.Step) error
	rebuy(snap store.Snapshot, step execution.Step) error
	handleResales(snap store.Snapshot, step execution.Step) error
	useTicket(snap store.Snapshot, step execution.Step) error
	readEventContract(snap store.Snapshot, step execution.Step) (string, error)
}

const (
	// ContractName is the name of the contract.
	ContractName = "go.dedis.ch/dela.Event"

	// INIT
	// InitPKArg is the pk of the event organizer
	InitPKArg = "value:initPK"
	// InitNameArg is the name of the event
	InitNameArg = "value:initName"
	// InitNumTicketsArg is the number of tickets sold
	InitNumTicketsArg = "value:initNumTickets"
	// InitPriceArg is the price per ticket
	InitPriceArg = "value:initPrice"
	// InitMaxResalePriceArg is the max resale price per ticket
	InitMaxResalePriceArg = "value:initMaxResalePrice"
	// InitResaleRoyaltyArg is the resale royalty the event organizer receives on each ticket resale
	InitResaleRoyaltyArg = "value:initResaleRoyalty"

	// BUY
	BuyPKArg              = "value:buyPK"
	BuyNumTicketsArg      = "value:buyNumTickets"
	BuyPaymentArg         = "value:buyPayment"
	BuyEventCredentialArg = "value:buyEventCredential"

	// RESELL
	ResellPKArg         = "value:resellPK"
	ResellNumTicketsArg = "value:resellNumTickets"
	ResellPriceArg      = "value:resellPrice"

	// REBUY
	RebuyPKArg              = "value:rebuyPK"
	RebuyNumTicketsArg      = "value:rebuyNumTickets"
	RebuyPriceArg           = "value:rebuyPrice"
	RebuyEventCredentialArg = "value:rebuyEventCredential"

	// HANDLERESALES
	HandleResalesPKArg = "value:handleResalesPK"

	// USETICKET
	UseTicketPKArg              = "value:useTicketPK"
	UseTicketNumTicketsArg      = "value:useTicketNumTickets"
	UseTicketEventCredentialArg = "value:useTicketEventCredential"

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
	// CmdBid defines the command to initialize Event SC Values
	CmdInit Command = "INIT"

	// CmdBuy defines the command to buy a ticket
	CmdBuy Command = "BUY"

	// CmdResell defines the command to resell a ticket
	CmdResell Command = "RESELL"

	// CmdRebuy defines the command to buy a secondary market ticket
	CmdRebuy Command = "REBUY"

	// CmdRebuy defines the command to handle all secondary transactions
	CmdHandleResales Command = "HANDLERESALES"

	// CmdUseTicket uses ticket from user
	CmdUseTicket Command = "USETICKET"

	// CmdReadEventContract reads all data from the event contract
	CmdReadEventContract Command = "READEVENTCONTRACT"
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

	// Used to hash reveals to check against bids
	hashFactory crypto.HashFactory

	// cmd provides the commands that can be executed by this smart contract
	cmd commands

	// printer is the output used by the READ and LIST commands
	printer io.Writer
}

// NewContract creates a new Auction contract
func NewContract(aKey []byte, srvc access.Service) Contract {
	// Create new contract
	contract := Contract{
		store:       value.NewContract(aKey, srvc),
		access:      srvc,
		hashFactory: crypto.NewSha256Factory(),
		accessKey:   aKey,
		printer:     infoLog{},
	}

	contract.cmd = eventCommand{Contract: &contract}

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
	case CmdBuy:
		err := c.cmd.buy(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to BUY: %v", err)
		}
	case CmdResell:
		err := c.cmd.resell(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to RESELL: %v", err)
		}
	case CmdRebuy:
		err := c.cmd.rebuy(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to REBUY: %v", err)
		}
	case CmdHandleResales:
		err := c.cmd.handleResales(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to REBUY: %v", err)
		}
	case CmdUseTicket:
		err := c.cmd.useTicket(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to USETICKET: %v", err)
		}
	case CmdReadEventContract:
		_, err := c.cmd.readEventContract(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to READEVENTCONTRACT: %v", err)
		}
	default:
		return xerrors.Errorf("unknown command: %s", cmd)
	}

	return nil
}

// eventCommand implements the commands of the auction contract
//
// - implements commands
type eventCommand struct {
	*Contract
}

// ############################################################################
// ############################ INIT FUNCTIONS ################################
// ############################################################################

// init implements commands. It performs the INIT command
// Event organizer initialises event name, num_tickets, price, max_resale_price, resale_royalty
func (c eventCommand) init(snap store.Snapshot, step execution.Step) error {
	// Initialize tx count to 1
	err := snap.Set([]byte(TXCountKey), []byte("1"))
	txcount := "1"
	if err != nil {
		errMessage := fmt.Sprintf("failed to set txcount to 1: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Check that received all init commands from auctionCommand
	initPK := step.Current.GetArg(InitPKArg)
	if len(initPK) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", InitPKArg)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	initName := step.Current.GetArg(InitNameArg)
	if len(initName) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", InitNameArg)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	initNumTickets := step.Current.GetArg(InitNumTicketsArg)
	if len(initNumTickets) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", InitNumTicketsArg)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	initPrice := step.Current.GetArg(InitPriceArg)
	if len(initPrice) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", InitPriceArg)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	initMaxResalePrice := step.Current.GetArg(InitMaxResalePriceArg)
	if len(initMaxResalePrice) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", InitMaxResalePriceArg)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	initResaleRoyalty := step.Current.GetArg(InitResaleRoyaltyArg)
	if len(initResaleRoyalty) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", InitResaleRoyaltyArg)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Check that variables are valid
	_, err = isValidPrice(initPrice)
	if err != nil {
		errMessage := fmt.Sprintf("Invalid Price: '%s'", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	_, err = isValidNumTickets(initNumTickets)
	if err != nil {
		errMessage := fmt.Sprintf("Invalid number of tickets: '%s'", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	err = isValidMaxResalePrice(initMaxResalePrice)
	if err != nil {
		errMessage := fmt.Sprintf("Invalid maximum resale price: '%s'", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	err = isValidResaleRoyalty(initResaleRoyalty)
	if err != nil {
		errMessage := fmt.Sprintf("Invalid resale royalty: '%s'", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Store pub_key as auction owner
	//   (event:owner)
	keyEventOwner := []byte(OwnerKey)
	valEventOwner := initPK
	err = snap.Set(keyEventOwner, valEventOwner)
	if err != nil {
		errMessage := fmt.Sprintf("failed to set event owner: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Store event name arg
	err = snap.Set([]byte(NameKey), initName)
	if err != nil {
		errMessage := fmt.Sprintf("failed to set event name: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Store number of tickets
	err = snap.Set([]byte(NumTicketsKey), initNumTickets)
	if err != nil {
		errMessage := fmt.Sprintf("failed to set number of tickets: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Store price
	err = snap.Set([]byte(PriceKey), initPrice)
	if err != nil {
		errMessage := fmt.Sprintf("failed to set ticket price: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Store max resale price
	err = snap.Set([]byte(MaxResalePriceKey), initMaxResalePrice)
	if err != nil {
		errMessage := fmt.Sprintf("failed to set max resale price: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Store resale royalty
	err = snap.Set([]byte(ResaleRoyaltyKey), initResaleRoyalty)
	if err != nil {
		errMessage := fmt.Sprintf("failed to set resale royalty: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Create empty byte array for event attendees
	err = snap.Set([]byte(AttendeesKey), []byte(""))
	if err != nil {
		errMessage := fmt.Sprintf("failed to set attendees array: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Set number of tickets remaining to number of tickets
	err = snap.Set([]byte(NumTicketsLeftKey), initNumTickets)
	if err != nil {
		errMessage := fmt.Sprintf("failed to set number of tickets remaining: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Initialize event account to 0
	err = snap.Set([]byte(EventBalanceKey), []byte("0"))
	if err != nil {
		errMessage := fmt.Sprintf("failed to set initialize event balance to 0: %v", err)
		fmt.Fprintln(c.printer, outputInitFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	fmt.Fprintln(c.printer, outputInitSuccess("1", string(initPK), string(initName), string(initNumTickets), string(initPrice), string(initMaxResalePrice), string(initResaleRoyalty)))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputInitSuccess("1", string(initPK), string(initName), string(initNumTickets), string(initPrice), string(initMaxResalePrice), string(initResaleRoyalty)))

	return nil
}

func outputInitSuccess(txcount string, pk string, eventName string, numTickets string, price string, maxResalePrice string, resaleRoyalty string) string {
	return fmt.Sprintf("//EVENTCONTRACT_INITOUTPUT;success;%s;%s;%s;%s;%s;%s;%s//", txcount, pk, eventName, numTickets, price, maxResalePrice, resaleRoyalty)
}

func outputInitFailure(txcount string, errMessage string) string {
	return fmt.Sprintf("//EVENTCONTRACT_INITOUTPUT;error;%s;%s//", txcount, errMessage)
}

func isValidPrice(priceByte []byte) (float64, error) {
	// Convert to int
	price, err := byteToDecimal(priceByte)
	if err != nil {
		return -1, xerrors.Errorf("price must be an integer")
	}

	// Check valid price
	if price < 0 {
		return -1, xerrors.Errorf("price cannot be a negative integer")
	}

	return price, nil
}

func isValidResalePrice(snap store.Snapshot, priceByte []byte) (float64, error) {
	// Check if valid price
	resalePrice, err := isValidPrice(priceByte)
	if err != nil {
		return -1, err
	}

	// Get max resale price
	maxResalePriceByte, err := snap.Get([]byte(MaxResalePriceKey))
	if err != nil {
		return -1, xerrors.Errorf("failed to get event max resale price: '%s'", err)
	}
	maxResalePrice, err := byteToDecimal(maxResalePriceByte)
	if err != nil {
		return -1, xerrors.Errorf("failed to convert max resale price to integer: '%s'", err)
	}

	// Check that resalePrice > maxResalePrice
	if resalePrice > maxResalePrice {
		return -1, xerrors.Errorf("resale price cannot be greater than max resale price")
	}

	return resalePrice, nil
}

func isValidNumTickets(numTicketsByte []byte) (int, error) {
	// Convert to int
	numTickets, err := byteToInt(numTicketsByte)
	if err != nil {
		return -1, xerrors.Errorf("number of tickets must be an integer")
	}

	// Check valid number of tickets
	if numTickets <= 0 {
		return -1, xerrors.Errorf("number of tickets must be a positive integer")
	}

	return numTickets, nil
}

func isValidMaxResalePrice(maxResalePriceByte []byte) error {
	// Convert to int
	maxResalePrice, err := byteToDecimal(maxResalePriceByte)
	if err != nil {
		return xerrors.Errorf("resale price must be an integer")
	}

	// Check valid max resale price
	if maxResalePrice <= 0 {
		return xerrors.Errorf("resale price cannot be a negative integer")
	}

	return nil
}

func isValidResaleRoyalty(resaleRoyaltyByte []byte) error {
	// Convert to int
	resaleRoyalty, err := byteToDecimal(resaleRoyaltyByte)
	if err != nil {
		return xerrors.Errorf("resale price must be an integer")
	}

	// Check valid resale royalty
	if resaleRoyalty < 0 || resaleRoyalty > 100 {
		return xerrors.Errorf("resale royalty must be between 0 and 100")
	}

	return nil
}

// ############################################################################
// ############################# BUY FUNCTIONS ################################
// ############################################################################

// buy implements commands. It performs the BUY command
// User buys ticket(s)
func (c eventCommand) buy(snap store.Snapshot, step execution.Step) error {
	incTxcount(snap)
	txcountByte, err := snap.Get([]byte(TXCountKey))
	txcount := string(txcountByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to increase txcount: %s", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Obtain elements from eventCommand
	pk := step.Current.GetArg(BuyPKArg)
	if len(pk) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", BuyPKArg)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	eventCredential := step.Current.GetArg(BuyEventCredentialArg)
	if len(eventCredential) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", BuyEventCredentialArg)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	buyPaymentByte := step.Current.GetArg(BuyPaymentArg)
	if len(buyPaymentByte) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", BuyPaymentArg)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	buyPayment, err := byteToDecimal(buyPaymentByte)
	if err != nil {
		errMessage := fmt.Sprintf("Could not convert ticket payment to float64: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	buyNumTicketsByte := step.Current.GetArg(BuyNumTicketsArg)
	if len(buyNumTicketsByte) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", BuyPaymentArg)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	buyNumTickets, err := byteToInt(buyNumTicketsByte)
	if err != nil {
		errMessage := fmt.Sprintf("Could not convert number of tickets to int: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Check payment is greater than price
	ticketPriceByte, err := snap.Get([]byte(PriceKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get ticket price: '%s'", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	ticketPrice, err := byteToDecimal(ticketPriceByte)
	if err != nil {
		errMessage := fmt.Sprintf("Could not convert ticket price to float64: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	if buyPayment < ticketPrice {
		errMessage := "Payment must be at least the price of each ticket"
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Check if enough tickets left
	isTicketsLeft, err := isTicketsLeft(snap, buyNumTickets)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to obtain number of tickets left: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	if !isTicketsLeft {
		errMessage := "No tickets left"
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Store ticket user
	err = storeTicketUser(snap, pk)
	if err != nil {
		errMessage := fmt.Sprintf("failed to store new user: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return err
	}

	// Transfer payment to event account
	ticketPayment := float64(buyNumTickets) * buyPayment
	err = transferSalePayment(snap, string(pk), ticketPayment)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to transfer payment: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Store ticket owner
	err = storeTicketOwner(snap, pk)
	if err != nil {
		errMessage := fmt.Sprintf("failed to store ticket owner: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return err
	}

	// Increase buyer number of tickets
	err = changeBuyerNumTickets(snap, pk, buyNumTickets)
	if err != nil {
		errMessage := fmt.Sprintf("failed to change the buyer's number of tickets: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return err
	}

	// Store buyer event credential
	err = storeBuyerEventCredential(snap, pk, eventCredential)
	if err != nil {
		errMessage := fmt.Sprintf("failed to store buyer event credential: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		return err
	}

	// Decrease number of tickets left
	err = changeNumTicketsLeft(snap, -1*buyNumTickets)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to decrease number of tickets left: %v", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return err
	}

	// Get event name for output
	eventNameByte, err := snap.Get([]byte(NameKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get event name: '%s'", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	eventName := string(eventNameByte)

	fmt.Fprintln(c.printer, outputBuySuccess(txcount, eventName, string(pk), buyNumTickets, buyPayment))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuySuccess(txcount, eventName, string(pk), buyNumTickets, buyPayment))

	return nil
}

func outputBuySuccess(txcount string, eventName string, pk string, numTickets int, payment float64) string {
	return fmt.Sprintf("//EVENTCONTRACT_BUYOUTPUT;success;%s;%s;%d;%s//", txcount, eventName, numTickets, fmt.Sprintf("%f", payment))
}

func outputBuyFailure(txcount string, errMessage string) string {
	return fmt.Sprintf("//EVENTCONTRACT_BUYOUTPUT;error;%s;%s//", txcount, errMessage)
}

// ############################################################################
// ########################### RESELL FUNCTIONS ###############################
// ############################################################################

// resell implements commands. It performs the RESELL command
// User places a ticket up for resale
func (c eventCommand) resell(snap store.Snapshot, step execution.Step) error {
	incTxcount(snap)
	txcountByte, err := snap.Get([]byte(TXCountKey))
	txcount := string(txcountByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to increase txcount: %s", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Get pk from command
	pk := step.Current.GetArg(ResellPKArg)
	if len(pk) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", ResellPKArg)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Check if pub_key has ticket
	numTicketsReseller, err := hasNumTickets(snap, pk)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to obtain number of tickets for reseller: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Check number of tickets reseller is reselling
	resellNumTicketsByte := step.Current.GetArg(ResellNumTicketsArg)
	if len(resellNumTicketsByte) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", ResellNumTicketsArg)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	resellNumTickets, err := byteToInt(resellNumTicketsByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to convert number of tickets to integer: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Check has atleast that number of tickets
	if numTicketsReseller < resellNumTickets {
		errMessage := fmt.Sprintf("Invalid resale number of tickets: reseller does not have %d tickets", resellNumTickets)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Check valid resale price
	resellPriceByte := step.Current.GetArg(ResellPriceArg)
	if len(resellPriceByte) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", ResellPriceArg)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	resellPrice, err := isValidResalePrice(snap, resellPriceByte)
	if err != nil {
		errMessage := fmt.Sprintf("Invalid resale price: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Store reseller, number of tickets for sale, and price
	err = storeTicketReseller(snap, pk)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to store ticket reseller: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	err = changeResellerNumTickets(snap, pk, resellNumTickets)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to store number of tickets for reseller: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	err = storeResellerPrice(snap, pk, resellPrice)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to store resell price for reseller: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Get event name
	eventNameByte, err := snap.Get([]byte(NameKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get event name: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	eventName := string(eventNameByte)

	fmt.Fprintln(c.printer, outputResellSuccess(txcount, eventName, string(pk), resellNumTickets, resellPrice))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputResellSuccess(txcount, eventName, string(pk), resellNumTickets, resellPrice))

	return nil
}

func outputResellSuccess(txcount string, eventName string, pk string, numTickets int, price float64) string {
	return fmt.Sprintf("//EVENTCONTRACT_RESELLOUTPUT;success;%s;%s;%d;%s//", txcount, eventName, numTickets, fmt.Sprintf("%f", price))
}

func outputResellFailure(txcount string, errMessage string) string {
	return fmt.Sprintf("//EVENTCONTRACT_RESELLOUTPUT;error;%s;%s//", txcount, errMessage)
}

// ############################################################################
// ############################ REBUY FUNCTIONS ###############################
// ############################################################################

// rebuy implements commands. It performs the REBUY command
// User purchases a ticket on the secondary ticket market
func (c eventCommand) rebuy(snap store.Snapshot, step execution.Step) error {
	incTxcount(snap)
	txcountByte, err := snap.Get([]byte(TXCountKey))
	txcount := string(txcountByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to increase txcount: %s", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Get pk from command
	pk := step.Current.GetArg(RebuyPKArg)
	if len(pk) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", RebuyPKArg)
		fmt.Fprintln(c.printer, outputRebuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Get event credential from command
	eventCredential := step.Current.GetArg(RebuyEventCredentialArg)
	if len(eventCredential) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", RebuyEventCredentialArg)
		fmt.Fprintln(c.printer, outputRebuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Add rebuyer as user
	err = storeTicketUser(snap, pk)
	if err != nil {
		errMessage := fmt.Sprintln("Failed to store rebuyer as user")
		fmt.Fprintln(c.printer, outputRebuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
	}
	// Check valid rebuy price
	rebuyPriceByte := step.Current.GetArg(RebuyPriceArg)
	if len(rebuyPriceByte) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", ResellPriceArg)
		fmt.Fprintln(c.printer, outputRebuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	rebuyPrice, err := isValidPrice(rebuyPriceByte)
	if err != nil {
		errMessage := fmt.Sprintf("Invalid rebuy price: '%s'", err)
		fmt.Fprintln(c.printer, outputRebuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Check valid number of tickets
	rebuyNumTicketsByte := step.Current.GetArg(RebuyNumTicketsArg)
	if len(rebuyNumTicketsByte) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", ResellPriceArg)
		fmt.Fprintln(c.printer, outputRebuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	rebuyNumTickets, err := isValidNumTickets(rebuyNumTicketsByte)
	if err != nil {
		errMessage := fmt.Sprintf("Invalid number of tickets: '%s'", err)
		fmt.Fprintln(c.printer, outputRebuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Store rebuyer, number of tickets for sale, price, and event credential
	err = storeTicketRebuyer(snap, pk)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to store ticket reseller: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	err = changeRebuyerNumTickets(snap, pk, rebuyNumTickets)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to store number of tickets for reseller: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	err = storeRebuyerPrice(snap, pk, rebuyPrice)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to store resell price for rebuyer: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	err = storeRebuyerEventCredential(snap, pk, eventCredential)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to store event credential for rebuyer: '%s'", err)
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		fmt.Fprintln(c.printer, outputResellFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Get event name
	eventNameByte, err := snap.Get([]byte(NameKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get event name: '%s'", err)
		fmt.Fprintln(c.printer, outputRebuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	eventName := string(eventNameByte)

	fmt.Fprintln(c.printer, outputRebuySuccess(txcount, eventName, string(pk), rebuyNumTickets, rebuyPrice))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuySuccess(txcount, eventName, string(pk), rebuyNumTickets, rebuyPrice))
	return nil
}

func outputRebuySuccess(txcount string, eventName string, pk string, numTickets int, price float64) string {
	return fmt.Sprintf("//EVENTCONTRACT_REBUYOUTPUT;success;%s;%s;%d;%s//", txcount, eventName, numTickets, fmt.Sprintf("%f", price))
}

func outputRebuyFailure(txcount string, errMessage string) string {
	return fmt.Sprintf("//EVENTCONTRACT_REBUYOUTPUT;error;%s;%s//", txcount, errMessage)
}

// ############################################################################
// ####################### HANDLE RESALES FUNCTIONS ###########################
// ############################################################################

// handleResales implements commands. It performs the HANDLERESALES command
// Event owner can initiate secondary market transactions
func (c eventCommand) handleResales(snap store.Snapshot, step execution.Step) error {
	incTxcount(snap)
	txcountByte, err := snap.Get([]byte(TXCountKey))
	txcount := string(txcountByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to increase txcount: %s", err)
		fmt.Fprintln(c.printer, outputBuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputBuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Make sure event owner is calling this function
	// Get pk from command
	pk := step.Current.GetArg(HandleResalesPKArg)
	if len(pk) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", HandleResalesPKArg)
		fmt.Fprintln(c.printer, outputRebuyFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputRebuyFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Get event owner from snapshot
	eventOwnerByte, err := snap.Get([]byte(OwnerKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get event owner: '%s'", err)
		fmt.Fprintln(c.printer, outputHandleResalesFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputHandleResalesFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Check if pk is event owner
	if string(eventOwnerByte) != string(pk) {
		errMessage := fmt.Sprintf("'%s' is not the event owner. '%s' is the event owner", pk, string(eventOwnerByte))
		fmt.Fprintln(c.printer, outputHandleResalesFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputHandleResalesFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Get rebuyers, number of tickets, price, and eventCredentials
	rebuyers, rebuyersNumTickets, rebuyersPrice, rebuyersEventCredential, err := getRebuyersNumTicketsPriceEC(snap)
	if err != nil {
		errMessage := fmt.Sprintf("Could not get rebuyers, number of tickets, and price: %s", err)
		fmt.Fprintln(c.printer, outputHandleResalesFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputHandleResalesFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Loop through rebuyers, find resellers, transfer tickets
	for i, thisRebuyer := range rebuyers {
		thisRebuyerNumTickets := rebuyersNumTickets[i]
		thisRebuyerPrice := rebuyersPrice[i]

		// Loop through resellers to find number of tickets, transfer ownership
		numTicketsBought := 0
		// Get resellers
		resellers, _, _, _ := getResellersNumTicketsPrice(snap)
		for _, thisReseller := range resellers {
			thisResellerNumTickets, _ := getResellerNumTickets(snap, []byte(thisReseller))
			thisResellerPrice, _ := getResellerPrice(snap, []byte(thisReseller))

			if thisRebuyerPrice < thisResellerPrice {
				continue
			}
			if thisRebuyerNumTickets < thisResellerNumTickets {
				// If this reseller has more than enough tickets
				// Transfer ticket payment from rebuyer to reseller
				transferAmount := float64(thisRebuyerNumTickets) * thisResellerPrice
				transferResalePayment(snap, thisRebuyer, thisReseller, transferAmount)
				// Buy tickets, add owner
				numTicketsBought = numTicketsBought + thisRebuyerNumTickets
				storeTicketOwner(snap, []byte(thisRebuyer))
				changeBuyerNumTickets(snap, []byte(thisRebuyer), thisRebuyerNumTickets)
				storeBuyerEventCredential(snap, []byte(thisRebuyer), []byte(rebuyersEventCredential[i]))
				// Reduce rebuyNumTickets from resellers and owners list
				changeBuyerNumTickets(snap, []byte(thisReseller), -1*thisRebuyerNumTickets)
				changeResellerNumTickets(snap, []byte(thisReseller), -1*thisRebuyerNumTickets)
				removeOwnerFromRebuyers(snap, []byte(thisRebuyer))
			} else {
				// If at most enough tickets from this reseller
				numTicketsBought = numTicketsBought + thisResellerNumTickets
				// Transfer ticket payment from rebuyer to reseller
				transferAmount := float64(thisResellerNumTickets) * thisResellerPrice
				transferResalePayment(snap, thisRebuyer, thisReseller, transferAmount)
				// Remove this reseller, remove number of tickets
				removeOwnerFromOwners(snap, []byte(thisReseller))
				removeOwnerFromResellers(snap, []byte(thisReseller))
				// Store new ticket owner, add number of tickets
				storeTicketOwner(snap, []byte(thisRebuyer))
				changeBuyerNumTickets(snap, []byte(thisRebuyer), thisResellerNumTickets)
				storeBuyerEventCredential(snap, []byte(thisRebuyer), []byte(rebuyersEventCredential[i]))
				// Remove old ticket owner, number of tickets
				changeBuyerNumTickets(snap, []byte(thisReseller), -1*thisResellerNumTickets)
				changeResellerNumTickets(snap, []byte(thisReseller), -1*thisResellerNumTickets)
				if thisRebuyerNumTickets == thisResellerNumTickets {
					removeOwnerFromRebuyers(snap, []byte(thisRebuyer))
				}
			}

			if numTicketsBought >= thisRebuyerNumTickets {
				break
			}
		}
	}

	// Get event name
	eventNameByte, err := snap.Get([]byte(NameKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get event name: '%s'", err)
		fmt.Fprintln(c.printer, outputHandleResalesFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputHandleResalesFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	eventName := string(eventNameByte)

	fmt.Fprintln(c.printer, outputHandleResalesSuccess(txcount, eventName))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputHandleResalesSuccess(txcount, eventName))
	return nil
}

func getRebuyersNumTicketsPriceEC(snap store.Snapshot) ([]string, []int, []float64, []string, error) {
	// Get rebuyers
	rebuyersList, err := getRebuyersList(snap)
	if err != nil {
		errMessage := fmt.Sprintf("Could not get rebuyerslist: %s", err)
		return nil, nil, nil, nil, xerrors.Errorf(errMessage)
	}
	// Get rebuyers, number of tickets, price
	rebuyers := []string{}
	rebuyersNumTickets := []int{}
	rebuyersPrice := []float64{}
	rebuyersEventCredential := []string{}
	for _, rebuyer := range rebuyersList {
		// Store rebuyer
		rebuyers = append(rebuyers, rebuyer)
		// Store rebuyer number of tickets
		rebuyerNumTickets, err := getRebuyerNumTickets(snap, []byte(rebuyer))
		if err != nil {
			errMessage := fmt.Sprintf("Could not get rebuyer number of tickets: %s", err)
			return nil, nil, nil, nil, xerrors.Errorf(errMessage)
		}
		rebuyersNumTickets = append(rebuyersNumTickets, rebuyerNumTickets)
		// Store rebuyer price
		rebuyerPrice, err := getRebuyerPrice(snap, []byte(rebuyer))
		if err != nil {
			errMessage := fmt.Sprintf("Could not get rebuyer price: %s", err)
			return nil, nil, nil, nil, xerrors.Errorf(errMessage)
		}
		rebuyersPrice = append(rebuyersPrice, rebuyerPrice)
		// Store rebuyer event credential
		rebuyerEventCredential, err := getRebuyerEventCredential(snap, []byte(rebuyer))
		if err != nil {
			errMessage := fmt.Sprintf("Could not get rebuyer event credential: %s", err)
			return nil, nil, nil, nil, xerrors.Errorf(errMessage)
		}
		rebuyersEventCredential = append(rebuyersEventCredential, rebuyerEventCredential)
	}

	return rebuyers, rebuyersNumTickets, rebuyersPrice, rebuyersEventCredential, nil
}

func getResellersNumTicketsPrice(snap store.Snapshot) ([]string, []int, []float64, error) {
	// Get resellers
	resellersList, err := getResellersList(snap)
	if err != nil {
		errMessage := fmt.Sprintf("Could not get resellers list: %s", err)
		return nil, nil, nil, xerrors.Errorf(errMessage)
	}
	// Get resellers, number of tickets, price
	resellers := []string{}
	resellersNumTickets := []int{}
	resellersPrice := []float64{}
	for _, reseller := range resellersList {
		// Store reseller
		resellers = append(resellers, reseller)
		// Store reseller number of tickets
		resellerNumTickets, err := getResellerNumTickets(snap, []byte(reseller))
		if err != nil {
			errMessage := fmt.Sprintf("Could not get reseller number of tickets: %s", err)
			return nil, nil, nil, xerrors.Errorf(errMessage)
		}
		resellersNumTickets = append(resellersNumTickets, resellerNumTickets)
		// Store reseller price
		resellerPrice, err := getResellerPrice(snap, []byte(reseller))
		if err != nil {
			errMessage := fmt.Sprintf("Could not get reseller price: %s", err)
			return nil, nil, nil, xerrors.Errorf(errMessage)
		}
		resellersPrice = append(resellersPrice, resellerPrice)
	}

	return resellers, resellersNumTickets, resellersPrice, nil
}

func outputHandleResalesSuccess(txcount string, eventName string) string {
	return fmt.Sprintf("//EVENTCONTRACT_HANDLERESALESOUTPUT;success;%s;%s//", txcount, eventName)
}

func outputHandleResalesFailure(txcount string, errMessage string) string {
	return fmt.Sprintf("//EVENTCONTRACT_HANDLERESALESOUTPUT;error;%s;%s//", txcount, errMessage)
}

// ############################################################################
// ######################### USE TICKET FUNCTIONS #############################
// ############################################################################

// // useTicket implements commands. It performs the USETICKET command
func (c eventCommand) useTicket(snap store.Snapshot, step execution.Step) error {
	incTxcount(snap)
	txcountByte, err := snap.Get([]byte(TXCountKey))
	txcount := string(txcountByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to increase txcount: %s", err)
		fmt.Fprintln(c.printer, outputUseTicketFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputUseTicketFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Get pk from command
	pk := step.Current.GetArg(UseTicketPKArg)
	if len(pk) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", UseTicketPKArg)
		fmt.Fprintln(c.printer, outputUseTicketFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputUseTicketFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Get numTickets from command
	numTicketsByte := step.Current.GetArg(UseTicketNumTicketsArg)
	if len(numTicketsByte) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", UseTicketNumTicketsArg)
		fmt.Fprintln(c.printer, outputUseTicketFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputUseTicketFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	numTickets, err := byteToInt(numTicketsByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to convert number of tickets to integer: '%s'", err)
		fmt.Fprintln(c.printer, outputUseTicketFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputUseTicketFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	// Get eventCredential from command
	eventCredential := step.Current.GetArg(UseTicketEventCredentialArg)
	if len(eventCredential) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", UseTicketEventCredentialArg)
		fmt.Fprintln(c.printer, outputUseTicketFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputUseTicketFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Reduce numTickets from owner
	err = changeBuyerNumTickets(snap, []byte(pk), -1*numTickets)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to remove tickets from ticket owner: '%s'", err)
		fmt.Fprintln(c.printer, outputUseTicketFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputUseTicketFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// Get number of tickets for owner
	numTicketsLeft, err := getBuyerNumTickets(snap, []byte(pk))
	if err != nil {
		errMessage := fmt.Sprintf("Failed to get number of tickets for owner: '%s'", err)
		fmt.Fprintln(c.printer, outputUseTicketFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputUseTicketFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}

	// If no tickets left, remove owner
	if numTicketsLeft == 0 {
		err = removeOwnerFromOwners(snap, []byte(pk))
		if err != nil {
			errMessage := fmt.Sprintf("Failed to remove user from owners list: '%s'", err)
			fmt.Fprintln(c.printer, outputUseTicketFailure(txcount, errMessage))
			dela.Logger.Info().Str("contract", ContractName).Msgf(outputUseTicketFailure(txcount, errMessage))
			return xerrors.Errorf(errMessage)
		}
	}

	// Get event name
	eventNameByte, err := snap.Get([]byte(NameKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get event name: '%s'", err)
		fmt.Fprintln(c.printer, outputHandleResalesFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputHandleResalesFailure(txcount, errMessage))
		return xerrors.Errorf(errMessage)
	}
	eventName := string(eventNameByte)

	fmt.Fprintln(c.printer, outputUseTicketSuccess(txcount, eventName))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputUseTicketSuccess(txcount, eventName))
	return nil
}

func outputUseTicketSuccess(txcount string, eventName string) string {
	return fmt.Sprintf("//EVENTCONTRACT_USETICKETOUTPUT;success;%s;%s//", txcount, eventName)
}

func outputUseTicketFailure(txcount string, errMessage string) string {
	return fmt.Sprintf("//EVENTCONTRACT_USETICKETOUTPUT;error;%s;%s//", txcount, errMessage)
}

// ############################################################################
// ######################### READ EVENT CONTRACT ##############################
// ############################################################################

// readEventContract implements commands. It performs the READEVENTCONTRACT command
func (c eventCommand) readEventContract(snap store.Snapshot, step execution.Step) (string, error) {
	incTxcount(snap)
	txcountByte, err := snap.Get([]byte(TXCountKey))
	txcount := string(txcountByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to increase txcount: %s", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}

	// Get event name
	eventNameByte, err := snap.Get([]byte(NameKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get event name: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}
	eventName := string(eventNameByte)

	// Get numTickets
	numTicketsByte, err := snap.Get([]byte(NumTicketsKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get event name: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}
	numTickets, err := byteToInt(numTicketsByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to convert number of tickets to integer: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}

	// Get NumTicketsLeft
	numTicketsLeftByte, err := snap.Get([]byte(NumTicketsLeftKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get number of tickets left: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}
	numTicketsLeft, err := byteToInt(numTicketsLeftByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to convert number of tickets left to integer: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}

	// Get price
	priceByte, err := snap.Get([]byte(PriceKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get price of ticket: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}
	price, err := byteToDecimal(priceByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to convert ticket price to float: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}

	// Get max resale price
	maxResalePriceByte, err := snap.Get([]byte(MaxResalePriceKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get max resale price per ticket: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}
	maxResalePrice, err := byteToDecimal(maxResalePriceByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to convert max resale price to float: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}

	// Get resale royalty
	resaleRoyaltyByte, err := snap.Get([]byte(ResaleRoyaltyKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get resale royalty per ticket: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}
	resaleRoyalty, err := byteToDecimal(resaleRoyaltyByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to get resale royalty per ticket: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}

	// Get ticket owners list
	ownersListString, err := getOwnersList(snap)
	if err != nil {
		errMessage := fmt.Sprintf("failed to get owners list: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
	}
	// For each ticket owner, get number of tickets
	ownersList := make([]Owner, 0)
	for _, thisOwner := range ownersListString {
		// Get number of tickets for owner
		thisOwnerNumTickets, err := getBuyerNumTickets(snap, []byte(thisOwner))
		if err != nil {
			errMessage := fmt.Sprintf("failed to get number of tickets for owner: '%s'", err)
			fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
			dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		}
		ownersList = append(ownersList, Owner{thisOwner, strconv.Itoa(thisOwnerNumTickets)})
	}

	// Get ticket resellers list
	resellersListString, err := getResellersList(snap)
	if err != nil {
		errMessage := fmt.Sprintf("failed to get resellers list: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
	}
	// For each ticket reseller, get number of tickets and ticket price
	resellersList := make([]Reseller, 0)
	for _, thisReseller := range resellersListString {
		// Get number of tickets for owner
		thisResellerNumTickets, err := getResellerNumTickets(snap, []byte(thisReseller))
		if err != nil {
			errMessage := fmt.Sprintf("failed to get number of tickets for reseller: '%s'", err)
			fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
			dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		}
		// Get ticket price for reseller
		thisResellerPrice, err := getResellerPrice(snap, []byte(thisReseller))
		if err != nil {
			errMessage := fmt.Sprintf("failed to get number of tickets for reseller: '%s'", err)
			fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
			dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		}
		resellersList = append(resellersList, Reseller{thisReseller, strconv.Itoa(thisResellerNumTickets), fmt.Sprintf("%f", thisResellerPrice)})
	}

	// Get ticket rebuyers list
	rebuyersListString, err := getRebuyersList(snap)
	if err != nil {
		errMessage := fmt.Sprintf("failed to get rebuyers list: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
	}
	// For each ticket reseller, get number of tickets and ticket price
	rebuyersList := make([]Rebuyer, 0)
	for _, thisRebuyer := range rebuyersListString {
		// Get number of tickets for owner
		thisRebuyerNumTickets, err := getRebuyerNumTickets(snap, []byte(thisRebuyer))
		if err != nil {
			errMessage := fmt.Sprintf("failed to get number of tickets for rebuyer: '%s'", err)
			fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
			dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		}
		// Get ticket price for rebuyer
		thisRebuyerPrice, err := getRebuyerPrice(snap, []byte(thisRebuyer))
		if err != nil {
			errMessage := fmt.Sprintf("failed to get number of tickets for rebuyer: '%s'", err)
			fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
			dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		}
		rebuyersList = append(rebuyersList, Rebuyer{thisRebuyer, strconv.Itoa(thisRebuyerNumTickets), fmt.Sprintf("%f", thisRebuyerPrice)})
	}

	// Get event balance
	eventBalanceByte, err := snap.Get([]byte(EventBalanceKey))
	if err != nil {
		errMessage := fmt.Sprintf("failed to get event balance: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}
	eventBalance, err := byteToDecimal(eventBalanceByte)
	if err != nil {
		errMessage := fmt.Sprintf("failed to convert number of tickets left to integer: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		return "", xerrors.Errorf(errMessage)
	}

	// Get user's balances list
	usersListString, err := getUsersList(snap)
	if err != nil {
		errMessage := fmt.Sprintf("failed to get users list: '%s'", err)
		fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
	}
	// For each user, get their bank balance
	usersBalanceList := make([]Balance, 0)
	usersBalanceList = append(usersBalanceList, Balance{eventName, fmt.Sprintf("%f", eventBalance)})
	for _, thisUser := range usersListString {
		// Get bank balance for user
		thisUserBalance, err := getUserBalance(snap, thisUser)
		if err != nil {
			errMessage := fmt.Sprintf("failed to get user balance: '%s'", err)
			fmt.Fprintln(c.printer, outputReadEventContractFailure(txcount, errMessage))
			dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractFailure(txcount, errMessage))
		}
		usersBalanceList = append(usersBalanceList, Balance{thisUser, fmt.Sprintf("%f", thisUserBalance)})
	}

	// Combine data and marshal to json
	eventContractData := EventContractData{
		eventName,
		strconv.Itoa(numTickets),
		strconv.Itoa(numTicketsLeft),
		fmt.Sprintf("%f", price),
		fmt.Sprintf("%f", maxResalePrice),
		fmt.Sprintf("%f", resaleRoyalty),
		ownersList,
		resellersList,
		rebuyersList,
		usersBalanceList,
	}

	output, err := json.Marshal(eventContractData)
	if err != nil {
		return "", xerrors.Errorf("failed to marshal event contract data: %s", err)
	}

	fmt.Fprintln(c.printer, outputReadEventContractSuccess(txcount, eventName, output))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadEventContractSuccess(txcount, eventName, output))
	return string(output), nil
}

type EventContractData struct {
	EventName      string     `json:"event_name"`
	NumTickets     string     `json:"num_tickets"`
	NumTicketsLeft string     `json:"num_tickets_left"`
	Price          string     `json:"price"`
	MaxResalePrice string     `json:"max_resale_price"`
	ResaleRoyalty  string     `json:"resale_royalty"`
	Owners         []Owner    `json:"owners"`
	Resellers      []Reseller `json:"resellers"`
	Rebuyers       []Rebuyer  `json:"rebuyers"`
	UsersBalance   []Balance  `json:"users_balance"`
}

type Owner struct {
	PK         string `json:"pk"`
	NumTickets string `json:"num_tickets"`
}

type Reseller struct {
	PK         string `json:"pk"`
	NumTickets string `json:"num_tickets"`
	Price      string `json:"price"`
}

type Rebuyer struct {
	PK         string `json:"pk"`
	NumTickets string `json:"num_tickets"`
	Price      string `json:"price"`
}

type Balance struct {
	PK      string `json:"pk"`
	Balance string `json:"balance"`
}

func outputReadEventContractSuccess(txcount string, eventName string, contractData []byte) string {
	return fmt.Sprintf("//EVENTCONTRACT_READEVENTOUTPUT;success;%s;%s;%s//", txcount, eventName, contractData)
}

func outputReadEventContractFailure(txcount string, errMessage string) string {
	return fmt.Sprintf("//EVENTCONTRACT_READEVENTOUTPUT;error;%s;%s//", txcount, errMessage)
}

// ########################################################################################
// ########################### GETTER/SETTER FUNCTIONS ###############################
// ########################################################################################

func getUserIndex(snap store.Snapshot, user []byte) (int, error) {
	// Get users list
	users_list, err := getUsersList(snap)
	if err != nil {
		return -1, xerrors.Errorf("failed to get users list: '%s'", err)
	}

	for i, thisUser := range users_list {
		if thisUser == string(user) {
			return i, nil
		}
	}

	return -1, xerrors.Errorf("user not found")
}

// getBuyerNumTickets gets the number of tickets for a buyer
func getBuyerNumTickets(snap store.Snapshot, owner []byte) (int, error) {
	// Get user index of owner
	userIndex, err := getUserIndex(snap, owner)
	if err != nil {
		return -1, xerrors.Errorf("failed to get user index: '%s'", err)
	}

	// Get tickets for owner
	thisTicketsKey := fmt.Sprintf("%s:%s", BuyerTicketsKey, fmt.Sprint(userIndex))
	tickets, err := snap.Get([]byte(thisTicketsKey))
	if err != nil {
		return -1, xerrors.Errorf("failed to get tickets for owner: '%s'", err)
	}
	if len(tickets) == 0 {
		return 0, nil
	}
	// If owner has tickets, increment number of tickets
	ticketsInt, err := byteToInt(tickets)
	if err != nil {
		return -1, xerrors.Errorf("failed to convert tickets to integer: '%s'", err)
	}

	return ticketsInt, nil
}

// changes number of tickets left on event smart contract
func changeNumTicketsLeft(snap store.Snapshot, diff int) error {
	numTicketsLeftByte, err := snap.Get([]byte(NumTicketsLeftKey))
	if err != nil {
		return xerrors.Errorf("failed to get number of tickets left: '%s'", err)
	}

	numTicketsLeft, err := byteToInt(numTicketsLeftByte)
	if err != nil {
		return xerrors.Errorf("failed to convert number of tickets left to integer: '%s'", err)
	}

	newNumTicketsLeft := numTicketsLeft + diff
	snap.Set([]byte(NumTicketsLeftKey), []byte(strconv.Itoa(newNumTicketsLeft)))

	return nil
}

// Checks the number of tickets a given pk own
func hasNumTickets(snap store.Snapshot, pk []byte) (int, error) {
	// Get index of pk
	userIndex, err := getUserIndex(snap, pk)
	if err != nil {
		return -1, xerrors.Errorf("failed to get user index: '%s'", err)
	}

	// Get number of tickets for pk
	thisTicketsKey := fmt.Sprintf("%s:%s", BuyerTicketsKey, fmt.Sprint(userIndex))
	numTicketsByte, err := snap.Get([]byte(thisTicketsKey))
	if err != nil {
		return -1, xerrors.Errorf("failed to get number of tickets for pk: '%s'", err)
	}

	// If no tickets, return 0
	if len(numTicketsByte) == 0 {
		return 0, nil
	}

	// Convert number of tickets to decimal
	numTickets, err := byteToInt(numTicketsByte)
	if err != nil {
		return -1, xerrors.Errorf("failed to convert number of tickets to integer: '%s'", err)
	}

	return numTickets, nil
}

// changeBuyerNumTickets updates the number of tickets for a buyer
func changeBuyerNumTickets(snap store.Snapshot, owner []byte, numTickets int) error {
	// Get user index of owner
	userIndex, err := getUserIndex(snap, owner)
	if err != nil {
		return xerrors.Errorf("failed to get user index: '%s'", err)
	}

	// Get tickets for owner
	thisTicketsKey := fmt.Sprintf("%s:%s", BuyerTicketsKey, fmt.Sprint(userIndex))
	tickets, err := snap.Get([]byte(thisTicketsKey))
	if err != nil {
		return xerrors.Errorf("failed to get tickets for owner: '%s'", err)
	}

	if len(tickets) == 0 {
		// If no tickets for owner, set number of tickets
		err = snap.Set([]byte(thisTicketsKey), []byte(strconv.Itoa(numTickets)))
		if err != nil {
			return xerrors.Errorf("failed to set tickets for owner: '%s'", err)
		}
	} else {
		// If owner has tickets, increment number of tickets
		ticketsInt, err := byteToInt(tickets)
		if err != nil {
			return xerrors.Errorf("failed to convert tickets to integer: '%s'", err)
		}
		newTicketsInt := ticketsInt + numTickets
		err = snap.Set([]byte(thisTicketsKey), []byte(strconv.Itoa(newTicketsInt)))
		if err != nil {
			return xerrors.Errorf("failed to set tickets for owner: '%s'", err)
		}
	}

	return nil
}

// storeTickets stores tickets for ticket owner
func changeResellerNumTickets(snap store.Snapshot, owner []byte, numTickets int) error {
	// Get user index of owner
	userIndex, err := getUserIndex(snap, owner)
	if err != nil {
		return xerrors.Errorf("failed to get user index: '%s'", err)
	}

	// Get tickets for owner
	thisTicketsKey := fmt.Sprintf("%s:%s", ResellerTicketsNumberKey, fmt.Sprint(userIndex))
	tickets, err := snap.Get([]byte(thisTicketsKey))
	if err != nil {
		return xerrors.Errorf("failed to get tickets for owner: '%s'", err)
	}

	if len(tickets) == 0 {
		// If no tickets for owner, set number of tickets
		err = snap.Set([]byte(thisTicketsKey), []byte(strconv.Itoa(numTickets)))
		if err != nil {
			return xerrors.Errorf("failed to set tickets for owner: '%s'", err)
		}
	} else {
		// If owner has tickets, increment number of tickets
		ticketsInt, err := byteToInt(tickets)
		if err != nil {
			return xerrors.Errorf("failed to convert tickets to integer: '%s'", err)
		}
		newTicketsInt := ticketsInt + numTickets
		err = snap.Set([]byte(thisTicketsKey), []byte(strconv.Itoa(newTicketsInt)))
		if err != nil {
			return xerrors.Errorf("failed to set tickets for owner: '%s'", err)
		}
	}

	return nil
}

// Gets number of tickets being sold for reseller
func getResellerNumTickets(snap store.Snapshot, reseller []byte) (int, error) {
	// Get user index of reseller
	userIndex, err := getUserIndex(snap, reseller)
	if err != nil {
		return -1, xerrors.Errorf("failed to get user index: %s", err)
	}

	// Get number of tickets for reseller
	resellerTicketsNumberKey := fmt.Sprintf("%s:%s", ResellerTicketsNumberKey, fmt.Sprint(userIndex))
	resellerTicketsNumberByte, err := snap.Get([]byte(resellerTicketsNumberKey))
	if err != nil {
		return -1, xerrors.Errorf("failed to get reseller tickets for reseller: %s", err)
	}
	resellerTicketsNumber := 0
	if len(resellerTicketsNumberByte) != 0 {
		resellerTicketsNumber, err = byteToInt(resellerTicketsNumberByte)
		if err != nil {
			return -1, xerrors.Errorf("failed to convert number of reseller tickets to integer: %s", err)
		}
	}

	return resellerTicketsNumber, nil
}

// getResellerPrice gets the price per ticket for the reseller
func getResellerPrice(snap store.Snapshot, reseller []byte) (float64, error) {
	// Get user index of reseller
	userIndex, err := getUserIndex(snap, reseller)
	if err != nil {
		return -1, xerrors.Errorf("failed to get reseller index: %s", err)
	}

	// Get number of tickets for reseller
	resellerTicketsPriceKey := fmt.Sprintf("%s:%s", ResellerTicketsPriceKey, fmt.Sprint(userIndex))
	resellerTicketsPriceByte, err := snap.Get([]byte(resellerTicketsPriceKey))
	if err != nil {
		return -1, xerrors.Errorf("failed to get reseller price for reseller: %s", err)
	}
	resellerTicketsPrice := float64(0)
	if len(resellerTicketsPriceByte) != 0 {
		resellerTicketsPrice, err = byteToDecimal(resellerTicketsPriceByte)
		if err != nil {
			return -1, xerrors.Errorf("failed to convert price of reseller tickets to integer: %s", err)
		}
	}

	return resellerTicketsPrice, nil
}

// storeResellerPrice stores the price per ticket for the reseller
func storeResellerPrice(snap store.Snapshot, reseller []byte, price float64) error {
	// Get user index of reseller
	userIndex, err := getUserIndex(snap, reseller)
	if err != nil {
		return xerrors.Errorf("failed to get reseller index: %s", err)
	}

	// Get number of resale tickets for owner
	resellerTicketsPriceKey := fmt.Sprintf("%s:%s", ResellerTicketsPriceKey, fmt.Sprint(userIndex))
	resellerTicketsPriceByte, err := snap.Get([]byte(resellerTicketsPriceKey))
	if err != nil {
		return xerrors.Errorf("failed to get reseller tickets for reseller: %s", err)
	}

	resellerTicketsPrice := float64(0)
	if len(resellerTicketsPriceByte) != 0 {
		resellerTicketsPrice, err = byteToDecimal(resellerTicketsPriceByte)
		if err != nil {
			return xerrors.Errorf("failed to convert number of reseller tickets to integer: %s", err)
		}
	}
	newResellerTicketsPrice := resellerTicketsPrice + price
	err = snap.Set([]byte(resellerTicketsPriceKey), []byte(fmt.Sprintf("%f", newResellerTicketsPrice)))
	if err != nil {
		return xerrors.Errorf("failed to set price of reseller tickets for reseller: %s", err)
	}

	return nil
}

// Gets number of tickets being bought for rebuyer
func getRebuyerNumTickets(snap store.Snapshot, rebuyer []byte) (int, error) {
	// Get user index of rebuyer
	userIndex, err := getUserIndex(snap, rebuyer)
	if err != nil {
		return -1, xerrors.Errorf("failed to get user index: %s", err)
	}

	// Get number of tickets for rebuyer
	rebuyerTicketsNumberKey := fmt.Sprintf("%s:%s", RebuyerTicketsNumberKey, fmt.Sprint(userIndex))
	rebuyerTicketsNumberByte, err := snap.Get([]byte(rebuyerTicketsNumberKey))
	if err != nil {
		return -1, xerrors.Errorf("failed to get rebuyer tickets for rebuyer: %s", err)
	}
	rebuyerTicketsNumber := 0
	if len(rebuyerTicketsNumberByte) != 0 {
		rebuyerTicketsNumber, err = byteToInt(rebuyerTicketsNumberByte)
		if err != nil {
			return -1, xerrors.Errorf("failed to convert number of rebuyer tickets to integer: %s", err)
		}
	}

	return rebuyerTicketsNumber, nil
}

// getrebuyerPrice gets the price per ticket for the rebuyer
func getRebuyerPrice(snap store.Snapshot, rebuyer []byte) (float64, error) {
	// Get user index of rebuyer
	userIndex, err := getUserIndex(snap, rebuyer)
	if err != nil {
		return -1, xerrors.Errorf("failed to get rebuyer index: %s", err)
	}

	// Get number of tickets for rebuyer
	rebuyerTicketsPriceKey := fmt.Sprintf("%s:%s", RebuyerTicketsPriceKey, fmt.Sprint(userIndex))
	rebuyerTicketsPriceByte, err := snap.Get([]byte(rebuyerTicketsPriceKey))
	if err != nil {
		return -1, xerrors.Errorf("failed to get rebuyer price for rebuyer: %s", err)
	}
	rebuyerTicketsPrice := float64(0)
	if len(rebuyerTicketsPriceByte) != 0 {
		rebuyerTicketsPrice, err = byteToDecimal(rebuyerTicketsPriceByte)
		if err != nil {
			return -1, xerrors.Errorf("failed to convert price of rebuyer tickets to integer: %s", err)
		}
	}

	return rebuyerTicketsPrice, nil
}

// storeRebuyerPrice stores the price per ticket for the rebuyer
func storeRebuyerPrice(snap store.Snapshot, rebuyer []byte, price float64) error {
	// Get user index of rebuyer
	userIndex, err := getUserIndex(snap, rebuyer)
	if err != nil {
		return xerrors.Errorf("failed to get rebuyer index: %s", err)
	}

	// Get price of rebuy tickets for owner
	rebuyerTicketsPriceKey := fmt.Sprintf("%s:%s", RebuyerTicketsPriceKey, fmt.Sprint(userIndex))
	err = snap.Set([]byte(rebuyerTicketsPriceKey), []byte(fmt.Sprintf("%f", price)))
	if err != nil {
		return xerrors.Errorf("failed to set price of rebuyer tickets for rebuyer: %s", err)
	}

	return nil
}

// changeRebuyerNumTickets stores number of rebuy tickets for ticket owner
func changeRebuyerNumTickets(snap store.Snapshot, owner []byte, numTickets int) error {
	// Get user index of owner
	userIndex, err := getUserIndex(snap, owner)
	if err != nil {
		return xerrors.Errorf("failed to get user index: '%s'", err)
	}

	// Get tickets for owner
	thisTicketsKey := fmt.Sprintf("%s:%s", RebuyerTicketsNumberKey, fmt.Sprint(userIndex))
	tickets, err := snap.Get([]byte(thisTicketsKey))
	if err != nil {
		return xerrors.Errorf("failed to get tickets for owner: '%s'", err)
	}

	if len(tickets) == 0 {
		// If no tickets for owner, set number of tickets
		err = snap.Set([]byte(thisTicketsKey), []byte(strconv.Itoa(numTickets)))
		if err != nil {
			return xerrors.Errorf("failed to set tickets for owner: '%s'", err)
		}
	} else {
		// If owner has tickets, increment number of tickets
		ticketsInt, err := byteToInt(tickets)
		if err != nil {
			return xerrors.Errorf("failed to convert tickets to integer: '%s'", err)
		}
		newTicketsInt := ticketsInt + numTickets
		err = snap.Set([]byte(thisTicketsKey), []byte(strconv.Itoa(newTicketsInt)))
		if err != nil {
			return xerrors.Errorf("failed to set tickets for owner: '%s'", err)
		}
	}

	return nil
}

// getBuyerEventCredential gets the event credential for the buyer
func getBuyerEventCredential(snap store.Snapshot, buyer []byte) (string, error) {
	// Get user index of buyer
	userIndex, err := getUserIndex(snap, buyer)
	if err != nil {
		return "", xerrors.Errorf("failed to get rebuyer index: %s", err)
	}

	// Get eventCredential for rebuyer
	buyerEventCredentialKey := fmt.Sprintf("%s:%s", BuyerEventCredentialKey, fmt.Sprint(userIndex))
	buyerEventCredentialByte, err := snap.Get([]byte(buyerEventCredentialKey))
	if err != nil {
		return "", xerrors.Errorf("failed to get buyer eventCredential for buyer: %s", err)
	}
	buyerEventCredential := string(buyerEventCredentialByte)

	return buyerEventCredential, nil
}

// storeBuyerEventCredential stores the buyer's eventCredential
func storeBuyerEventCredential(snap store.Snapshot, owner []byte, eventCredential []byte) error {
	// Get user index of owner
	userIndex, err := getUserIndex(snap, owner)
	if err != nil {
		return xerrors.Errorf("failed to get user index: '%s'", err)
	}

	// Set event credential for owner
	thisEventCredentialKey := fmt.Sprintf("%s:%s", BuyerEventCredentialKey, fmt.Sprint(userIndex))
	err = snap.Set([]byte(thisEventCredentialKey), eventCredential)
	if err != nil {
		return xerrors.Errorf("failed to set event credential for owner: '%s'", err)
	}

	return nil
}

// getrebuyerEventCredential gets the event credential for the rebuyer
func getRebuyerEventCredential(snap store.Snapshot, rebuyer []byte) (string, error) {
	// Get user index of rebuyer
	userIndex, err := getUserIndex(snap, rebuyer)
	if err != nil {
		return "", xerrors.Errorf("failed to get rebuyer index: %s", err)
	}

	// Get eventCredential for rebuyer
	rebuyerEventCredentialKey := fmt.Sprintf("%s:%s", RebuyerEventCredentialKey, fmt.Sprint(userIndex))
	rebuyerEventCredentialByte, err := snap.Get([]byte(rebuyerEventCredentialKey))
	if err != nil {
		return "", xerrors.Errorf("failed to get rebuyer eventCredential for rebuyer: %s", err)
	}
	rebuyerEventCredential := string(rebuyerEventCredentialByte)

	return rebuyerEventCredential, nil
}

// storeRebuyerEventCredential stores the rebuyer's eventCredential
func storeRebuyerEventCredential(snap store.Snapshot, owner []byte, eventCredential []byte) error {
	// Get user index of owner
	userIndex, err := getUserIndex(snap, owner)
	if err != nil {
		return xerrors.Errorf("failed to get user index: '%s'", err)
	}

	// Set event credential for owner
	thisEventCredentialKey := fmt.Sprintf("%s:%s", RebuyerEventCredentialKey, fmt.Sprint(userIndex))
	err = snap.Set([]byte(thisEventCredentialKey), eventCredential)
	if err != nil {
		return xerrors.Errorf("failed to set event credential for rebuyer: '%s'", err)
	}

	return nil
}

// ########################################################################################
// ########################### GETTER/SETTER LIST FUNCTIONS ###############################
// ########################################################################################

// getUsersList gets list of users in string format
func getUsersList(snap store.Snapshot) ([]string, error) {
	// Get owners list
	key := []byte(UserIndexKey)
	users_list, err := snap.Get(key)
	if err != nil {
		return []string{}, xerrors.Errorf("failed to get users list: '%s'", err)
	}

	// Format
	users := strings.Split(string(users_list), ListDelimiter)
	users = users[:len(users)-1]
	return users, nil
}

// storeTicketUser stores the user in order to obtain their index
func storeTicketUser(snap store.Snapshot, newUser []byte) error {
	// Get users list
	users_list, err := snap.Get([]byte(UserIndexKey))
	if err != nil {
		return xerrors.Errorf("failed to get ticket users list: '%s'", err)
	}

	// Check that user not already in user index list
	if bytes.Contains(users_list, newUser) {
		return nil
	}

	// If newUser not already in users list, add newUser to users list
	users_list = []byte(fmt.Sprintf("%s%s;", string(users_list), string(newUser)))
	err = snap.Set([]byte(UserIndexKey), users_list)
	if err != nil {
		return xerrors.Errorf("failed to set ticket owners list: '%s'", err)
	}

	return nil
}

// getOwnersList gets list of owners in string format
func getOwnersList(snap store.Snapshot) ([]string, error) {
	// Get owners list
	key := []byte(TicketOwnersKey)
	owners_list, err := snap.Get(key)
	if err != nil {
		return []string{}, xerrors.Errorf("failed to get owners list: '%s'", err)
	}

	// Format
	owners := strings.Split(string(owners_list), ListDelimiter)
	owners = owners[:len(owners)-1]
	return owners, nil
}

// storeTicketOwner stores a ticket owner public key in a ticket owners list
func storeTicketOwner(snap store.Snapshot, owner []byte) error {
	// Get ticket owners list
	owners_list, err := snap.Get([]byte(TicketOwnersKey))
	if err != nil {
		return xerrors.Errorf("failed to get ticket owners list: '%s'", err)
	}

	// Check that ticket owner not already in owners list
	if bytes.Contains(owners_list, owner) {
		return nil
	}

	// If owner not already in owners list, add ticket owner to ticket owners list
	owners_list = []byte(fmt.Sprintf("%s%s;", string(owners_list), string(owner)))
	err = snap.Set([]byte(TicketOwnersKey), owners_list)
	if err != nil {
		return xerrors.Errorf("failed to set ticket owners list: '%s'", err)
	}

	return nil
}

// storeTicketOwner stores a ticket owner public key in a ticket owners list
func removeOwnerFromOwners(snap store.Snapshot, ownerByte []byte) error {
	// Get ticket owners list
	oldOwnersList, err := getOwnersList(snap)
	if err != nil {
		return xerrors.Errorf("failed to get ticket owners list: %s", err)
	}

	owner := string(ownerByte)

	// Check if owner is in ticket owners list, remove
	newOwnersList := ""
	for _, thisOwner := range oldOwnersList {
		if owner != thisOwner {
			newOwnersList = newOwnersList + thisOwner + ListDelimiter
		}
	}

	// Store new list of ticket owners
	err = snap.Set([]byte(TicketOwnersKey), []byte(newOwnersList))
	if err != nil {
		return xerrors.Errorf("failed to set ticket owners list: %s", err)
	}

	return nil
}

// getOwnersList gets list of owners in string format
func getResellersList(snap store.Snapshot) ([]string, error) {
	// Get resellers list
	key := []byte(TicketResellersKey)
	resellers_list, err := snap.Get(key)
	if err != nil {
		return []string{}, xerrors.Errorf("failed to get resellers list: %s", err)
	}

	// Format
	resellers := strings.Split(string(resellers_list), ListDelimiter)
	resellers = resellers[:len(resellers)-1]
	return resellers, nil
}

// storeTicketReseller stores a ticket reseller public key in a ticket resellers list
func storeTicketReseller(snap store.Snapshot, reseller []byte) error {
	// Get ticket resellers list
	resellers_list, err := snap.Get([]byte(TicketResellersKey))
	if err != nil {
		return xerrors.Errorf("failed to get ticket resellers list: %s", err)
	}

	// Add reseller to resellers list
	if !bytes.Contains(resellers_list, reseller) {
		// Add reseller to resellers list if not already in resellers list
		resellers_list = []byte(fmt.Sprintf("%s%s;", string(resellers_list), string(reseller)))
	}
	err = snap.Set([]byte(TicketResellersKey), resellers_list)
	if err != nil {
		return xerrors.Errorf("failed to set ticket resellers list: %s", err)
	}

	return nil
}

// removeOwnerFromResellers removes a ticket owner from this ticket resellers list
func removeOwnerFromResellers(snap store.Snapshot, resellerByte []byte) error {
	// Get ticket owners list
	oldResellersList, err := getResellersList(snap)
	if err != nil {
		return xerrors.Errorf("failed to get ticket owners list: %s", err)
	}

	reseller := string(resellerByte)

	// Check if reseller is in ticket resellers list, remove
	newResellersList := ""
	for _, thisReseller := range oldResellersList {
		if reseller != thisReseller {
			newResellersList = newResellersList + thisReseller + ListDelimiter
		}
	}

	// Store new list of ticket owners
	err = snap.Set([]byte(TicketResellersKey), []byte(newResellersList))
	if err != nil {
		return xerrors.Errorf("failed to set ticket owners list: %s", err)
	}

	return nil
}

// getRebuyersList gets list of rebuyers in string format
func getRebuyersList(snap store.Snapshot) ([]string, error) {
	// Get rebuyers list
	key := []byte(TicketRebuyersKey)
	rebuyers_list, err := snap.Get(key)
	if err != nil {
		return []string{}, xerrors.Errorf("failed to get resellers list: %s", err)
	}

	// Format
	rebuyers := strings.Split(string(rebuyers_list), ListDelimiter)
	rebuyers = rebuyers[:len(rebuyers)-1]
	return rebuyers, nil
}

// storeTicketRebuyer stores a ticket rebuyer public key in a ticket rebuyers list
func storeTicketRebuyer(snap store.Snapshot, rebuyer []byte) error {
	// Get ticket rebuyers list
	rebuyers_list, err := snap.Get([]byte(TicketRebuyersKey))
	if err != nil {
		return xerrors.Errorf("failed to get ticket rebuyers list: %s", err)
	}

	// Add rebuyer to rebuyers list
	if !bytes.Contains(rebuyers_list, rebuyer) {
		// Add rebuyer to rebuyers list if not already in rebuyers list
		rebuyers_list = []byte(fmt.Sprintf("%s%s;", string(rebuyers_list), string(rebuyer)))
	}
	err = snap.Set([]byte(TicketRebuyersKey), rebuyers_list)
	if err != nil {
		return xerrors.Errorf("failed to set ticket rebuyers list: %s", err)
	}

	return nil
}

// removeOwnerFromReuyersb removes a ticket owner from this ticket rebuyers list
func removeOwnerFromRebuyers(snap store.Snapshot, rebuyerByte []byte) error {
	// Get ticket rebuyers list
	oldRebuyersList, err := getRebuyersList(snap)
	if err != nil {
		return xerrors.Errorf("failed to get ticket owners list: %s", err)
	}

	rebuyer := string(rebuyerByte)

	// Check if rebuyer is in ticket rebuyers list, remove
	newRebuyersList := ""
	for _, thisRebuyer := range oldRebuyersList {
		if rebuyer != thisRebuyer {
			newRebuyersList = newRebuyersList + thisRebuyer + ListDelimiter
		}
	}

	// Store new list of ticket rebuyers
	err = snap.Set([]byte(TicketRebuyersKey), []byte(newRebuyersList))
	if err != nil {
		return xerrors.Errorf("failed to set ticket rebuyers list: %s", err)
	}

	return nil
}

// ############################################################################
// ############################ BANK FUNCTIONS ################################
// ############################################################################

// transferPayment stores the bidder's payment in event:account
func transferPaymentToEvent(snap store.Snapshot, ticketPayment float64) error {
	// Get event balance
	eventBalanceByte, err := snap.Get([]byte(EventBalanceKey))
	if err != nil {
		return xerrors.Errorf("failed to get event balance: '%s'", err)
	}
	eventBalance, err := byteToDecimal(eventBalanceByte)
	if err != nil {
		return xerrors.Errorf("failed to convert event balance to integer: '%s'", err)
	}

	// Add payment to event balance
	eventBalance = eventBalance + ticketPayment

	// Store event balance
	err = snap.Set([]byte(EventBalanceKey), []byte(fmt.Sprintf("%f", eventBalance)))
	if err != nil {
		return xerrors.Errorf("failed to set event balance: %v", err)
	}

	return nil
}

// transferPayment transfers payment from buyer to reseller/event organizer based on resale royalty
// stores payments in event:account
func transferSalePayment(snap store.Snapshot, buyer string, ticketPayment float64) error {
	// Transfer to event
	err := transferPaymentToEvent(snap, ticketPayment)
	if err != nil {
		return xerrors.Errorf("failed to transfer payment to event: '%s'", err)
	}
	// Transfer from buyer
	err = changeUserBalance(snap, buyer, -ticketPayment)
	if err != nil {
		return xerrors.Errorf("failed to transfer payment to reseller: '%s'", err)
	}

	return nil
}

// transferPayment transfers payment from buyer to reseller/event organizer based on resale royalty
// stores payments in event:account
func transferResalePayment(snap store.Snapshot, rebuyer string, reseller string, ticketPayment float64) error {
	// Get resale royalty
	resaleRoyaltyByte, err := snap.Get([]byte(ResaleRoyaltyKey))
	if err != nil {
		return xerrors.Errorf("failed to get resale royalty: '%s'", err)
	}
	resaleRoyalty, err := byteToDecimal(resaleRoyaltyByte)
	if err != nil {
		return xerrors.Errorf("failed to convert resale royalty to float64: '%s'", err)
	}

	paymentToReseller := ticketPayment * ((100 - resaleRoyalty) / 100)
	paymentToEvent := ticketPayment * (resaleRoyalty / 100)

	// Transfer to event
	err = transferPaymentToEvent(snap, paymentToEvent)
	if err != nil {
		return xerrors.Errorf("failed to transfer payment to event: '%s'", err)
	}
	// Transfer to reseller
	err = changeUserBalance(snap, reseller, paymentToReseller)
	if err != nil {
		return xerrors.Errorf("failed to transfer payment to reseller: '%s'", err)
	}
	// Remove from buyer
	err = changeUserBalance(snap, rebuyer, -ticketPayment)
	if err != nil {
		return xerrors.Errorf("failed to transfer payment from buyer: '%s'", err)
	}

	return nil
}

// getUserBalance gets the user's balance from the bank
func getUserBalance(snap store.Snapshot, pk string) (float64, error) {
	userIndex, err := getUserIndex(snap, []byte(pk))
	if err != nil {
		return -1, xerrors.Errorf("failed to get user balance: %s", err)
	}

	// Get user balance
	userBalanceKey := fmt.Sprintf("%s:account:%s", BankKey, fmt.Sprint(userIndex))
	userBalanceByte, err := snap.Get([]byte(userBalanceKey))
	if err != nil {
		return -1, xerrors.Errorf("failed to get user balance: '%s'", err)
	}
	if len(userBalanceByte) == 0 {
		err = snap.Set([]byte(userBalanceKey), []byte("0.000000"))
		if err != nil {
			return -1, xerrors.Errorf("failed to set user balance: '%s'", err)
		}
		return float64(0), nil
	}

	// Convert user balance to float64
	userBalance, err := byteToDecimal(userBalanceByte)
	if err != nil {
		return -1, xerrors.Errorf("failed to convert user balance to float64: '%s'", err)
	}

	return userBalance, nil
}

// changeUserBalance sets the user's balance from the bank
func changeUserBalance(snap store.Snapshot, pk string, amount float64) error {
	userIndex, err := getUserIndex(snap, []byte(pk))
	if err != nil {
		return xerrors.Errorf("failed to get user balance: %s", err)
	}

	// Get user balance
	userBalanceKey := fmt.Sprintf("%s:account:%s", BankKey, fmt.Sprint(userIndex))
	userBalanceByte, err := snap.Get([]byte(userBalanceKey))
	if err != nil {
		return xerrors.Errorf("failed to get user balance: '%s'", err)
	}
	if len(userBalanceByte) == 0 {
		userBalanceByte = []byte("0.000000")
	}

	// Convert user balance to float64
	userBalance, err := byteToDecimal(userBalanceByte)
	if err != nil {
		return xerrors.Errorf("failed to convert user balance to float64: '%s'", err)
	}

	// Add amount to user balance
	newUserBalance := userBalance + amount

	// Set new user balance
	err = snap.Set([]byte(userBalanceKey), []byte(fmt.Sprintf("%f", newUserBalance)))
	if err != nil {
		return xerrors.Errorf("failed to set user balance: %v", err)
	}

	return nil
}

// ############################################################################
// ########################## HELPER FUNCTIONS ################################
// ############################################################################

// Increments transaction count for event smart contract
func incTxcount(snap store.Snapshot) error {
	txcountByte, err := snap.Get([]byte(TXCountKey))
	if err != nil {
		return xerrors.Errorf("failed to get txcount: '%s'", err)
	}

	txcount, err := byteToInt(txcountByte)
	if err != nil {
		return xerrors.Errorf("failed to convert txcount to integer: '%s'", err)
	}

	txcount = txcount + 1

	err = snap.Set([]byte(TXCountKey), []byte(strconv.Itoa(txcount)))
	if err != nil {
		return xerrors.Errorf("failed to set txcount: '%s'", err)
	}

	return nil
}

// Checks if contract is in bidding period
func isTicketsLeft(snap store.Snapshot, numTickets int) (bool, error) {
	// Get number of tickets left from database
	key := []byte(NumTicketsLeftKey)
	numTicketsLeftByte, err := snap.Get(key)
	if err != nil {
		return false, xerrors.Errorf("failed to get block_number: %v", err)
	}
	// Convert number of tickets left to float64
	numTicketsLeft, err := byteToInt(numTicketsLeftByte)
	if err != nil {
		return false, xerrors.Errorf("failed to convert number of tickets left to float64: %v.", err)
	}

	isTicketsLeft := numTicketsLeft-numTickets >= 0

	return isTicketsLeft, nil
}

// Converts a byte array to an integer
// Returns error if cannot be converted
func byteToInt(b []byte) (int, error) {
	byteInt, err := strconv.Atoi(string(b))
	if err != nil {
		return -1, xerrors.Errorf("Failed to convert Byte Array to int: '%s'", err)
	}

	return byteInt, nil
}

func byteToDecimal(b []byte) (float64, error) {
	byteDecimal, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return float64(-1), xerrors.Errorf("Failed to convert Byte Array to decimal: '%s'", err)
	}

	return byteDecimal, nil
}

// Checks if pk is the auction owner
func isEventOwner(snap store.Snapshot, pk []byte) (bool, error) {
	// Get event owner
	owner, err := snap.Get([]byte(OwnerKey))
	if err != nil {
		return false, xerrors.Errorf("owner not found in store")
	}

	// Check if pk matches event owner
	if string(owner) == string(pk) {
		return true, nil
	} else {
		return false, nil
	}
}

// infoLog defines an output using zerolog
//
// - implements io.writer
type infoLog struct{}

func (h infoLog) Write(p []byte) (int, error) {
	dela.Logger.Info().Msg(string(p))

	return len(p), nil
}
