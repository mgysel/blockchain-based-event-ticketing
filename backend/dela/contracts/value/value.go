// Package value implements a simple native contract that can store, delete, and
// display values.
package value

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"go.dedis.ch/dela"
	"go.dedis.ch/dela/core/access"
	"go.dedis.ch/dela/core/execution"
	"go.dedis.ch/dela/core/execution/native"
	"go.dedis.ch/dela/core/store"
	"golang.org/x/xerrors"
)

// commands defines the commands of the value contract. This interface helps in
// testing the contract.
type commands interface {
	write(snap store.Snapshot, step execution.Step) error
	read(snap store.Snapshot, step execution.Step) error
	delete(snap store.Snapshot, step execution.Step) error
	list(snap store.Snapshot) error
}

const (
	// ContractName is the name of the contract.
	ContractName = "go.dedis.ch/dela.Value"

	// KeyArg is the argument's name in the transaction that contains the
	// provided key to update.
	KeyArg = "value:key"

	// ValueArg is the argument's name in the transaction that contains the
	// provided value to set.
	ValueArg = "value:value"

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
	// CmdWrite defines the command to write a value
	CmdWrite Command = "WRITE"

	// CmdRead defines a command to read a value
	CmdRead Command = "READ"

	// CmdDelete defines a command to delete a value
	CmdDelete Command = "DELETE"

	// CmdList defines a command to list all values set (and not deleted)
	// so far.
	CmdList Command = "LIST"
)

// NewCreds creates new credentials for a value contract execution. We might
// want to use in the future a separate credential for each command.
func NewCreds(id []byte) access.Credential {
	return access.NewContractCreds(id, ContractName, credentialAllCommand)
}

// RegisterContract registers the value contract to the given execution service.
func RegisterContract(exec *native.Service, c Contract) {
	exec.Set(ContractName, c)
}

// Contract is a simple smart contract that allows one to handle the storage by
// performing CRUD operations.
//
// - implements native.Contract
type Contract struct {
	// index contains all the keys set (and not delete) by this contract so far
	index map[string]struct{}

	// access is the access control service managing this smart contract
	access access.Service

	// accessKey is the access identifier allowed to use this smart contract
	accessKey []byte

	// cmd provides the commands that can be executed by this smart contract
	cmd commands

	// printer is the output used by the READ and LIST commands
	printer io.Writer
}

// NewContract creates a new Value contract
func NewContract(aKey []byte, srvc access.Service) Contract {
	contract := Contract{
		index:     map[string]struct{}{},
		access:    srvc,
		accessKey: aKey,
		printer:   infoLog{},
	}

	contract.cmd = valueCommand{Contract: &contract}

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
	case CmdWrite:
		err := c.cmd.write(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to WRITE: %v", err)
		}
	case CmdRead:
		err := c.cmd.read(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to READ: %v", err)
		}
	case CmdDelete:
		err := c.cmd.delete(snap, step)
		if err != nil {
			return xerrors.Errorf("failed to DELETE: %v", err)
		}
	case CmdList:
		err := c.cmd.list(snap)
		if err != nil {
			return xerrors.Errorf("failed to LIST: %v", err)
		}
	default:
		return xerrors.Errorf("unknown command: %s", cmd)
	}

	return nil
}

// valueCommand implements the commands of the value contract
//
// - implements commands
type valueCommand struct {
	*Contract
}

// write implements commands. It performs the WRITE command
func (c valueCommand) write(snap store.Snapshot, step execution.Step) error {
	key := step.Current.GetArg(KeyArg)
	if len(key) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", KeyArg)
		fmt.Fprintln(c.printer, outputWriteFailure(errMessage, string(key), ""))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputWriteFailure(errMessage, string(key), ""))
		return xerrors.Errorf(errMessage)
	}

	value := step.Current.GetArg(ValueArg)
	if len(value) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", ValueArg)
		fmt.Fprintln(c.printer, outputWriteFailure(errMessage, string(key), string(value)))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputWriteFailure(errMessage, string(key), string(value)))
		return xerrors.Errorf(errMessage)
	}

	err := snap.Set(key, value)
	if err != nil {
		errMessage := fmt.Sprintf("failed to set value: %v", err)
		fmt.Fprintln(c.printer, outputWriteFailure(errMessage, string(key), string(value)))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputWriteFailure(errMessage, string(key), string(value)))
		return xerrors.Errorf("failed to set value: %v", err)
	}

	c.index[string(key)] = struct{}{}

	dela.Logger.Info().Str("contract", ContractName).Msgf("VALUECONTRACT_WRITEOUTPUT: //setting %x==%s//", key, value)

	fmt.Fprintln(c.printer, outputWriteSuccess(string(key), string(value)))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputWriteSuccess(string(key), string(value)))
	return nil
}

