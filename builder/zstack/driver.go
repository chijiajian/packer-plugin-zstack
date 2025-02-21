package zstack

import (
	"fmt"
	"log"
	"net"
	"time"

	"zstack.io/zstack-sdk-go/pkg/client"
	"zstack.io/zstack-sdk-go/pkg/param"
	"zstack.io/zstack-sdk-go/pkg/view"
)

type ZStackDriver struct {
	client *client.ZSClient
}

type Driver interface {
	GetBackupStorage(uuid string) (*view.BackupStorageInventoryView, error)
	QueryBackStorage(backupStorageName string) ([]view.BackupStorageInventoryView, error)
	GetImage(uuid string) (*view.ImageView, error)
	QueryImage(imageName string) ([]view.ImageView, error)
	GetVmInstance(uuid string) (*view.VmInstanceInventoryView, error)
	GetL3Network(uuid string) (*view.L3NetworkInventoryView, error)
	QueryL3Network(networkName string) ([]view.L3NetworkInventoryView, error)
	GetInstanceOffering(uuid string) (*view.InstanceOfferingInventoryView, error)
	QueryInstanceOffering(instanceOfferingName string) ([]view.InstanceOfferingInventoryView, error)
	GetVolume(uuid string) (*view.VolumeView, error)
	GetZone(uuid string) (*view.ZoneView, error)

	CreateVmInstance(vmInstance param.CreateVmInstanceParam) (*view.VmInstanceInventoryView, error)
	StopVminstance(uuid string) (*view.VmInstanceInventoryView, error)
	DeleteVmInstance(uuid string) error

	CreateImage(uuid param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageView, error)
	CreateDataVolume(volume param.CreateDataVolumeParam) (*view.VolumeView, error)
	ExportImage(image param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageResultView, error)

	AttachGuestToolsToVm(vmUuid string) error
	AttachDataVolumeToVm(vmUuid, volumeUuid string) (*view.VolumeView, error)
}

func (d *ZStackDriver) GetBackupStorage(uuid string) (*view.BackupStorageInventoryView, error) {
	backupStorage, err := d.client.GetBackupStorage(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query backup storage: %v", err)
	}
	return backupStorage, nil
}

func (d *ZStackDriver) QueryBackStorage(backupStorageName string) ([]view.BackupStorageInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + backupStorageName)

	backupStorages, err := d.client.QueryBackupStorage(params)
	if err != nil {
		return nil, fmt.Errorf("failed to query image storage: %v", err)
	}
	return backupStorages, nil
}

func (d *ZStackDriver) GetImage(uuid string) (*view.ImageView, error) {
	image, err := d.client.GetImage(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query image: %v", err)
	}
	return image, nil
}

func (d *ZStackDriver) QueryImage(imageName string) ([]view.ImageView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + imageName)

	images, err := d.client.QueryImage(params)
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
	return &l3Network, nil
}

func (d *ZStackDriver) QueryL3Network(networkName string) ([]view.L3NetworkInventoryView, error) {
	params := param.NewQueryParam()
	params.AddQ("name=" + networkName)

	networks, err := d.client.QueryL3Network(params)
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

	instanceOffering, err := d.client.QueryInstaceOffering(params)
	if err != nil {
		return nil, fmt.Errorf("failed to query instance offering %v", err)
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

func (d *ZStackDriver) GetVolume(uuid string) (*view.VolumeView, error) {
	volume, err := d.client.GetVolume(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query volume: %v", err)
	}
	if volume == nil {
		return nil, fmt.Errorf("volume not found %s", uuid)
	}
	return volume, nil
}

func (d *ZStackDriver) GetZone(uuid string) (*view.ZoneView, error) {
	zone, err := d.client.GetZone(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query zone: %v", err)
	}
	return zone, nil
}

func (d *ZStackDriver) CreateVmInstance(vmInstance param.CreateVmInstanceParam) (*view.VmInstanceInventoryView, error) {
	log.Printf("creating...")
	vm, err := d.client.CreateVmInstance(vmInstance)
	log.Printf("fr")
	if err != nil {
		return nil, fmt.Errorf("failed to create VM instance: %v", err)
	}
	return vm, nil
}

func (d *ZStackDriver) StopVminstance(uuid string) (*view.VmInstanceInventoryView, error) {
	vmInstance, err := d.client.StopVmInstance(uuid, param.StopVmInstanceParam{
		StopVmInstance: param.StopVmInstanceDetailParam{
			Type:   param.Grace,
			StopHA: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to stop VM instance: %v", err)
	}
	return vmInstance, nil
}

func (d *ZStackDriver) DeleteVmInstance(uuid string) error {
	err := d.client.ExpungeVmInstance(uuid)
	if err != nil {
		return fmt.Errorf("failed to delete VM instance: %v", err)
	}
	return nil
}

func (d *ZStackDriver) CreateImage(rootVolumeParam param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageView, error) {
	img, err := d.client.CreateRootVolumeTemplateFromRootVolume(rootVolumeParam)
	if err != nil {
		return nil, fmt.Errorf("failed to create image: %v", err)
	}
	return &img, nil
}

func (d *ZStackDriver) ExportImage(image param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageResultView, error) {
	exportedImg, err := d.client.ExportImageFromBackupStorage(image)
	if err != nil {
		return nil, fmt.Errorf("failed to export image: %v", err)
	}
	return &exportedImg, nil
}

func (d *ZStackDriver) CreateDataVolume(volume param.CreateDataVolumeParam) (*view.VolumeView, error) {
	vol, err := d.client.CreateDataVolume(volume)
	if err != nil {
		return nil, fmt.Errorf("failed to create data volume: %v", err)
	}
	return vol, nil
}

func (d *ZStackDriver) AttachGuestToolsToVm(vmUuid string) error {
	err := d.client.AttachGuestToolsIsoToVm(vmUuid)

	if err != nil {
		return fmt.Errorf("failed to attach guest tools to VM: %v", err)
	}
	return nil
}

func (d *ZStackDriver) AttachDataVolumeToVm(vmUuid, volumeUuid string) (*view.VolumeView, error) {
	datavol, err := d.client.AttachDataVolumeToVm(volumeUuid, vmUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to attach data volume to VM: %v", err)
	}
	return datavol, nil
}

func (d *ZStackDriver) WaitForSSH(vmUuid string, sshPort int, timeout time.Duration) error {
	vm, err := d.GetVmInstance(vmUuid)
	if err != nil {
		return fmt.Errorf("failed to get VM instance: %v", err)
	}

	ip := vm.VMNics[0].IP
	if ip == "" {
		return fmt.Errorf("VM %s has no default IP", vmUuid)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, sshPort), 5*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timeout waiting for SSH on VM %s", vmUuid)
}

func addSystemTags(tags []string, args ...string) []string {
	tags = append(tags, args...)
	/*
		for _, v := range args {
			tags = append(tags, v)
		}
	*/
	return tags
}
