// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type ownedResource struct {
	UUID    string
	Created bool
	Deleted bool
	Removed bool
}
type buildResources struct {
	VM            ownedResource
	ImportedImage ownedResource
	Image         ownedResource
	DataVolumes   []ownedResource
	ImageReady    bool
	ExportRemoved bool
}

func resources(state multistep.StateBag) *buildResources {
	if v, ok := state.GetOk("resources"); ok {
		return v.(*buildResources)
	}
	r := &buildResources{}
	state.Put("resources", r)
	return r
}
func newResource() ownedResource {
	return ownedResource{UUID: strings.ReplaceAll(uuid.NewString(), "-", "")}
}
func buildFailed(state multistep.StateBag) bool {
	for _, key := range []string{"error", multistep.StateCancelled, multistep.StateHalted} {
		if _, ok := state.GetOk(key); ok {
			return true
		}
	}
	return false
}
func halt(state multistep.StateBag, err error) multistep.StepAction {
	state.Put("error", err)
	state.Get("ui").(packersdk.Ui).Error(err.Error())
	return multistep.ActionHalt
}
func stepDriver(ctx context.Context, state multistep.StateBag) Driver {
	return state.Get("driver").(Driver).WithContext(ctx)
}
func cleanupWarning(state multistep.StateBag, kind, id string, err error) {
	msg := fmt.Sprintf("Cleanup incomplete for %s %s: %v. Inspect this UUID before retrying the build.", kind, id, err)
	state.Get("ui").(packersdk.Ui).Error(msg)
	warnings, _ := state.Get("cleanup_warnings").([]string)
	state.Put("cleanup_warnings", append(warnings, msg))
}
func cleanupImage(state multistep.StateBag, r *ownedResource) {
	if r.UUID == "" || r.Removed {
		return
	}
	config := state.Get("config").(*Config)
	ctx, cancel := context.WithTimeout(context.Background(), config.CleanupTimeout())
	defer cancel()
	d := stepDriver(ctx, state)
	build := resources(state)
	if r.UUID == build.Image.UUID && config.ImageUrl != "" && !build.ExportRemoved {
		if err := d.DeleteExportedImage(config.BackupStorageUuid, r.UUID); err != nil {
			cleanupWarning(state, "exported image", r.UUID, err)
			return
		}
		build.ExportRemoved = true
	}
	if !r.Deleted {
		if err := d.DeleteImage(r.UUID); err != nil {
			cleanupWarning(state, "image", r.UUID, err)
			return
		}
		r.Deleted = true
	}
	if err := d.ExpungeImage(r.UUID); err != nil {
		cleanupWarning(state, "image", r.UUID, err)
		return
	}
	r.Removed = true
}
func cleanupVM(state multistep.StateBag) {
	r := resources(state)
	if r.VM.UUID == "" {
		return
	}
	config := state.Get("config").(*Config)
	ctx, cancel := context.WithTimeout(context.Background(), config.CleanupTimeout())
	defer cancel()
	d := stepDriver(ctx, state)
	// A cancelled create may still finish on the server. The preassigned UUID is
	// retained in warnings; never search by name or delete an unowned resource.
	if !r.VM.Removed && !r.VM.Created {
		vm, err := d.GetVmInstance(r.VM.UUID)
		if err != nil {
			cleanupWarning(state, "pending VM", r.VM.UUID, err)
			return
		}
		if vm == nil {
			cleanupWarning(state, "pending VM", r.VM.UUID, fmt.Errorf("empty response"))
			return
		}
		r.VM.Created = true
		for _, v := range vm.AllVolumes {
			if v.Type == "Data" {
				r.DataVolumes = append(r.DataVolumes, ownedResource{UUID: v.UUID, Created: true})
			}
		}
	}
	if !r.VM.Removed && !r.VM.Deleted {
		if err := d.DestroyVmInstance(r.VM.UUID); err != nil {
			cleanupWarning(state, "VM", r.VM.UUID, err)
			return
		}
		r.VM.Deleted = true
	}
	if !r.VM.Removed {
		if err := d.DeleteVmInstance(r.VM.UUID); err != nil {
			cleanupWarning(state, "VM", r.VM.UUID, err)
			return
		}
		r.VM.Removed = true
		config.InstanceUuid = ""
	}
	// ZStack can detach data disks when destroying a VM. These disks were created
	// by this build, are excluded from the root template, and must be expunged.
	for i := range r.DataVolumes {
		v := &r.DataVolumes[i]
		if v.Removed {
			continue
		}
		if !v.Deleted {
			if err := d.DeleteDataVolume(v.UUID); err != nil {
				cleanupWarning(state, "data volume", v.UUID, err)
				continue
			}
			v.Deleted = true
		}
		if err := d.ExpungeDataVolume(v.UUID); err != nil {
			cleanupWarning(state, "data volume", v.UUID, err)
			continue
		}
		v.Removed = true
	}
}

type sanitizedError struct {
	cause   error
	message string
}

func (e *sanitizedError) Error() string { return e.message }
func (e *sanitizedError) Unwrap() error { return e.cause }

var loggedURL = regexp.MustCompile(`(?i)(?:https?|sftp)://[^\s"'<>]+`)

func redactError(err error, secrets []string) error {
	if err == nil {
		return nil
	}
	text := err.Error()
	for _, secret := range secrets {
		if secret != "" {
			encoded, _ := json.Marshal(secret)
			for _, value := range []string{secret, url.QueryEscape(secret), base64.StdEncoding.EncodeToString([]byte(secret)), string(encoded[1 : len(encoded)-1])} {
				text = strings.ReplaceAll(text, value, "[REDACTED]")
			}
		}
	}
	text = loggedURL.ReplaceAllStringFunc(text, func(raw string) string {
		u, e := url.Parse(raw)
		if e != nil {
			return "[REDACTED URL]"
		}
		u.User = nil
		u.RawQuery = ""
		u.Fragment = ""
		return u.String()
	})
	return &sanitizedError{cause: err, message: text}
}
