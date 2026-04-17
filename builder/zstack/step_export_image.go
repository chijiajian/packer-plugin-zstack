package zstack

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
)

type StepExportImage struct {
}

func (s *StepExportImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	// AC-003-01/03: Skip export when no backup storage configured
	if config.BackupStorageUuid == "" {
		log.Printf("[INFO] Skipping image export: no backup storage configured")
		ui.Say("Skipping image export: no backup storage configured")
		return multistep.ActionContinue
	}

	if config.ImageUuid == "" {
		err := fmt.Errorf("image UUID is empty, cannot export")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say("Exporting image to backup storage...")

	exportImageParam := param.ExportImageFromBackupStorageParam{
		Params: param.ExportImageFromBackupStorageParamDetail{
			ImageUuid: config.ImageUuid,
		},
	}

	exportImageResult, err := driver.ExportImage(config.BackupStorageUuid, exportImageParam)
	if err != nil {
		// Some backup storage types don't implement export API; skip instead of failing the whole build.
		if isUnsupportedExportError(err) {
			msg := fmt.Sprintf("Skipping image export: backup storage may not support export (%v)", err)
			log.Printf("[WARN] %s", msg)
			ui.Say(msg)
			return multistep.ActionContinue
		}
		ui.Error("Failed to export image: " + err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	config.ImageUrl = exportImageResult.ImageUrl
	state.Put("config", config)
	ui.Say("Successfully exported image: " + config.ImageUrl)
	return multistep.ActionContinue
}

func (s *StepExportImage) Cleanup(state multistep.StateBag) {}

func isUnsupportedExportError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "apiexportimagefrombackupstoragemsg") ||
		strings.Contains(msg, "no service deals with message") ||
		strings.Contains(msg, "not support") ||
		strings.Contains(msg, "unsupported")
}
