// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"context"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

type StepExpungeVmInstance struct{}

func (s *StepExpungeVmInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	cleanupVM(state)
	return multistep.ActionContinue
}
func (s *StepExpungeVmInstance) Cleanup(state multistep.StateBag) {}
