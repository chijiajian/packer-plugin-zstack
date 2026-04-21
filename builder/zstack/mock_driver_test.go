package zstack

import (
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

type MockDriver struct {
	GetBackupStorageResult *view.BackupStorageInventoryView
	GetBackupStorageErr    error
	GetBackupStorageCalled bool
	GetBackupStorageUuid   string

	QueryBackStorageResult []view.BackupStorageInventoryView
	QueryBackStorageErr    error
	QueryBackStorageCalled bool
	QueryBackStorageName   string

	GetImageResult *view.ImageInventoryView
	GetImageErr    error
	GetImageCalled bool
	GetImageUuid   string

	QueryImageResult []view.ImageInventoryView
	QueryImageErr    error
	QueryImageCalled bool
	QueryImageName   string

	GetVmInstanceResult *view.VmInstanceInventoryView
	GetVmInstanceErr    error
	GetVmInstanceCalled bool
	GetVmInstanceUuid   string

	GetL3NetworkResult *view.L3NetworkInventoryView
	GetL3NetworkErr    error
	GetL3NetworkCalled bool
	GetL3NetworkUuid   string

	QueryL3NetworkResult []view.L3NetworkInventoryView
	QueryL3NetworkErr    error
	QueryL3NetworkCalled bool
	QueryL3NetworkName   string

	GetInstanceOfferingResult *view.InstanceOfferingInventoryView
	GetInstanceOfferingErr    error
	GetInstanceOfferingCalled bool
	GetInstanceOfferingUuid   string

	QueryInstanceOfferingResult []view.InstanceOfferingInventoryView
	QueryInstanceOfferingErr    error
	QueryInstanceOfferingCalled bool
	QueryInstanceOfferingName   string

	GetVolumeResult *view.VolumeInventoryView
	GetVolumeErr    error
	GetVolumeCalled bool
	GetVolumeUuid   string

	GetZoneResult *view.ZoneInventoryView
	GetZoneErr    error
	GetZoneCalled bool
	GetZoneUuid   string

	CreateVmInstanceResult *view.VmInstanceInventoryView
	CreateVmInstanceErr    error
	CreateVmInstanceCalled bool
	CreateVmInstanceParam  param.CreateVmInstanceParam

	StopVminstanceResult *view.VmInstanceInventoryView
	StopVminstanceErr    error
	StopVminstanceCalled bool
	StopVminstanceUuid   string

	DestroyVmInstanceErr    error
	DestroyVmInstanceCalled bool
	DestroyVmInstanceUuid   string

	DeleteVmInstanceErr    error
	DeleteVmInstanceCalled bool
	DeleteVmInstanceUuid   string

	CreateImageResult         *view.ImageInventoryView
	CreateImageErr            error
	CreateImageCalled         bool
	CreateImageRootVolumeUuid string
	CreateImageParam          param.CreateRootVolumeTemplateFromRootVolumeParam

	AddImageResult *view.ImageInventoryView
	AddImageErr    error
	AddImageCalled bool
	AddImageParam  param.AddImageParam

	DeleteImageErr    error
	DeleteImageCalled bool
	DeleteImageUuid   string

	ExpungeImageErr    error
	ExpungeImageCalled bool
	ExpungeImageUuid   string

	ValidateCredentialsErr    error
	ValidateCredentialsCalled bool

	CreateDataVolumeResult *view.VolumeInventoryView
	CreateDataVolumeErr    error
	CreateDataVolumeCalled bool
	CreateDataVolumeParam  param.CreateDataVolumeParam

	ExportImageResult            *view.ExportImageFromBackupStorageEventView
	ExportImageErr               error
	ExportImageCalled            bool
	ExportImageBackupStorageUuid string
	ExportImageParam             param.ExportImageFromBackupStorageParam

	AttachGuestToolsErr    error
	AttachGuestToolsCalled bool
	AttachGuestToolsVmUuid string
	AttachDataVolumeResult *view.VolumeInventoryView
	AttachDataVolumeErr    error
	AttachDataVolumeCalled bool
	AttachDataVolumeVmUuid string
	AttachDataVolumeUuid   string
}

func (m *MockDriver) GetBackupStorage(uuid string) (*view.BackupStorageInventoryView, error) {
	m.GetBackupStorageCalled = true
	m.GetBackupStorageUuid = uuid
	return m.GetBackupStorageResult, m.GetBackupStorageErr
}
func (m *MockDriver) QueryBackStorage(name string) ([]view.BackupStorageInventoryView, error) {
	m.QueryBackStorageCalled = true
	m.QueryBackStorageName = name
	return m.QueryBackStorageResult, m.QueryBackStorageErr
}
func (m *MockDriver) GetImage(uuid string) (*view.ImageInventoryView, error) {
	m.GetImageCalled = true
	m.GetImageUuid = uuid
	return m.GetImageResult, m.GetImageErr
}
func (m *MockDriver) QueryImage(name string) ([]view.ImageInventoryView, error) {
	m.QueryImageCalled = true
	m.QueryImageName = name
	return m.QueryImageResult, m.QueryImageErr
}
func (m *MockDriver) GetVmInstance(uuid string) (*view.VmInstanceInventoryView, error) {
	m.GetVmInstanceCalled = true
	m.GetVmInstanceUuid = uuid
	return m.GetVmInstanceResult, m.GetVmInstanceErr
}
func (m *MockDriver) GetL3Network(uuid string) (*view.L3NetworkInventoryView, error) {
	m.GetL3NetworkCalled = true
	m.GetL3NetworkUuid = uuid
	return m.GetL3NetworkResult, m.GetL3NetworkErr
}
func (m *MockDriver) QueryL3Network(name string) ([]view.L3NetworkInventoryView, error) {
	m.QueryL3NetworkCalled = true
	m.QueryL3NetworkName = name
	return m.QueryL3NetworkResult, m.QueryL3NetworkErr
}
func (m *MockDriver) GetInstanceOffering(uuid string) (*view.InstanceOfferingInventoryView, error) {
	m.GetInstanceOfferingCalled = true
	m.GetInstanceOfferingUuid = uuid
	return m.GetInstanceOfferingResult, m.GetInstanceOfferingErr
}
func (m *MockDriver) QueryInstanceOffering(name string) ([]view.InstanceOfferingInventoryView, error) {
	m.QueryInstanceOfferingCalled = true
	m.QueryInstanceOfferingName = name
	return m.QueryInstanceOfferingResult, m.QueryInstanceOfferingErr
}
func (m *MockDriver) GetVolume(uuid string) (*view.VolumeInventoryView, error) {
	m.GetVolumeCalled = true
	m.GetVolumeUuid = uuid
	return m.GetVolumeResult, m.GetVolumeErr
}
func (m *MockDriver) GetZone(uuid string) (*view.ZoneInventoryView, error) {
	m.GetZoneCalled = true
	m.GetZoneUuid = uuid
	return m.GetZoneResult, m.GetZoneErr
}
func (m *MockDriver) CreateVmInstance(p param.CreateVmInstanceParam) (*view.VmInstanceInventoryView, error) {
	m.CreateVmInstanceCalled = true
	m.CreateVmInstanceParam = p
	return m.CreateVmInstanceResult, m.CreateVmInstanceErr
}
func (m *MockDriver) StopVminstance(uuid string) (*view.VmInstanceInventoryView, error) {
	m.StopVminstanceCalled = true
	m.StopVminstanceUuid = uuid
	return m.StopVminstanceResult, m.StopVminstanceErr
}
func (m *MockDriver) DestroyVmInstance(uuid string) error {
	m.DestroyVmInstanceCalled = true
	m.DestroyVmInstanceUuid = uuid
	return m.DestroyVmInstanceErr
}
func (m *MockDriver) DeleteVmInstance(uuid string) error {
	m.DeleteVmInstanceCalled = true
	m.DeleteVmInstanceUuid = uuid
	return m.DeleteVmInstanceErr
}
func (m *MockDriver) CreateImage(rootVolumeUuid string, params param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageInventoryView, error) {
	m.CreateImageCalled = true
	m.CreateImageRootVolumeUuid = rootVolumeUuid
	m.CreateImageParam = params
	return m.CreateImageResult, m.CreateImageErr
}
func (m *MockDriver) AddImage(p param.AddImageParam) (*view.ImageInventoryView, error) {
	m.AddImageCalled = true
	m.AddImageParam = p
	return m.AddImageResult, m.AddImageErr
}
func (m *MockDriver) DeleteImage(uuid string) error {
	m.DeleteImageCalled = true
	m.DeleteImageUuid = uuid
	return m.DeleteImageErr
}
func (m *MockDriver) ExpungeImage(uuid string) error {
	m.ExpungeImageCalled = true
	m.ExpungeImageUuid = uuid
	return m.ExpungeImageErr
}
func (m *MockDriver) ValidateCredentials() error {
	m.ValidateCredentialsCalled = true
	return m.ValidateCredentialsErr
}
func (m *MockDriver) CreateDataVolume(volume param.CreateDataVolumeParam) (*view.VolumeInventoryView, error) {
	m.CreateDataVolumeCalled = true
	m.CreateDataVolumeParam = volume
	return m.CreateDataVolumeResult, m.CreateDataVolumeErr
}
func (m *MockDriver) ExportImage(backupStorageUuid string, params param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageEventView, error) {
	m.ExportImageCalled = true
	m.ExportImageBackupStorageUuid = backupStorageUuid
	m.ExportImageParam = params
	return m.ExportImageResult, m.ExportImageErr
}
func (m *MockDriver) AttachGuestToolsToVm(vmUuid string) error {
	m.AttachGuestToolsCalled = true
	m.AttachGuestToolsVmUuid = vmUuid
	return m.AttachGuestToolsErr
}
func (m *MockDriver) AttachDataVolumeToVm(vmUuid, volumeUuid string) (*view.VolumeInventoryView, error) {
	m.AttachDataVolumeCalled = true
	m.AttachDataVolumeVmUuid = vmUuid
	m.AttachDataVolumeUuid = volumeUuid
	return m.AttachDataVolumeResult, m.AttachDataVolumeErr
}

func testStateBag(config *Config, driver Driver) multistep.StateBag {
	state := new(multistep.BasicStateBag)
	state.Put("config", config)
	state.Put("driver", driver)
	state.Put("ui", &packersdk.MockUi{})
	return state
}
