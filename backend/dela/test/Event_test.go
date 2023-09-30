package integration

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	accessContract "go.dedis.ch/dela/contracts/access"
	eventContract "go.dedis.ch/dela/contracts/event"
	"go.dedis.ch/dela/core/txn"
	"go.dedis.ch/dela/core/txn/signed"
	"go.dedis.ch/dela/crypto/bls"
	"go.dedis.ch/dela/crypto/loader"
)

func init() {
	rand.Seed(0)
}

// Start 3 nodes
// Use the auction contract
// Check the state
func Test_Event_Correctness(t *testing.T) {
	numNodes := 3

	dir, err := os.MkdirTemp(os.TempDir(), "dela-event-test")
	require.NoError(t, err)

	timeout := time.Second * 10 // transaction inclusion timeout

	defer os.RemoveAll(dir)

	nodes := make([]dela, numNodes)

	for i := range nodes {
		node := newDelaNode(t, filepath.Join(dir, "node"+strconv.Itoa(i)), 0)
		nodes[i] = node
	}

	nodes[0].Setup(nodes[1:]...)

	l := loader.NewFileLoader(filepath.Join(dir, "private.key"))

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
	err = addAndWait(t, timeout, manager, nodes[0].(cosiDelaNode), args...)
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
	err = addAndWait(t, timeout, manager, nodes[0].(cosiDelaNode), args...)
	require.NoError(t, err)

	// *******************************************************************************
	// INIT COMMAND
	// *******************************************************************************
	name, numTickets, price, maxResalePrice, resaleRoyalty := "Event Name", "1000", "50", "50", "10"
	args = getInitCommandArgs(pubKeyString, name, numTickets, price, maxResalePrice, resaleRoyalty)
	err = addAndWait(t, timeout, manager, nodes[0].(cosiDelaNode), args...)
	require.NoError(t, err)

	// Check Event Name set correctly
	proof, err := nodes[0].GetOrdering().GetProof([]byte("event:name"))
	require.NoError(t, err)
	require.Equal(t, name, string(proof.GetValue()))

	// Check Price set correctly
	proof, err = nodes[0].GetOrdering().GetProof([]byte("event:price"))
	require.NoError(t, err)
	require.Equal(t, price, string(proof.GetValue()))

	// Check Max Resale Price set correctly
	proof, err = nodes[0].GetOrdering().GetProof([]byte("event:max_resale_price"))
	require.NoError(t, err)
	require.Equal(t, maxResalePrice, string(proof.GetValue()))

	// Check Resale Royalty set correctly
	proof, err = nodes[0].GetOrdering().GetProof([]byte("event:resale_royalty"))
	require.NoError(t, err)
	require.Equal(t, resaleRoyalty, string(proof.GetValue()))

	// *******************************************************************************
	// BUY COMMAND
	// *******************************************************************************
	payment := "50"
	numTickets = "5"
	args = getBuyCommandArgs(pubKeyString, numTickets, payment)
	err = addAndWait(t, timeout, manager, nodes[0].(cosiDelaNode), args...)
	require.NoError(t, err)

	// Check Ticket Owners set correctly
	proof, err = nodes[0].GetOrdering().GetProof([]byte("event:ticket_owners"))
	require.NoError(t, err)
	require.Equal(t, pubKeyString, strings.Split(string(proof.GetValue()), ";")[0])

	// *******************************************************************************
	// RESELL COMMAND
	// *******************************************************************************
	resellPrice := "50"
	resellNumTickets := "5"
	args = getResellCommandArgs(pubKeyString, resellNumTickets, resellPrice)
	err = addAndWait(t, timeout, manager, nodes[0].(cosiDelaNode), args...)
	require.NoError(t, err)

	// Check Ticket Resellers set correctly
	proof, err = nodes[0].GetOrdering().GetProof([]byte("event:ticket_resellers"))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s;", pubKeyString), string(proof.GetValue()))

	// *******************************************************************************
	// REBUY COMMAND
	// *******************************************************************************
	rebuyPrice := "50"
	rebuyNumTickets := "5"
	args = getRebuyCommandArgs(pubKeyString, rebuyNumTickets, rebuyPrice)
	err = addAndWait(t, timeout, manager, nodes[0].(cosiDelaNode), args...)
	require.NoError(t, err)

	// Check Ticket Rebuyers set correctly
	proof, err = nodes[0].GetOrdering().GetProof([]byte("event:ticket_rebuyers"))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s;", pubKeyString), string(proof.GetValue()))

	// *******************************************************************************
	// HANDLERESALE COMMAND
	// *******************************************************************************
	args = getHandleResalesCommandArgs(pubKeyString)
	err = addAndWait(t, timeout, manager, nodes[0].(cosiDelaNode), args...)
	require.NoError(t, err)

	// Check Ticket Owners set correctly
	proof, err = nodes[0].GetOrdering().GetProof([]byte("event:ticket_owners"))
	require.NoError(t, err)
	require.Equal(t, pubKeyString, strings.Split(string(proof.GetValue()), ";")[0])

	// Check Ticket Resellers set correctly
	proof, err = nodes[0].GetOrdering().GetProof([]byte("event:ticket_resellers"))
	require.NoError(t, err)
	require.Equal(t, "", string(proof.GetValue()))

}

