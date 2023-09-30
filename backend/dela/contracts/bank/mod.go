// Package value implements a simple bank contract
// Users can deposit, withdraw, transfer money to bank account
package bank

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
	BalanceKey string = "bank:"
)

// commands defines the commands of the bank contract. This interface helps in
// testing the contract.
type commands interface {
	deposit(snap store.Snapshot, step execution.Step) error
	withdraw(snap store.Snapshot, step execution.Step) error
	transfer(snap store.Snapshot, step execution.Step) error
}

const (
	// ContractName is the name of the contract.
	ContractName = "go.dedis.ch/dela.Bank"

	// DepositArg is the argument's name in the transaction that contains the
	// amount to deposit.
	DepositArg = "value:deposit"

	// WithdrawArg is the argument's name in the transaction that contains the
	// provided value to withdraw.
	WithdrawArg = "value:withdraw"

	// TransferAmountArg is the argument's name in the transaction that contains the
	// provided value to transfer.
	TransferAmountArg = "value:transfer_amount"

	// TransferAmountArg is the argument's name in the transaction that contains the
	// provided value to transfer funds to.
	TransferAccountArg = "value:transfer_account"

	// CmdArg is the argument's name to indicate the kind of command we want to
	// run on the contract. Should be one of the Command type.
	CmdArg = "value:command"

	// credentialAllCommand defines the credential command that is allowed to
	// perform all commands.
	credentialAllCommand = "all"
)

// Command defines a type of command for the value contract
type Command string

const (
	// CmdWrite defines the command to deposit a value
	CmdDeposit Command = "DEPOSIT"

	// CmdRead defines a command to withdraw a value
	CmdWithdraw Command = "WITHDRAW"

	// CmdTransfer defines a command to transfer funds from one account to another
	CmdTransfer Command = "TRANSFER"
)

// NewCreds creates new credentials for a bank contract execution. We might
// want to use in the future a separate credential for each command.
func NewCreds(id []byte) access.Credential {
	return access.NewContractCreds(id, ContractName, credentialAllCommand)
}

// RegisterContract registers the bank contract to the given execution service.
func RegisterContract(exec *native.Service, c Contract) {
	exec.Set(ContractName, c)
}

// Contract is a simple smart contract that allows for a simple bank
//
// - implements native.Contract
type Contract struct {
	// store is esed to store/retrieve data
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

// NewContract creates a new Bank contract
func NewContract(aKey []byte, srvc access.Service) Contract {
	contract := Contract{
		store:     value.NewContract(aKey, srvc),
		access:    srvc,
		accessKey: aKey,
		printer:   infoLog{},
	}

	contract.cmd = bankCommand{Contract: &contract}

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
	case CmdDeposit:
		err := c.cmd.deposit(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to DEPOSIT: %v", err)
		}
	case CmdWithdraw:
		err := c.cmd.withdraw(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to WITHDRAW: %v", err)
		}
	case CmdTransfer:
		err := c.cmd.transfer(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to TRANSFER: %v", err)
		}
	default:
		return xerrors.Errorf("unknown command: %s", cmd)
	}

	return nil
}

// bankCommand implements the commands of the value contract
//
// - implements commands
type bankCommand struct {
	*Contract
}

// ############################################################################
// ########################### Deposit FUNCTIONS ##############################
// ############################################################################

// write implements commands. It performs the DEPOSIT command
func (c bankCommand) deposit(snap store.Snapshot, step execution.Step) error {
	// Get deposit argument
	depositByte := step.Current.GetArg(DepositArg)
	if len(depositByte) == 0 {
		return xerrors.Errorf("'%s' not found in tx arg", DepositArg)
	}

	// Is the deposit valid?
	deposit, err := byteToInt(depositByte)
	if err != nil {
		return err
	}

	// Obtain public key from this txn
	pk, err := step.Current.GetIdentity().MarshalText()
	if err != nil {
		return xerrors.Errorf("failed to obtain public key in tx")
	}

	err = addDeposit(snap, pk, deposit)
	if err != nil {
		return xerrors.Errorf("failed to add deposit: %v", err)
	}

	return nil
}

func addDeposit(snap store.Snapshot, pk []byte, deposit int) error {
	// Obtain balance
	key := []byte(fmt.Sprintf("%v%v", BalanceKey, string(pk)))
	balance, err := getBalanceInt(snap, pk)
	if err != nil {
		return xerrors.Errorf("failed to get balance: %v", err)
	}

	newBalance := balance + deposit
	newBalanceByte := []byte(fmt.Sprint(newBalance))
	err = snap.Set(key, newBalanceByte)
	if err != nil {
		return xerrors.Errorf("failed to set balance: %v", err)
	}

	return nil
}

