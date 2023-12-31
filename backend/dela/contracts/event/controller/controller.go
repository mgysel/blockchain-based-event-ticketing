package controller

import (
	"go.dedis.ch/dela/cli"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/contracts/event"
	"go.dedis.ch/dela/core/access"
	"go.dedis.ch/dela/core/execution/native"
	"golang.org/x/xerrors"
)

// aKey is the access key used for the event contract
var aKey = [32]byte{2}

// miniController is a CLI initializer to register the event contract
//
// - implements node.Initializer
type miniController struct {
}

// NewController creates a new minimal controller for the event contract.
func NewController() node.Initializer {
	return miniController{}
}

// SetCommands implements node.Initializer.
func (miniController) SetCommands(builder node.Builder) {
}

// OnStart implements node.Initializer. It registers the event contract.
func (m miniController) OnStart(flags cli.Flags, inj node.Injector) error {
	var access access.Service
	err := inj.Resolve(&access)
	if err != nil {
		return xerrors.Errorf("failed to resolve access service: %v", err)
	}

	var exec *native.Service
	err = inj.Resolve(&exec)
	if err != nil {
		return xerrors.Errorf("failed to resolve native service: %v", err)
	}

	// Create and register event contract
	eventContract := event.NewContract(aKey[:], access)
	event.RegisterContract(exec, eventContract)

	return nil
}

// OnStop implements node.Initializer.
func (miniController) OnStop(inj node.Injector) error {
	return nil
}