// Helper functions

// Gets tx args for the event init command
func getInitCommandArgs(pk string, name string, numTickets string, price string, maxResalePrice string, resaleRoyalty string) []txn.Arg {
	args := []txn.Arg{
		{Key: "go.dedis.ch/dela.ContractArg", Value: []byte(eventContract.ContractName)},
		{Key: "value:initPK", Value: []byte(pk)},
		{Key: "value:initName", Value: []byte(name)},
		{Key: "value:initNumTickets", Value: []byte(numTickets)},
		{Key: "value:initPrice", Value: []byte(price)},
		{Key: "value:initMaxResalePrice", Value: []byte(maxResalePrice)},
		{Key: "value:initResaleRoyalty", Value: []byte(resaleRoyalty)},
		{Key: "value:command", Value: []byte("INIT")},
	}

	return args
}

// Gets tx args for the event buy command
func getBuyCommandArgs(pk string, numTickets string, payment string) []txn.Arg {
	args := []txn.Arg{
		{Key: "go.dedis.ch/dela.ContractArg", Value: []byte(eventContract.ContractName)},
		{Key: "value:buyPK", Value: []byte(pk)},
		{Key: "value:buyNumTickets", Value: []byte(numTickets)},
		{Key: "value:buyPayment", Value: []byte(payment)},
		{Key: "value:command", Value: []byte("BUY")},
	}

	return args
}

// Gets tx args for the event resell command
func getResellCommandArgs(pk string, resellNumTickets string, resellPrice string) []txn.Arg {
	args := []txn.Arg{
		{Key: "go.dedis.ch/dela.ContractArg", Value: []byte(eventContract.ContractName)},
		{Key: "value:resellPK", Value: []byte(pk)},
		{Key: "value:resellNumTickets", Value: []byte(resellNumTickets)},
		{Key: "value:resellPrice", Value: []byte(resellPrice)},
		{Key: "value:command", Value: []byte("RESELL")},
	}

	return args
}

// Gets tx args for the event rebuy command
func getRebuyCommandArgs(pk string, rebuyNumTickets string, rebuyPrice string) []txn.Arg {
	args := []txn.Arg{
		{Key: "go.dedis.ch/dela.ContractArg", Value: []byte(eventContract.ContractName)},
		{Key: "value:rebuyPK", Value: []byte(pk)},
		{Key: "value:rebuyNumTickets", Value: []byte(rebuyNumTickets)},
		{Key: "value:rebuyPrice", Value: []byte(rebuyPrice)},
		{Key: "value:command", Value: []byte("REBUY")},
	}

	return args
}

// Gets tx args for the handleResale command
func getHandleResalesCommandArgs(pk string) []txn.Arg {
	args := []txn.Arg{
		{Key: "go.dedis.ch/dela.ContractArg", Value: []byte(eventContract.ContractName)},
		{Key: "value:handleResalesPK", Value: []byte(pk)},
		{Key: "value:command", Value: []byte("HANDLERESALES")},
	}

	return args
}