// ############################################################################
// ########################## Withdraw FUNCTIONS ##############################
// ############################################################################

// write implements commands. It performs the DEPOSIT command
func (c bankCommand) withdraw(snap store.Snapshot, step execution.Step) error {
	// Get withdraw argument
	withdrawByte := step.Current.GetArg(WithdrawArg)
	if len(withdrawByte) == 0 {
		return xerrors.Errorf("'%s' not found in tx arg", WithdrawArg)
	}

	// Is the withdraw valid?
	withdraw, err := byteToInt(withdrawByte)
	if err != nil {
		return err
	}

	// Obtain public key from this txn
	pk, err := step.Current.GetIdentity().MarshalText()
	if err != nil {
		return xerrors.Errorf("failed to obtain public key in tx")
	}

	err = removeWithdraw(snap, pk, withdraw)
	if err != nil {
		return xerrors.Errorf("failed to remove withdraw: %v", err)
	}

	return nil
}

func removeWithdraw(snap store.Snapshot, pk []byte, withdraw int) error {
	// Obtain balance
	key := []byte(fmt.Sprintf("%v%v", BalanceKey, string(pk)))
	balance, err := getBalanceInt(snap, pk)
	if err != nil {
		return xerrors.Errorf("failed to get balance: %v", err)
	}

	// Check if withdraw > balance
	if withdraw > balance {
		return xerrors.Errorf("Withdraw is greater than balance")
	}

	// Set new balance
	balance = balance - withdraw
	balanceByte := []byte(fmt.Sprint(balance))
	err = snap.Set(key, balanceByte)
	if err != nil {
		return xerrors.Errorf("failed to set balance: %v", err)
	}

	return nil
}

// ############################################################################
// ########################## Transfer FUNCTIONS ##############################
// ############################################################################

// write implements commands. It performs the DEPOSIT command
func (c bankCommand) transfer(snap store.Snapshot, step execution.Step) error {
	// Get withdraw argument
	transferAmountByte := step.Current.GetArg(TransferAmountArg)
	if len(transferAmountByte) == 0 {
		return xerrors.Errorf("'%s' not found in tx arg", TransferAmountArg)
	}
	transferAccountByte := step.Current.GetArg(TransferAccountArg)
	if len(transferAccountByte) == 0 {
		return xerrors.Errorf("'%s' not found in tx arg", TransferAccountArg)
	}

	// Is the transfer ammount valid?
	transferAmount, err := byteToInt(transferAmountByte)
	if err != nil {
		return err
	}

	// Obtain public key from this txn
	pk, err := step.Current.GetIdentity().MarshalText()
	if err != nil {
		return xerrors.Errorf("failed to obtain public key in tx")
	}

	err = transferFunds(snap, pk, transferAccountByte, transferAmount)
	if err != nil {
		return xerrors.Errorf("failed to remove withdraw: %v", err)
	}

	return nil
}

func transferFunds(snap store.Snapshot, from []byte, to []byte, amount int) error {
	// Obtain balance
	balance, err := getBalanceInt(snap, from)
	if err != nil {
		return xerrors.Errorf("failed to get balance: %v", err)
	}

	// Check if transfer > balance
	newAmount := amount
	if amount > balance {
		newAmount = balance
	}

	// Remove funds from "from"
	err = removeWithdraw(snap, from, newAmount)
	if err != nil {
		return xerrors.Errorf("failed to remove transfer: %v", err)
	}

	// Add funds to "to"
	err = addDeposit(snap, to, newAmount)
	if err != nil {
		return xerrors.Errorf("failed to add transfer: %v", err)
	}

	return nil
}

// ############################################################################
// ########################### Helper FUNCTIONS ###############################
// ############################################################################

func getBalanceInt(snap store.Snapshot, pk []byte) (int, error) {
	// Obtain balance
	key := []byte(fmt.Sprintf("%v%v", BalanceKey, string(pk)))
	balanceByte, err := snap.Get(key)
	if err != nil {
		return -1, xerrors.Errorf("failed to get balance: %v", err)
	}
	// If balance is empty, set to 0
	if len(balanceByte) == 0 {
		err = snap.Set(key, []byte("0"))
		if err != nil {
			return -1, xerrors.Errorf("failed to set balance to 0: %v", err)
		}
		balanceByte = []byte("0")
	}
	// Convert balance to integer
	balance, err := byteToInt(balanceByte)
	if err != nil {
		return -1, xerrors.Errorf("failed to convert balance to int: %v", err)
	}

	return balance, nil
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
