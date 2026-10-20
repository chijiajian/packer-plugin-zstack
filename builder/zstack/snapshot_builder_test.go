// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func snapshotBuilderInput() map[string]any {
	return map[string]any{"zstack_host": "localhost", "account_name": "admin", "account_password": "test", "backup_storage_uuid": "bs", "source_volume_snapshot_uuid": "snapshot", "image_name": "output"}
}

func TestSnapshotBuilderPrepare(t *testing.T) {
	var b SnapshotBuilder
	_, warnings, err := b.Prepare(snapshotBuilderInput())
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Equal(t, "none", b.config.Comm.Type)
	require.NotContains(t, b.ConfigSpec(), "ssh_username")
	require.NotContains(t, b.ConfigSpec(), "network_uuid")
	require.Contains(t, b.ConfigSpec(), "source_volume_snapshot_uuid")
}

func TestSnapshotBuilderRejectsVMOptions(t *testing.T) {
	for _, key := range []string{"image_uuid", "source_image_url", "ssh_username", "network_uuid", "instance_name", "cpu_num", "iso_install", "userdata"} {
		t.Run(key, func(t *testing.T) {
			input := snapshotBuilderInput()
			input[key] = "unused"
			var b SnapshotBuilder
			_, _, err := b.Prepare(input)
			require.Error(t, err)
		})
	}
}

func TestSnapshotBuilderRequiresSnapshotAndNoCommunicator(t *testing.T) {
	input := snapshotBuilderInput()
	delete(input, "source_volume_snapshot_uuid")
	var b SnapshotBuilder
	_, _, err := b.Prepare(input)
	require.ErrorContains(t, err, "requires source_volume_snapshot_uuid")
	input = snapshotBuilderInput()
	input["communicator"] = "ssh"
	_, _, err = b.Prepare(input)
	require.ErrorContains(t, err, "communicator = none")
}

func TestLegacySnapshotBuilderWarning(t *testing.T) {
	var b Builder
	_, warnings, err := b.Prepare(snapshotBuilderInput())
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "deprecated")
	require.Equal(t, "none", b.config.Comm.Type)
}
