// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestImageDeletionOnlyIgnoresExactMissingImage(t *testing.T) {
	for _, tc := range []struct {
		name, code, field, id, kind string
		missing                     bool
	}{
		{"missing-image", "SYS.1003", "imageUuid", "owned-image", "ImageVO", true},
		{"other-image", "SYS.1003", "imageUuid", "other-image", "ImageVO", false},
		{"missing-dependency", "SYS.1003", "volumeUuid", "owned-image", "VolumeVO", false},
		{"permission-denied", "ID.1001", "imageUuid", "owned-image", "ImageVO", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Contains(t, []string{http.MethodDelete, http.MethodPut}, r.Method)
				path := "/zstack/v1/images/owned-image"
				if r.Method == http.MethodPut {
					path += "/actions"
				}
				require.Equal(t, path, r.URL.Path)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": tc.code, "formatArgs": []string{tc.field, tc.id, tc.kind}, "details": "not found"}})
			}))
			defer server.Close()
			client, err := newAPIClient(&AccessConfig{Host: server.URL})
			require.NoError(t, err)
			d := &ZStackDriver{client: client, secrets: []string{"test-secret"}}
			for _, call := range []func(string) error{d.DeleteImage, d.ExpungeImage} {
				err := call("owned-image")
				if tc.missing {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
			}
		})
	}
	require.False(t, imageAlreadyAbsent(context.DeadlineExceeded, "owned-image"))
}

// Supply the UUID of a failed import owned by this acceptance run. Never point
// this opt-in test at a live image; absence is checked before either mutation.
func TestAcceptanceISOImageCleanup(t *testing.T) {
	id := os.Getenv("ZSTACK_ACC_ISO_CLEANUP_UUID")
	if os.Getenv("ZSTACK_ACC_ISO") != "1" || id == "" {
		t.Skip("set ISO acceptance config and owned absent image UUID")
	}
	data, err := os.ReadFile(os.Getenv("ZSTACK_ACC_ISO_CONFIG_FILE"))
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	b := &ISOBuilder{}
	_, _, err = b.Prepare(raw)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	d, err := b.config.AccessConfig.driver(ctx, b.config.APITimeout())
	require.NoError(t, err)
	_, err = d.GetImage(id)
	require.ErrorContains(t, err, "expected one inventory, got 0")
	require.NoError(t, d.DeleteImage(id))
	require.NoError(t, d.ExpungeImage(id))
	require.NoError(t, d.DeleteImage(id))
	require.NoError(t, d.ExpungeImage(id))
	t.Log("Owned absent import Delete/Expunge are idempotent against the real cloud")
}
