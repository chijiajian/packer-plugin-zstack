// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"fmt"

	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
	"time"
)

// Retain SDK wire models with a transport that enforces cancellation and deadlines.
type ZStackDriver struct {
	client  *apiClient
	ctx     context.Context
	timeout time.Duration
	secrets []string
}

func (d *ZStackDriver) WithContext(ctx context.Context) Driver {
	copy := *d
	copy.ctx = ctx
	return &copy
}

func (d *ZStackDriver) request(fn func(context.Context) error) error {
	ctx := d.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := d.timeout
	if timeout <= 0 {
		timeout = defaultAPITimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	err := fn(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return redactError(err, d.secrets)
}

type Driver interface {
	DeleteExportedImage(backupStorageUUID, imageUUID string) error
	WithContext(context.Context) Driver
	DeleteDataVolume(string) error
	ExpungeDataVolume(string) error
	GetBackupStorage(uuid string) (*view.BackupStorageInventoryView, error)
	QueryBackStorage(backupStorageName string) ([]view.BackupStorageInventoryView, error)
	GetImage(uuid string) (*view.ImageInventoryView, error)
	QueryImage(imageName string) ([]view.ImageInventoryView, error)
	GetVmInstance(uuid string) (*view.VmInstanceInventoryView, error)
	GetL3Network(uuid string) (*view.L3NetworkInventoryView, error)
	QueryL3Network(networkName string) ([]view.L3NetworkInventoryView, error)
	GetInstanceOffering(uuid string) (*view.InstanceOfferingInventoryView, error)
	QueryInstanceOffering(instanceOfferingName string) ([]view.InstanceOfferingInventoryView, error)
	GetVolume(uuid string) (*view.VolumeInventoryView, error)
	GetZone(uuid string) (*view.ZoneInventoryView, error)

	CreateVmInstance(vmInstance param.CreateVmInstanceParam) (*view.VmInstanceInventoryView, error)
	StopVminstance(uuid string) (*view.VmInstanceInventoryView, error)
	DestroyVmInstance(uuid string) error
	DeleteVmInstance(uuid string) error

	CreateVolumeSnapshot(volumeUuid string, params param.CreateVolumeSnapshotParam) (*view.VolumeSnapshotInventoryView, error)
	CreateImage(rootVolumeUuid string, params param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageInventoryView, error)
	CreateImageFromVolumeSnapshot(snapshotUuid string, params param.CreateRootVolumeTemplateFromVolumeSnapshotParam) (*view.ImageInventoryView, error)
	GetVolumeSnapshot(uuid string) (*view.VolumeSnapshotInventoryView, error)
	AddImage(param param.AddImageParam) (*view.ImageInventoryView, error)
	DeleteImage(uuid string) error
	ExpungeImage(uuid string) error
	CreateDataVolume(volume param.CreateDataVolumeParam) (*view.VolumeInventoryView, error)
	ExportImage(backupStorageUuid string, params param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageEventView, error)

	AttachGuestToolsToVm(vmUuid string) error
	AttachDataVolumeToVm(vmUuid, volumeUuid string) (*view.VolumeInventoryView, error)

	ValidateCredentials() error
}

func (d *ZStackDriver) GetBackupStorage(uuid string) (*view.BackupStorageInventoryView, error) {
	var out view.BackupStorageInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/backup-storage", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get BackupStorage: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("BackupStorage %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZStackDriver) GetImage(uuid string) (*view.ImageInventoryView, error) {
	var out view.ImageInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.Get(ctx, "v1/images", uuid, nil, &out) })
	if err != nil {
		return nil, fmt.Errorf("get Image: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("Image %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZStackDriver) GetVmInstance(uuid string) (*view.VmInstanceInventoryView, error) {
	var out view.VmInstanceInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/vm-instances", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get VmInstance: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("VmInstance %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZStackDriver) GetL3Network(uuid string) (*view.L3NetworkInventoryView, error) {
	var out view.L3NetworkInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/l3-networks", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get L3Network: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("L3Network %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZStackDriver) GetInstanceOffering(uuid string) (*view.InstanceOfferingInventoryView, error) {
	var out view.InstanceOfferingInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/instance-offerings", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get InstanceOffering: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("InstanceOffering %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZStackDriver) GetVolume(uuid string) (*view.VolumeInventoryView, error) {
	var out view.VolumeInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.Get(ctx, "v1/volumes", uuid, nil, &out) })
	if err != nil {
		return nil, fmt.Errorf("get Volume: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("Volume %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZStackDriver) GetZone(uuid string) (*view.ZoneInventoryView, error) {
	var out view.ZoneInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.Get(ctx, "v1/zones", uuid, nil, &out) })
	if err != nil {
		return nil, fmt.Errorf("get Zone: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("Zone %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZStackDriver) GetVolumeSnapshot(uuid string) (*view.VolumeSnapshotInventoryView, error) {
	var out view.VolumeSnapshotInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Get(ctx, "v1/volume-snapshots", uuid, nil, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("get VolumeSnapshot: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("VolumeSnapshot %s not found or returned empty inventory", uuid)
	}
	return &out, nil
}
func (d *ZStackDriver) QueryBackStorage(name string) ([]view.BackupStorageInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + name)
	var out []view.BackupStorageInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.List(ctx, "v1/backup-storage", &params, &out)
	})
	return out, err
}
func (d *ZStackDriver) QueryImage(name string) ([]view.ImageInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + name)
	var out []view.ImageInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.List(ctx, "v1/images", &params, &out) })
	return out, err
}
func (d *ZStackDriver) QueryL3Network(name string) ([]view.L3NetworkInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + name)
	var out []view.L3NetworkInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.List(ctx, "v1/l3-networks", &params, &out)
	})
	return out, err
}
func (d *ZStackDriver) QueryInstanceOffering(name string) ([]view.InstanceOfferingInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + name)
	var out []view.InstanceOfferingInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.List(ctx, "v1/instance-offerings", &params, &out)
	})
	return out, err
}
func (d *ZStackDriver) CreateVmInstance(params param.CreateVmInstanceParam) (*view.VmInstanceInventoryView, error) {
	var out view.VmInstanceInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/vm-instances", params, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateVmInstance: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateVmInstance returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZStackDriver) AddImage(params param.AddImageParam) (*view.ImageInventoryView, error) {
	var out view.ImageInventoryView
	err := d.request(func(ctx context.Context) error { return d.client.Post(ctx, "v1/images", params, &out) })
	if err != nil {
		return nil, fmt.Errorf("AddImage: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("AddImage returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZStackDriver) CreateImage(rootVolumeUuid string, params param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageInventoryView, error) {
	var out view.ImageInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/images/root-volume-templates/from/volumes/"+rootVolumeUuid, params, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateImage: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateImage returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZStackDriver) CreateImageFromVolumeSnapshot(snapshotUuid string, params param.CreateRootVolumeTemplateFromVolumeSnapshotParam) (*view.ImageInventoryView, error) {
	var out view.ImageInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/images/root-volume-templates/from/volume-snapshots/"+snapshotUuid, params, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateImageFromVolumeSnapshot: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateImageFromVolumeSnapshot returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZStackDriver) CreateVolumeSnapshot(volumeUuid string, params param.CreateVolumeSnapshotParam) (*view.VolumeSnapshotInventoryView, error) {
	var out view.VolumeSnapshotInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/volumes/"+volumeUuid+"/volume-snapshots", params, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateVolumeSnapshot: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateVolumeSnapshot returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZStackDriver) CreateDataVolume(params param.CreateDataVolumeParam) (*view.VolumeInventoryView, error) {
	var out view.VolumeInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/volumes/data", params, &out)
	})
	if err != nil {
		return nil, fmt.Errorf("CreateDataVolume: %w", err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("CreateDataVolume returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZStackDriver) StopVminstance(uuid string) (*view.VmInstanceInventoryView, error) {
	var out view.VmInstanceInventoryView
	params := map[string]any{"stopVmInstance": map[string]any{"type": "grace", "stopHA": "true"}}
	err := d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/vm-instances", uuid, params, &out)
	})
	if err != nil {
		return nil, err
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("StopVmInstance returned empty inventory UUID")
	}
	return &out, nil
}
func (d *ZStackDriver) ExportImage(uuid string, params param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageEventView, error) {
	var out view.ExportImageFromBackupStorageEventView
	err := d.request(func(ctx context.Context) error {
		return d.client.PutWithRespKey(ctx, "v1/backup-storage", uuid, "", map[string]any{"exportImageFromBackupStorage": params.Params}, &out)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
func (d *ZStackDriver) AttachGuestToolsToVm(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/vm-instances", uuid, map[string]any{"attachGuestToolsIsoToVm": map[string]any{}}, nil)
	})
}
func (d *ZStackDriver) AttachDataVolumeToVm(vmUuid, volumeUuid string) (*view.VolumeInventoryView, error) {
	var out view.VolumeInventoryView
	err := d.request(func(ctx context.Context) error {
		return d.client.Post(ctx, "v1/volumes/"+volumeUuid+"/vm-instances/"+vmUuid, param.AttachDataVolumeToVmParam{}, &out)
	})
	return &out, err
}
func (d *ZStackDriver) ValidateCredentials() error {
	params := param.NewQueryParam()
	var out []view.ZoneInventoryView
	return d.request(func(ctx context.Context) error { return d.client.List(ctx, "v1/zones", &params, &out) })
}
func (d *ZStackDriver) DestroyVmInstance(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Delete(ctx, "v1/vm-instances", uuid, string(param.DeleteModePermissive))
	})
}
func (d *ZStackDriver) DeleteImage(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Delete(ctx, "v1/images", uuid, string(param.DeleteModePermissive))
	})
}
func (d *ZStackDriver) DeleteDataVolume(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Delete(ctx, "v1/volumes", uuid, string(param.DeleteModePermissive))
	})
}
func (d *ZStackDriver) DeleteVmInstance(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/vm-instances", uuid, map[string]any{"expungeVmInstance": map[string]any{}}, nil)
	})
}
func (d *ZStackDriver) ExpungeImage(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/images", uuid, map[string]any{"expungeImage": map[string]any{}}, nil)
	})
}
func (d *ZStackDriver) ExpungeDataVolume(uuid string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Put(ctx, "v1/volumes", uuid, map[string]any{"expungeDataVolume": map[string]any{}}, nil)
	})
}

func (d *ZStackDriver) DeleteExportedImage(backupStorageUUID, imageUUID string) error {
	return d.request(func(ctx context.Context) error {
		return d.client.Delete(ctx, "v1/backup-storage/"+backupStorageUUID+"/exported-images", imageUUID, string(param.DeleteModePermissive))
	})
}
