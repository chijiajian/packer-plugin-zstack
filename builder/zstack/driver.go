package zstack

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/zstackio/zstack-sdk-go-v2/pkg/client"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

type ZStackDriver struct {
	client *client.ZSClient
}

type Driver interface {
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

	CreateImage(rootVolumeUuid string, params param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageInventoryView, error)
	AddImage(param param.AddImageParam) (*view.ImageInventoryView, error)
	CreateDataVolume(volume param.CreateDataVolumeParam) (*view.VolumeInventoryView, error)
	ExportImage(backupStorageUuid string, params param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageEventView, error)

	AttachGuestToolsToVm(vmUuid string) error
	AttachDataVolumeToVm(vmUuid, volumeUuid string) (*view.VolumeInventoryView, error)
}

func (d *ZStackDriver) GetBackupStorage(uuid string) (*view.BackupStorageInventoryView, error) {
	log.Printf("[DEBUG] Getting backup storage with UUID: %s", uuid)
	backupStorage, err := d.client.GetBackupStorage(uuid)
	if err != nil {
		log.Printf("[ERROR] Failed to get backup storage: %v", err)
		return nil, fmt.Errorf("failed to query backup storage: %v", err)
	}
	log.Printf("[INFO] Successfully retrieved backup storage")
	return backupStorage, nil
}

func (d *ZStackDriver) QueryBackStorage(backupStorageName string) ([]view.BackupStorageInventoryView, error) {
	log.Printf("[DEBUG] Querying backup storage with name: %s", backupStorageName)
	params := param.NewQueryParam()
	params.AddQ("name=" + backupStorageName)

	backupStorages, err := d.client.QueryBackupStorage(&params)
	if err != nil {
		log.Printf("[ERROR] Failed to query backup storage: %v", err)
		return nil, fmt.Errorf("failed to query backup storage: %v", err)
	}
	log.Printf("[INFO] Found %d backup storage(s)", len(backupStorages))
	return backupStorages, nil
}

func (d *ZStackDriver) GetImage(uuid string) (*view.ImageInventoryView, error) {
	image, err := d.client.GetImage(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query image: %v", err)
	}
	return image, nil
}

func (d *ZStackDriver) QueryImage(imageName string) ([]view.ImageInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + imageName)

	images, err := d.client.QueryImage(&params)
	if err != nil {
		return nil, fmt.Errorf("failed to query image: %v", err)
	}
	return images, nil
}

func (d *ZStackDriver) GetL3Network(uuid string) (*view.L3NetworkInventoryView, error) {
	l3Network, err := d.client.GetL3Network(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query L3 network: %v", err)
	}
	return l3Network, nil
}

func (d *ZStackDriver) QueryL3Network(networkName string) ([]view.L3NetworkInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + networkName)

	networks, err := d.client.QueryL3Network(&params)
	if err != nil {
		return nil, fmt.Errorf("failed to query networks: %v", err)
	}
	return networks, nil
}

func (d *ZStackDriver) GetInstanceOffering(uuid string) (*view.InstanceOfferingInventoryView, error) {
	instanceOffering, err := d.client.GetInstanceOffering(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query instance offering: %v", err)
	}
	return instanceOffering, nil
}

func (d *ZStackDriver) QueryInstanceOffering(instanceOfferingName string) ([]view.InstanceOfferingInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + instanceOfferingName)

	instanceOffering, err := d.client.QueryInstanceOffering(&params)
	if err != nil {
		return nil, fmt.Errorf("failed to query instance offering: %v", err)
	}
	return instanceOffering, nil
}

func (d *ZStackDriver) GetVmInstance(uuid string) (*view.VmInstanceInventoryView, error) {
	vmInstance, err := d.client.GetVmInstance(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query VM instance: %v", err)
	}
	return vmInstance, nil
}

func (d *ZStackDriver) GetVolume(uuid string) (*view.VolumeInventoryView, error) {
	volume, err := d.client.GetVolume(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query volume: %v", err)
	}
	if volume == nil {
		return nil, fmt.Errorf("volume not found %s", uuid)
	}
	return volume, nil
}

func (d *ZStackDriver) GetZone(uuid string) (*view.ZoneInventoryView, error) {
	zone, err := d.client.GetZone(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query zone: %v", err)
	}
	return zone, nil
}

func (d *ZStackDriver) CreateVmInstance(vmInstance param.CreateVmInstanceParam) (*view.VmInstanceInventoryView, error) {
	log.Printf("[INFO] Creating VM instance with name: %s", vmInstance.Params.Name)
	vm, err := d.client.CreateVmInstance(vmInstance)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM instance '%s': %v", vmInstance.Params.Name, err)
	}
	log.Printf("[INFO] Successfully created VM instance with UUID: %s", vm.UUID)
	return vm, nil
}

