// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

// ISO operations are optional for drivers used by template-only builds.
type isoDriver interface {
	SetBootOrder(string, []string) error
	StartVM(string) (*view.VmInstanceInventoryView, error)
	DetachISO(string, string) error
	Console(string) (*consoleAddress, error)
}
type consoleAddress struct {
	HostIP   string `json:"hostIp"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Password string `json:"-"`
}

func (d *ZStackDriver) SetBootOrder(id string, order []string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/vm-instances", id, map[string]any{"setVmBootOrder": map[string]any{"bootOrder": order}}, nil)
	})
}
func (d *ZStackDriver) StartVM(id string) (*view.VmInstanceInventoryView, error) {
	var out view.VmInstanceInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/vm-instances", id, map[string]any{"startVmInstance": map[string]any{}}, &out)
	})
	if err == nil && (out.UUID == "" || out.State != "Running") {
		err = fmt.Errorf("VM start returned no Running inventory")
	}
	return &out, err
}
func (d *ZStackDriver) DetachISO(vm, id string) error {
	return d.request(func(ctx context.Context) error {
		_, err := d.client.do(ctx, http.MethodDelete, "v1/vm-instances/"+url.PathEscape(vm)+"/iso", url.Values{"isoUuid": {id}}, nil)
		return err
	})
}
func (d *ZStackDriver) Console(id string) (*consoleAddress, error) {
	var out consoleAddress
	err := d.request(func(ctx context.Context) error {
		data, err := d.client.do(ctx, http.MethodGet, "v1/vm-instances/"+url.PathEscape(id)+"/console-addresses", nil, nil)
		if err != nil {
			return err
		}
		if err = data.Unmarshal(&out); err != nil {
			return err
		}
		data, err = d.client.do(ctx, http.MethodGet, "v1/vm-instances/"+url.PathEscape(id)+"/console-passwords", nil, nil)
		if err != nil {
			return err
		}
		out.Password, _ = data.GetString("password")
		return nil
	})
	return &out, err
}
