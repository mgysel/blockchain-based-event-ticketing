package bank

import (
	"fmt"
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

func TestExecute(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{err: fake.GetError()})
	signer := fake.NewSigner()

	err := contract.Execute(fakeStore{}, makeStep(t, signer))
	require.EqualError(t, err, "identity not authorized: fake.PublicKey ("+fake.GetError().Error()+")")

	contract = NewContract([]byte{}, fakeAccess{})
	err = contract.Execute(fakeStore{}, makeStep(t, signer))
	require.EqualError(t, err, "'value:command' not found in tx arg")

	contract.cmd = fakeCmd{err: fake.GetError()}

	err = contract.Execute(fakeStore{}, makeStep(t, signer, CmdArg, "DEPOSIT"))
	require.EqualError(t, err, fake.Err("failed to DEPOSIT"))

	err = contract.Execute(fakeStore{}, makeStep(t, signer, CmdArg, "WITHDRAW"))
	require.EqualError(t, err, fake.Err("failed to WITHDRAW"))

	err = contract.Execute(fakeStore{}, makeStep(t, signer, CmdArg, "TRANSFER"))
	require.EqualError(t, err, fake.Err("failed to TRANSFER"))

	contract.cmd = fakeCmd{}
	err = contract.Execute(fakeStore{}, makeStep(t, signer, CmdArg, "DEPOSIT"))
	require.NoError(t, err)
}

func TestDeposit(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()

	cmd := bankCommand{
		Contract: &contract,
	}

	// No deposit argument
	err := cmd.deposit(fake.NewSnapshot(), makeStep(t, signer))
	require.EqualError(t, err, "'value:deposit' not found in tx arg")

	// Make deposit
	snap := fake.NewSnapshot()
	deposit := "10"
	step := makeStep(t, signer, DepositArg, deposit)
	err = cmd.deposit(snap, step)

	// Check deposit successful
	pk, err := signer.GetPublicKey().MarshalText()
	key := []byte(fmt.Sprintf("bank:%s", string(pk)))
	val, err := snap.Get(key)
	val_res := string(val)
	require.Equal(t, deposit, val_res)

	// Make second deposit
	deposit = "15"
	step = makeStep(t, signer, DepositArg, deposit)
	err = cmd.deposit(snap, step)

	// Check deposit successful
	key = []byte(fmt.Sprintf("bank:%s", string(pk)))
	val, err = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "25", val_res)
}

func TestWithdraw(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()

	cmd := bankCommand{
		Contract: &contract,
	}

	snap := fake.NewSnapshot()

	// No withdraw argument
	err := cmd.withdraw(fake.NewSnapshot(), makeStep(t, signer))
	require.EqualError(t, err, "'value:withdraw' not found in tx arg")

	// Make withdraw without depositing
	withdraw := "10"
	step := makeStep(t, signer, WithdrawArg, withdraw)
	err = cmd.withdraw(snap, step)
	require.EqualError(t, err, "failed to remove withdraw: Withdraw is greater than balance")

	// Check no funds withdrawn
	pk, err := signer.GetPublicKey().MarshalText()
	key := []byte(fmt.Sprintf("bank:%s", string(pk)))
	val, err := snap.Get(key)
	val_res := string(val)
	require.Equal(t, "0", val_res)

	// Make deposit
	deposit := "15"
	step = makeStep(t, signer, DepositArg, deposit)
	err = cmd.deposit(snap, step)

	// Make withdraw
	withdraw = "10"
	step = makeStep(t, signer, WithdrawArg, withdraw)
	err = cmd.withdraw(snap, step)

	// Check funds withdrawn
	pk, err = signer.GetPublicKey().MarshalText()
	key = []byte(fmt.Sprintf("bank:%s", string(pk)))
	val, err = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "5", val_res)
}

func TestTransfer(t *testing.T) {
	contract := NewContract([]byte{}, fakeAccess{})
	signer := bls.NewSigner()

	cmd := bankCommand{
		Contract: &contract,
	}

	snap := fake.NewSnapshot()

	// No transfer_amount/transfer_account arguments
	err := cmd.transfer(fake.NewSnapshot(), makeStep(t, signer))
	require.EqualError(t, err, "'value:transfer_amount' not found in tx arg")
	err = cmd.transfer(fake.NewSnapshot(), makeStep(t, signer, TransferAmountArg, "10"))
	require.EqualError(t, err, "'value:transfer_account' not found in tx arg")

	// Make deposit
	deposit := "15"
	step := makeStep(t, signer, DepositArg, deposit)
	err = cmd.deposit(snap, step)

	// Make Transfer
	transfer := "10"
	step = makeStep(t, signer, TransferAmountArg, transfer, TransferAccountArg, "auction")
	err = cmd.transfer(snap, step)

	// Check correct funds removed from 'from' account
	pk, err := signer.GetPublicKey().MarshalText()
	key := []byte(fmt.Sprintf("bank:%s", string(pk)))
	val, err := snap.Get(key)
	val_res := string(val)
	require.Equal(t, "5", val_res)

	// Check correct funds added to 'to' account
	key = []byte("bank:auction")
	val, err = snap.Get(key)
	val_res = string(val)
	require.Equal(t, "10", val_res)
}

// -----------------------------------------------------------------------------
// Utility functions

func makeStep(t *testing.T, signer crypto.Signer, args ...string) execution.Step {
	return execution.Step{Current: makeTx(t, signer, args...)}
}

func makeTx(t *testing.T, signer crypto.Signer, args ...string) txn.Transaction {
	_, err := signer.GetPublicKey().MarshalBinary()
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

func (c fakeCmd) deposit(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) withdraw(snap store.Snapshot, step execution.Step) error {
	return c.err
}

func (c fakeCmd) transfer(snap store.Snapshot, step execution.Step) error {
	return c.err
}