func (d *ZStackDriver) StopVminstance(uuid string) (*view.VmInstanceInventoryView, error) {
	log.Printf("[INFO] Stopping VM instance '%s'", uuid)
	stopType := "grace"
	stopHA := "true"
	vmInstance, err := d.client.StopVmInstance(uuid, param.StopVmInstanceParam{
		Params: param.StopVmInstanceParamDetail{
			Type:   &stopType,
			StopHA: &stopHA,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to stop VM instance '%s': %v", uuid, err)
	}
	return vmInstance, nil
}

func (d *ZStackDriver) DeleteVmInstance(uuid string) error {
	log.Printf("[INFO] Deleting VM instance '%s'", uuid)
	err := d.client.ExpungeVmInstance(uuid)
	if err != nil {
		return fmt.Errorf("failed to delete VM instance '%s': %v", uuid, err)
	}
	log.Printf("[INFO] Successfully deleted VM instance '%s'", uuid)
	return nil
}

func (d *ZStackDriver) DestroyVmInstance(uuid string) error {
	log.Printf("[INFO] Destroying VM instance '%s'", uuid)
	err := d.client.DestroyVmInstance(uuid, param.DeleteModePermissive)
	if err != nil {
		return fmt.Errorf("failed to destroy VM instance '%s': %v", uuid, err)
	}
	log.Printf("[INFO] Successfully destroyed VM instance '%s'", uuid)
	return nil
}

func (d *ZStackDriver) CreateImage(rootVolumeUuid string, rootVolumeParam param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageInventoryView, error) {
	log.Printf("[INFO] Creating image from root volume.")
	img, err := d.client.CreateRootVolumeTemplateFromRootVolume(rootVolumeUuid, rootVolumeParam)
	if err != nil {
		return nil, fmt.Errorf("failed to create image: %v", err)
	}
	log.Printf("[INFO] Successfully created image '%s'", img.UUID)
	return img, nil
}

func (d *ZStackDriver) AddImage(image param.AddImageParam) (*view.ImageInventoryView, error) {
	log.Printf("[INFO] Adding image '%s'", image.Params.Name)
	img, err := d.client.AddImage(image)
	if err != nil {
		return nil, fmt.Errorf("failed to add image '%s': %v", image.Params.Name, err)
	}
	log.Printf("[INFO] Successfully added image '%s' with UUID: %s", img.Name, img.UUID)
	return img, nil
}

func (d *ZStackDriver) ExportImage(backupStorageUuid string, params param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageEventView, error) {
	log.Printf("[INFO] Exporting image from backup storage '%s'", backupStorageUuid)
	exportedImg, err := d.client.ExportImageFromBackupStorage(backupStorageUuid, params)
	if err != nil {
		return nil, fmt.Errorf("failed to export image: %v", err)
	}
	return exportedImg, nil
}

func (d *ZStackDriver) CreateDataVolume(volume param.CreateDataVolumeParam) (*view.VolumeInventoryView, error) {
	vol, err := d.client.CreateDataVolume(volume)
	if err != nil {
		return nil, fmt.Errorf("failed to create data volume: %v", err)
	}
	return vol, nil
}

func (d *ZStackDriver) AttachGuestToolsToVm(vmUuid string) error {
	log.Printf("[INFO] Attaching guest tools to VM '%s'", vmUuid)
	_, err := d.client.AttachGuestToolsIsoToVm(vmUuid, param.AttachGuestToolsIsoToVmParam{})
	if err != nil {
		return fmt.Errorf("failed to attach guest tools to VM '%s': %v", vmUuid, err)
	}
	log.Printf("[INFO] Successfully attached guest tools to VM '%s'", vmUuid)
	return nil
}

func (d *ZStackDriver) AttachDataVolumeToVm(vmUuid, volumeUuid string) (*view.VolumeInventoryView, error) {
	log.Printf("[INFO] Attaching data volume '%s' to VM '%s'", volumeUuid, vmUuid)
	datavol, err := d.client.AttachDataVolumeToVm(volumeUuid, vmUuid, param.AttachDataVolumeToVmParam{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach data volume '%s' to VM '%s': %v", volumeUuid, vmUuid, err)
	}
	log.Printf("[INFO] Successfully attached data volume '%s' to VM '%s'", volumeUuid, vmUuid)
	return datavol, nil
}

func (d *ZStackDriver) WaitForSSH(vmUuid string, sshPort int, timeout time.Duration) error {
	log.Printf("[INFO] Waiting for SSH connectivity on VM %s", vmUuid)
	vm, err := d.GetVmInstance(vmUuid)
	if err != nil {
		return fmt.Errorf("failed to get VM instance: %v", err)
	}

	if len(vm.VmNics) == 0 || vm.VmNics[0].Ip == "" {
		return fmt.Errorf("VM '%s' has no default IP to connect", vmUuid)
	}
	ip := vm.VmNics[0].Ip

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for SSH on VM '%s'", vmUuid)
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, sshPort), 5*time.Second)
			if err == nil {
				conn.Close()
				log.Printf("[INFO] Successfully established SSH connection to %s:%d", ip, sshPort)
				return nil
			}
			log.Printf("[DEBUG] SSH connection attempt to %s:%d failed, retrying...", ip, sshPort)
		}
	}
}

func addSystemTags(tags []string, args ...string) []string {
	tags = append(tags, args...)
	return tags
}
