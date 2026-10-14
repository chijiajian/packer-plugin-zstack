// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"context"
	"fmt"
)

type Artifact struct {
	config        Config
	driver        Driver
	owned         bool
	deleted       bool
	removed       bool
	warnings      []string
	exportRemoved bool
}

func (*Artifact) BuilderId() string { return BuilderId }

// A cloud image is a remote resource; URLs are metadata, not local files.
func (*Artifact) Files() []string { return []string{} }
func (a *Artifact) Id() string    { return a.config.ImageUuid }
func (a *Artifact) String() string {
	return fmt.Sprintf("ZStack image: %s", a.Id())
}
func (a *Artifact) State(name string) interface{} {
	switch name {
	case "image_uuid":
		return a.Id()
	case "image_url":
		return a.config.ImageUrl
	case "backup_storage_uuid":
		return a.config.BackupStorageUuid
	case "cleanup_warnings":
		return append([]string(nil), a.warnings...)
	}
	return nil
}
func (a *Artifact) Destroy() error {
	if !a.owned || a.removed || a.driver == nil || a.Id() == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), a.config.CleanupTimeout())
	defer cancel()
	d := a.driver.WithContext(ctx)
	if a.config.ImageUrl != "" && !a.exportRemoved {
		if err := d.DeleteExportedImage(a.config.BackupStorageUuid, a.Id()); err != nil {
			return fmt.Errorf("failed to delete exported image: %w", err)
		}
		a.exportRemoved = true
	}
	if !a.deleted {
		if err := d.DeleteImage(a.Id()); err != nil {
			return fmt.Errorf("failed to delete image: %w", err)
		}
		a.deleted = true
	}
	if err := d.ExpungeImage(a.Id()); err != nil {
		return fmt.Errorf("failed to expunge image: %w", err)
	}
	a.removed = true
	return nil
}
