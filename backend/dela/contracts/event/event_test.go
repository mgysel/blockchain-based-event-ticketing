package event

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
	err := contract.Execute(fakeStore{}, makeStep(t, signer, CmdArg, "BUY"))
	require.NoError(t, err)
}

func TestCommand_Init(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()
	pkByte, _ := signer.GetPublicKey().MarshalText()

	cmd := eventCommand{
		Contract: &contract,
	}

	pk := string(pkByte)
	name := "Event"
	num_tickets := "100"
	price := "50"
	max_resale_price := "50"
	resale_royalty := "10"

	// Check error when do not have all arguments
	err := cmd.init(fake.NewBadSnapshot(), makeStep(t, signer, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty))
	require.EqualError(t, err, fake.Err("failed to set txcount to 1"))
	// missing pk arg
	err = cmd.init(fake.NewSnapshot(), makeStep(t, signer, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty))
	require.EqualError(t, err, "'value:initPK' not found in tx arg")
	// missing name
	err = cmd.init(fake.NewSnapshot(), makeStep(t, signer, InitPKArg, pk, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty))
	require.EqualError(t, err, "'value:initName' not found in tx arg")
	// missing num_tickets
	err = cmd.init(fake.NewSnapshot(), makeStep(t, signer, InitPKArg, pk, InitNameArg, name, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty))
	require.EqualError(t, err, "'value:initNumTickets' not found in tx arg")
	// missing price
	err = cmd.init(fake.NewSnapshot(), makeStep(t, signer, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty))
	require.EqualError(t, err, "'value:initPrice' not found in tx arg")
	// missing resale price
	err = cmd.init(fake.NewSnapshot(), makeStep(t, signer, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitResaleRoyaltyArg, resale_royalty))
	require.EqualError(t, err, "'value:initMaxResalePrice' not found in tx arg")
	// missing resale royalty
	err = cmd.init(fake.NewSnapshot(), makeStep(t, signer, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price))
	require.EqualError(t, err, "'value:initResaleRoyalty' not found in tx arg")

	// Correct init
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err = cmd.init(snap, step)
	require.NoError(t, err)

	// Check store for (event:owner)
	key := []byte("event:owner")
	val, _ := snap.Get(key)
	val_res := string(val)
	require.Equal(t, pk, val_res)

	// Check store for (event:name)
	key = []byte("event:name")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, name, val_res)

	// Check store for (event:num_tickets)
	key = []byte("event:num_tickets")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, num_tickets, val_res)

	// Check store for (event:price)
	key = []byte("event:price")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, price, val_res)

	// Check store for (event:max_resale_price)
	key = []byte("event:max_resale_price")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, max_resale_price, val_res)

	// Check store for (event:resale_royalty)
	key = []byte("event:resale_royalty")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, resale_royalty, val_res)
}