func outputWriteSuccess(key string, value string) string {
	return fmt.Sprintf("//VALUECONTRACT_WRITEOUTPUT;success;%s;%s//", key, value)
}

func outputWriteFailure(errMessage string, key string, value string) string {
	return fmt.Sprintf("//VALUECONTRACT_WRITEOUTPUT;error;%s;%s;%s//", errMessage, key, value)
}

// read implements commands. It performs the READ command
func (c valueCommand) read(snap store.Snapshot, step execution.Step) error {
	key := step.Current.GetArg(KeyArg)
	if len(key) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", KeyArg)
		fmt.Fprintln(c.printer, outputReadFailure(errMessage, string(key)))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadFailure(errMessage, string(key)))
		return xerrors.Errorf(errMessage)
	}

	val, err := snap.Get(key)
	if err != nil {
		errMessage := fmt.Sprintf("failed to get key '%s': %v", key, err)
		fmt.Fprintln(c.printer, outputReadFailure(errMessage, string(key)))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadFailure(errMessage, string(key)))
		return xerrors.Errorf(errMessage)
	}

	fmt.Fprintln(c.printer, outputReadSuccess(string(key), string(val)))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputReadSuccess(string(key), string(val)))
	return nil
}

func outputReadSuccess(key string, value string) string {
	return fmt.Sprintf("//VALUECONTRACT_READOUTPUT;success;%s;%s//", key, value)
}

func outputReadFailure(errMessage string, key string) string {
	return fmt.Sprintf("//VALUECONTRACT_READOUTPUT;error;%s;%s//", errMessage, key)
}

// delete implements commands. It performs the DELETE command
func (c valueCommand) delete(snap store.Snapshot, step execution.Step) error {
	key := step.Current.GetArg(KeyArg)
	if len(key) == 0 {
		errMessage := fmt.Sprintf("'%s' not found in tx arg", KeyArg)
		fmt.Fprintln(c.printer, outputDeleteFailure(errMessage, string(key)))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputDeleteFailure(errMessage, string(key)))
		return xerrors.Errorf(errMessage, KeyArg)
	}

	err := snap.Delete(key)
	if err != nil {
		errMessage := fmt.Sprintf("failed to delete key '%s': %v", key, err)
		fmt.Fprintln(c.printer, outputDeleteFailure(errMessage, string(key)))
		dela.Logger.Info().Str("contract", ContractName).Msgf(outputDeleteFailure(errMessage, string(key)))
		return xerrors.Errorf("failed to delete key '%x': %v", key, err)
	}

	delete(c.index, string(key))
	fmt.Fprintln(c.printer, outputDeleteSuccess(string(key)))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputDeleteSuccess(string(key)))

	return nil
}

func outputDeleteSuccess(key string) string {
	return fmt.Sprintf("//VALUECONTRACT_DELETEOUTPUT;success;%s//", key)
}

func outputDeleteFailure(errMessage string, key string) string {
	return fmt.Sprintf("//VALUECONTRACT_DELETEOUTPUT;error;%s;%s//", errMessage, key)
}

// list implements commands. It performs the LIST command
func (c valueCommand) list(snap store.Snapshot) error {
	res := []string{}

	for k := range c.index {
		v, err := snap.Get([]byte(k))
		if err != nil {
			errMessage := fmt.Sprintf("failed to get key '%s': %v", k, err)
			fmt.Fprintln(c.printer, outputListFailure(errMessage))
			dela.Logger.Info().Str("contract", ContractName).Msgf(outputListFailure(errMessage))
			return xerrors.Errorf(errMessage)
		}

		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(res)
	outputString := strings.Join(res, ";")

	fmt.Fprintln(c.printer, outputListSuccess(outputString))
	dela.Logger.Info().Str("contract", ContractName).Msgf(outputListSuccess(outputString))
	return nil
}

func outputListSuccess(keyValuePairs string) string {
	return fmt.Sprintf("//VALUECONTRACT_LISTOUTPUT;success;%s//", keyValuePairs)
}

func outputListFailure(errMessage string) string {
	return fmt.Sprintf("//VALUECONTRACT_LISTOUTPUT;error;%s//", errMessage)
}

// infoLog defines an output using zerolog
//
// - implements io.writer
type infoLog struct{}

func (h infoLog) Write(p []byte) (int, error) {
	dela.Logger.Info().Msg(string(p))

	return len(p), nil
}