func TestCommand_Buy(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()
	pkByte, _ := signer.GetPublicKey().MarshalText()

	cmd := eventCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	pk1, name, num_tickets, price, max_resale_price, resale_royalty := string(pkByte), "Event", "100", "100", "100", "10"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitPKArg, pk1, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err := cmd.init(snap, step)
	require.NoError(t, err)

	// BUY 1 TICKET
	// Buy ticket with not enough funds
	buyPayment := "99"
	numTickets := "1"
	eventCredential := "eventCredential"
	step = makeStep(t, signer, BuyPKArg, pk1, BuyNumTicketsArg, numTickets, BuyPaymentArg, buyPayment, BuyEventCredentialArg, eventCredential)
	err = cmd.buy(snap, step)
	require.EqualError(t, err, "Payment must be at least the price of each ticket")

	// Buy ticket
	buyPayment = "100"
	numTickets = "1"
	eventCredential = "eventCredential"
	step = makeStep(t, signer, BuyPKArg, pk1, BuyNumTicketsArg, numTickets, BuyPaymentArg, buyPayment, BuyEventCredentialArg, eventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// Check that signer is now a ticket owner
	key := []byte("event:ticket_owners")
	val, _ := snap.Get(key)
	val_res := strings.Trim(string(val), ";")
	require.Equal(t, pk1, val_res)

	// Check that signer is now a ticket user
	key = []byte("event:users")
	val, _ = snap.Get(key)
	val_res = strings.Trim(string(val), ";")
	require.Equal(t, pk1, val_res)

	// Check that signer now has a ticket in their account
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", fmt.Sprint(0)))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, numTickets, val_res)

	// Check that signer now has event credential
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", fmt.Sprint(0)))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, eventCredential, val_res)

	// BUY 2 TICKETS
	// Buy 2 tickets with not enough funds
	buyPayment = "99"
	numTickets = "2"
	step = makeStep(t, signer, BuyPKArg, pk1, BuyNumTicketsArg, numTickets, BuyPaymentArg, buyPayment, BuyEventCredentialArg, eventCredential)
	err = cmd.buy(snap, step)
	require.EqualError(t, err, "Payment must be at least the price of each ticket")

	// Buy a second ticket
	buyPayment = "200"
	numTickets = "2"
	step = makeStep(t, signer, BuyPKArg, pk1, BuyNumTicketsArg, numTickets, BuyPaymentArg, buyPayment, BuyEventCredentialArg, eventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// Check that signer is still a ticket owner
	key = []byte("event:ticket_owners")
	val, _ = snap.Get(key)
	val_res = strings.Trim(string(val), ";")
	require.Equal(t, pk1, val_res)

	// Check that signer is still a user
	key = []byte("event:users")
	val, _ = snap.Get(key)
	val_res = strings.Trim(string(val), ";")
	require.Equal(t, pk1, val_res)

	// Check that signer now has 2 tickets in their account
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", fmt.Sprint(0)))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "3", val_res)

	// Check that event credential stored
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", fmt.Sprint(0)))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, eventCredential, val_res)
}

func TestCommand_Resell(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()
	pkByte, _ := signer.GetPublicKey().MarshalText()

	cmd := eventCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	pk, name, num_tickets, price, max_resale_price, resale_royalty := string(pkByte), "Event", "100", "50", "50", "10"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err := cmd.init(snap, step)
	require.NoError(t, err)

	// Buy ticket
	buyNumTickets := "10"
	buyPayment := "50"
	eventCredential := "eventCredential"
	step = makeStep(t, signer, BuyPKArg, pk, BuyNumTicketsArg, buyNumTickets, BuyPaymentArg, buyPayment, BuyEventCredentialArg, eventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// Set ticket for sale for too much
	resellPrice := "100"
	resellNumTickets := "1"
	step = makeStep(t, signer, ResellPKArg, pk, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.EqualError(t, err, "Invalid resale price: 'resale price cannot be greater than max resale price'")

	// Set too many tickets for sale
	resellPrice = "100"
	resellNumTickets = "11"
	step = makeStep(t, signer, ResellPKArg, pk, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.EqualError(t, err, "Invalid resale number of tickets: reseller does not have 11 tickets")

	// Set ticket for sale for $50
	resellPrice = "50"
	resellNumTickets = "1"
	step = makeStep(t, signer, ResellPKArg, pk, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.NoError(t, err)

	// Check that signer now has a ticket up for sale
	key := []byte("event:ticket_resellers")
	val, _ := snap.Get(key)
	val_res := string(val)
	expected_res := fmt.Sprintf("%s;", pk)
	require.Equal(t, expected_res, val_res)

	// Check the number of tickets signer has up for sale
	userIndex := "0"
	key = []byte(fmt.Sprintf("event:reseller_tickets_number:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, resellNumTickets, val_res)

	// Check the price of the tickets the signer has up for sale
	key = []byte(fmt.Sprintf("event:reseller_tickets_price:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "50.000000", val_res)

	// Set second ticket for sale for $50
	resellPrice = "50"
	resellNumTickets = "1"
	step = makeStep(t, signer, ResellPKArg, pk, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.NoError(t, err)

	// Check that signer now has a ticket up for sale
	key = []byte("event:ticket_resellers")
	val, _ = snap.Get(key)
	val_res = string(val)
	expected_res = fmt.Sprintf("%s;", pk)
	require.Equal(t, expected_res, val_res)

	// Check the number of tickets signer has up for sale
	userIndex = "0"
	key = []byte(fmt.Sprintf("event:reseller_tickets_number:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "2", val_res)

	// Check the price of the tickets the signer has up for sale
	key = []byte(fmt.Sprintf("event:reseller_tickets_price:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "100.000000", val_res)
}

func TestCommand_Rebuy(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()
	pkByte, _ := signer.GetPublicKey().MarshalText()

	cmd := eventCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	pk, name, num_tickets, price, max_resale_price, resale_royalty := string(pkByte), "Event", "100", "50", "50", "10"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err := cmd.init(snap, step)
	require.NoError(t, err)

	// Rebuy Ticket
	rebuyPrice := "100"
	rebuyNumTickets := "1"
	rebuyEventCredential := "eventCredential"
	step = makeStep(t, signer, RebuyPKArg, pk, RebuyNumTicketsArg, rebuyNumTickets, RebuyPriceArg, rebuyPrice, RebuyEventCredentialArg, rebuyEventCredential)
	err = cmd.rebuy(snap, step)
	require.NoError(t, err)

	// Check that signer now has a ticket to rebuy
	key := []byte("event:ticket_rebuyers")
	val, _ := snap.Get(key)
	val_res := string(val)
	expected_res := fmt.Sprintf("%s;", pk)
	require.Equal(t, expected_res, val_res)

	// Check the number of tickets signer has proposed to rebuy
	userIndex := "0"
	key = []byte(fmt.Sprintf("event:rebuyer_tickets_number:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, rebuyNumTickets, val_res)

	// Check the price of the tickets the signer has proposed to rebuy
	key = []byte(fmt.Sprintf("event:rebuyer_tickets_price:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "100.000000", val_res)

	// Check event credential of the signer
	key = []byte(fmt.Sprintf("event:rebuyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential", val_res)
}

func TestCommand_HandleResales(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := bls.NewSigner()
	signer2 := bls.NewSigner()
	signer3 := bls.NewSigner()
	signer4 := bls.NewSigner()
	pk1Byte, _ := signer1.GetPublicKey().MarshalText()
	pk2Byte, _ := signer2.GetPublicKey().MarshalText()
	pk3Byte, _ := signer3.GetPublicKey().MarshalText()
	pk4Byte, _ := signer4.GetPublicKey().MarshalText()
	pk1 := string(pk1Byte)
	pk2 := string(pk2Byte)
	pk3 := string(pk3Byte)
	pk4 := string(pk4Byte)

	cmd := eventCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	pk, name, num_tickets, price, max_resale_price, resale_royalty := string(pk1Byte), "Event", "100", "100", "100", "10"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err := cmd.init(snap, step)
	require.NoError(t, err)

	// User 1 Buy 10 Tickets
	buyPrice := "100"
	buyNumTickets := "10"
	buyEventCredential := "eventCredential1"
	step = makeStep(t, signer1, BuyPKArg, pk1, BuyNumTicketsArg, buyNumTickets, BuyPaymentArg, buyPrice, BuyEventCredentialArg, buyEventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// User 2 Buy 10 Tickets
	buyPrice = "100"
	buyNumTickets = "10"
	buyEventCredential = "eventCredential2"
	step = makeStep(t, signer2, BuyPKArg, pk2, BuyNumTicketsArg, buyNumTickets, BuyPaymentArg, buyPrice, BuyEventCredentialArg, buyEventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// User 1 Resell 5 Tickets
	resellPrice := "100"
	resellNumTickets := "5"
	step = makeStep(t, signer1, ResellPKArg, pk1, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.NoError(t, err)

	// User 2 Resell 5 Tickets
	resellPrice = "100"
	resellNumTickets = "5"
	step = makeStep(t, signer2, ResellPKArg, pk2, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.NoError(t, err)

	// User 3 Rebuy 5 Tickets
	rebuyPrice := "100"
	rebuyNumTickets := "5"
	rebuyEventCredential := "eventCredential3"
	step = makeStep(t, signer3, RebuyPKArg, pk3, RebuyNumTicketsArg, rebuyNumTickets, RebuyPriceArg, rebuyPrice, RebuyEventCredentialArg, rebuyEventCredential)
	err = cmd.rebuy(snap, step)
	require.NoError(t, err)

	// User 4 Rebuy 5 Tickets
	rebuyPrice = "100"
	rebuyNumTickets = "5"
	rebuyEventCredential = "eventCredential4"
	step = makeStep(t, signer4, RebuyPKArg, pk4, RebuyNumTicketsArg, rebuyNumTickets, RebuyPriceArg, rebuyPrice, RebuyEventCredentialArg, rebuyEventCredential)
	err = cmd.rebuy(snap, step)
	require.NoError(t, err)

	// User 2 try to handle resales
	step = makeStep(t, signer2, HandleResalesPKArg, pk2)
	err = cmd.handleResales(snap, step)
	require.EqualError(t, err, fmt.Sprintf("'%s' is not the event owner. '%s' is the event owner", pk2, pk1))

	// User 1 handle resales
	step = makeStep(t, signer1, HandleResalesPKArg, pk1)
	err = cmd.handleResales(snap, step)
	require.NoError(t, err)

	// Check that every user has 5 tickets
	// User 1
	userIndex := "0"
	key := []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ := snap.Get(key)
	val_res := string(val)
	require.Equal(t, "5", val_res)
	// User 2
	userIndex = "1"
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "5", val_res)
	// User 3
	userIndex = "2"
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "5", val_res)
	// User 4
	userIndex = "3"
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "5", val_res)

	// Check that no resellers left
	key = []byte("event:resellers")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "", val_res)

	// Check that no rebuyers left
	key = []byte("event:rebuyers")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "", val_res)

	// Check that correct event credentials stored
	// User 1
	userIndex = "0"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential1", val_res)
	// User 2
	userIndex = "1"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential2", val_res)
	// User 3
	userIndex = "2"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential3", val_res)
	// User 4
	userIndex = "3"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential4", val_res)
}

func TestCommand_HandleResales_FewerRebuyers(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := bls.NewSigner()
	signer2 := bls.NewSigner()
	signer3 := bls.NewSigner()
	pk1Byte, _ := signer1.GetPublicKey().MarshalText()
	pk2Byte, _ := signer2.GetPublicKey().MarshalText()
	pk3Byte, _ := signer3.GetPublicKey().MarshalText()
	pk1 := string(pk1Byte)
	pk2 := string(pk2Byte)
	pk3 := string(pk3Byte)

	cmd := eventCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	pk, name, num_tickets, price, max_resale_price, resale_royalty := string(pk1Byte), "Event", "100", "100", "100", "10"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err := cmd.init(snap, step)
	require.NoError(t, err)

	// User 1 Buy 10 Tickets
	buyPrice := "100"
	buyNumTickets := "10"
	buyEventCredential := "eventCredential1"
	step = makeStep(t, signer1, BuyPKArg, pk1, BuyNumTicketsArg, buyNumTickets, BuyPaymentArg, buyPrice, BuyEventCredentialArg, buyEventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// User 2 Buy 10 Tickets
	buyPrice = "100"
	buyNumTickets = "10"
	buyEventCredential = "eventCredential2"
	step = makeStep(t, signer2, BuyPKArg, pk2, BuyNumTicketsArg, buyNumTickets, BuyPaymentArg, buyPrice, BuyEventCredentialArg, buyEventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// User 1 Resell 5 Tickets
	resellPrice := "100"
	resellNumTickets := "5"
	step = makeStep(t, signer1, ResellPKArg, pk1, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.NoError(t, err)

	// User 2 Resell 5 Tickets
	resellPrice = "100"
	resellNumTickets = "5"
	step = makeStep(t, signer2, ResellPKArg, pk2, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.NoError(t, err)

	// Check two resellers
	key := []byte("event:ticket_resellers")
	val, _ := snap.Get(key)
	val_res := string(val)
	require.Equal(t, fmt.Sprintf("%s;%s;", pk1, pk2), val_res)

	// User 3 Rebuy 5 Tickets
	rebuyPrice := "100"
	rebuyNumTickets := "5"
	rebuyEventCredential := "eventCredential3"
	step = makeStep(t, signer3, RebuyPKArg, pk3, RebuyNumTicketsArg, rebuyNumTickets, RebuyPriceArg, rebuyPrice, RebuyEventCredentialArg, rebuyEventCredential)
	err = cmd.rebuy(snap, step)
	require.NoError(t, err)

	// Check one rebuyer
	key = []byte("event:ticket_rebuyers")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, fmt.Sprintf("%s;", pk3), val_res)

	// User 1 handle resales
	step = makeStep(t, signer1, HandleResalesPKArg, pk1)
	err = cmd.handleResales(snap, step)
	require.NoError(t, err)

	// Check User 1 has 5 tickets
	userIndex := "0"
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "5", val_res)
	// Check User 2 has 10 tickets
	userIndex = "1"
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "10", val_res)
	// Check User 3 has 5 tickets
	userIndex = "2"
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "5", val_res)

	// Check that correct event credentials stored
	// User 1
	userIndex = "0"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential1", val_res)
	// User 2
	userIndex = "1"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential2", val_res)
	// User 3
	userIndex = "2"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential3", val_res)

	// Check that no rebuyers left
	key = []byte("event:ticket_rebuyers")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "", val_res)

	// Check that pk2 is a reseller left
	key = []byte("event:ticket_resellers")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, fmt.Sprintf("%s;", pk2), val_res)
}

func TestCommand_HandleResales_FewerResellers(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := bls.NewSigner()
	signer2 := bls.NewSigner()
	signer3 := bls.NewSigner()
	pk1Byte, _ := signer1.GetPublicKey().MarshalText()
	pk2Byte, _ := signer2.GetPublicKey().MarshalText()
	pk3Byte, _ := signer3.GetPublicKey().MarshalText()
	pk1 := string(pk1Byte)
	pk2 := string(pk2Byte)
	pk3 := string(pk3Byte)

	cmd := eventCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	pk, name, num_tickets, price, max_resale_price, resale_royalty := string(pk1Byte), "Event", "100", "100", "100", "10"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err := cmd.init(snap, step)
	require.NoError(t, err)

	// User 1 Buy 10 Tickets
	buyPrice := "100"
	buyNumTickets := "10"
	buyEventCredential := "eventCredential1"
	step = makeStep(t, signer1, BuyPKArg, pk1, BuyNumTicketsArg, buyNumTickets, BuyPaymentArg, buyPrice, BuyEventCredentialArg, buyEventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// Check event balance
	key := []byte("event:balance")
	val, _ := snap.Get(key)
	val_res := string(val)
	require.Equal(t, "1000.000000", val_res)

	// Check user 1 balance
	key = []byte("bank:account:0")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "-1000.000000", val_res)

	// User 1 Resell 5 Tickets
	resellPrice := "100"
	resellNumTickets := "5"
	step = makeStep(t, signer1, ResellPKArg, pk1, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.NoError(t, err)

	// Check one reseller
	key = []byte("event:ticket_resellers")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, fmt.Sprintf("%s;", pk1), val_res)

	// User 2 Rebuy 5 Tickets
	rebuyPrice := "100"
	rebuyNumTickets := "5"
	rebuyEventCredential := "eventCredential2"
	step = makeStep(t, signer2, RebuyPKArg, pk2, RebuyNumTicketsArg, rebuyNumTickets, RebuyPriceArg, rebuyPrice, RebuyEventCredentialArg, rebuyEventCredential)
	err = cmd.rebuy(snap, step)
	require.NoError(t, err)

	// User 3 Rebuy 5 Tickets
	rebuyPrice = "100"
	rebuyNumTickets = "5"
	rebuyEventCredential = "eventCredential3"
	step = makeStep(t, signer3, RebuyPKArg, pk3, RebuyNumTicketsArg, rebuyNumTickets, RebuyPriceArg, rebuyPrice, RebuyEventCredentialArg, rebuyEventCredential)
	err = cmd.rebuy(snap, step)
	require.NoError(t, err)

	// Check two rebuyers
	key = []byte("event:ticket_rebuyers")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, fmt.Sprintf("%s;%s;", pk2, pk3), val_res)

	// User 1 handle resales
	step = makeStep(t, signer1, HandleResalesPKArg, pk1)
	err = cmd.handleResales(snap, step)
	require.NoError(t, err)

	// Check User 1 has 5 tickets
	userIndex := "0"
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "5", val_res)
	// Check User 2 has 5 tickets
	userIndex = "1"
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "5", val_res)
	// Check User 3 has 0 tickets
	userIndex = "2"
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "", val_res)

	// Check that correct event credentials stored
	// User 1
	userIndex = "0"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential1", val_res)
	// User 2
	userIndex = "1"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "eventCredential2", val_res)
	// User 3
	userIndex = "2"
	key = []byte(fmt.Sprintf("event:buyer_event_credential:%s", userIndex))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "", val_res)

	// Check that one rebuyer is left
	key = []byte("event:ticket_rebuyers")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, fmt.Sprintf("%s;", pk3), val_res)

	// Check that no resellers left
	key = []byte("event:ticket_resellers")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "", val_res)

	// Check event balance
	key = []byte("event:balance")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "1050.000000", val_res)

	// Check user 1 balance
	key = []byte("bank:account:0")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "-550.000000", val_res)
}

func TestCommand_CheckBalances(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := bls.NewSigner()
	signer2 := bls.NewSigner()
	pk1Byte, _ := signer1.GetPublicKey().MarshalText()
	pk2Byte, _ := signer2.GetPublicKey().MarshalText()
	pk1 := string(pk1Byte)
	pk2 := string(pk2Byte)

	cmd := eventCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	pk, name, num_tickets, price, max_resale_price, resale_royalty := string(pk1Byte), "Event", "100", "100", "100", "10"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitPKArg, pk, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err := cmd.init(snap, step)
	require.NoError(t, err)

	// User 1 Buy 10 Tickets
	buyPrice := "100"
	buyNumTickets := "10"
	buyEventCredential := "eventCredential1"
	step = makeStep(t, signer1, BuyPKArg, pk1, BuyNumTicketsArg, buyNumTickets, BuyPaymentArg, buyPrice, BuyEventCredentialArg, buyEventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// Check event balance
	key := []byte("event:balance")
	val, _ := snap.Get(key)
	val_res := string(val)
	require.Equal(t, "1000.000000", val_res)

	// User 1 Resell 5 Tickets
	resellPrice := "100"
	resellNumTickets := "1"
	step = makeStep(t, signer1, ResellPKArg, pk1, ResellNumTicketsArg, resellNumTickets, ResellPriceArg, resellPrice)
	err = cmd.resell(snap, step)
	require.NoError(t, err)

	// User 2 Rebuy 1 Tickets
	rebuyPrice := "100"
	rebuyNumTickets := "1"
	rebuyEventCredential := "eventCredential2"
	step = makeStep(t, signer2, RebuyPKArg, pk2, RebuyNumTicketsArg, rebuyNumTickets, RebuyPriceArg, rebuyPrice, RebuyEventCredentialArg, rebuyEventCredential)
	err = cmd.rebuy(snap, step)
	require.NoError(t, err)

	// User 1 handle resales
	step = makeStep(t, signer1, HandleResalesPKArg, pk1)
	err = cmd.handleResales(snap, step)
	require.NoError(t, err)

	// Check event balance
	key = []byte("event:balance")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "1010.000000", val_res)

	// Check reseller balance
	key = []byte("bank:account:0")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "-910.000000", val_res)

	// Check rebuyer balance
	key = []byte("bank:account:1")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "-100.000000", val_res)
}

func TestCommand_UseTicket(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()
	pkByte, _ := signer.GetPublicKey().MarshalText()

	cmd := eventCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	pk1, name, num_tickets, price, max_resale_price, resale_royalty := string(pkByte), "Event", "100", "100", "100", "10"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer, InitPKArg, pk1, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err := cmd.init(snap, step)
	require.NoError(t, err)

	// Buy 10 ticket
	buyPayment := "100"
	numTickets := "10"
	eventCredential := "eventCredential1"
	step = makeStep(t, signer, BuyPKArg, pk1, BuyNumTicketsArg, numTickets, BuyPaymentArg, buyPayment, BuyEventCredentialArg, eventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// Use 5 tickets
	numTicketsToUse := "5"
	eventCredential = "eventCredential1"
	step = makeStep(t, signer, UseTicketPKArg, pk1, UseTicketNumTicketsArg, numTicketsToUse, UseTicketEventCredentialArg, eventCredential)
	err = cmd.useTicket(snap, step)
	require.NoError(t, err)

	// Check that 5 tickets remaining
	key := []byte(fmt.Sprintf("event:buyer_tickets:%s", fmt.Sprint(0)))
	val, _ := snap.Get(key)
	val_res := string(val)
	require.Equal(t, "5", val_res)

	// Check that signer is still a ticket owner
	key = []byte("event:ticket_owners")
	val, _ = snap.Get(key)
	val_res = strings.Trim(string(val), ";")
	require.Equal(t, pk1, val_res)

	// Use 5 tickets
	numTicketsToUse = "5"
	step = makeStep(t, signer, UseTicketPKArg, pk1, UseTicketNumTicketsArg, numTicketsToUse, UseTicketEventCredentialArg, eventCredential)
	err = cmd.useTicket(snap, step)
	require.NoError(t, err)

	// Check that 0 tickets remaining
	key = []byte(fmt.Sprintf("event:buyer_tickets:%s", fmt.Sprint(0)))
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "0", val_res)

	// Check that signer is not a ticket owner
	key = []byte("event:ticket_owners")
	val, _ = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "", val_res)
}

func TestCommand_ReadEventContract(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer1 := bls.NewSigner()
	signer2 := bls.NewSigner()
	pk1Byte, _ := signer1.GetPublicKey().MarshalText()
	pk2Byte, _ := signer2.GetPublicKey().MarshalText()
	pk1 := string(pk1Byte)
	pk2 := string(pk2Byte)

	cmd := eventCommand{
		Contract: &contract,
	}

	// Initialize smart contract
	pk1, name, num_tickets, price, max_resale_price, resale_royalty := pk1, "Event", "100", "50", "50", "10"
	snap := fake.NewSnapshot()
	step := makeStep(t, signer1, InitPKArg, pk1, InitNameArg, name, InitNumTicketsArg, num_tickets, InitPriceArg, price, InitMaxResalePriceArg, max_resale_price, InitResaleRoyaltyArg, resale_royalty)
	err := cmd.init(snap, step)
	require.NoError(t, err)

	// Buy ticket
	buyPayment := "50"
	buyNumTickets := "1"
	buyEventCredential := "eventCredential1"
	step = makeStep(t, signer1, BuyPKArg, pk1, BuyNumTicketsArg, buyNumTickets, BuyPaymentArg, buyPayment, BuyEventCredentialArg, buyEventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// Buy 3 tickets
	buyPayment = "50"
	buyNumTickets = "3"
	buyEventCredential = "eventCredential2"
	step = makeStep(t, signer1, BuyPKArg, pk2, BuyNumTicketsArg, buyNumTickets, BuyPaymentArg, buyPayment, BuyEventCredentialArg, buyEventCredential)
	err = cmd.buy(snap, step)
	require.NoError(t, err)

	// Check number of tickets left
	_, err = cmd.readEventContract(snap, step)
	require.NoError(t, err)
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
	str string
	err error
}

func (c fakeCmd) init(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) buy(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) resell(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) rebuy(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) handleResales(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) useTicket(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) readEventContract(snap store.Snapshot, step execution.Step) (string, error) {
	return c.str, c.err
}
