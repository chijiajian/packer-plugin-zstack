package zstack

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArtifact_BuilderId(t *testing.T) {
	a := &Artifact{}
	assert.Equal(t, BuilderId, a.BuilderId())
}

func TestArtifact_Files(t *testing.T) {
	t.Run("EmptyWhenNoUrls", func(t *testing.T) {
		a := &Artifact{}
		assert.Equal(t, []string{}, a.Files())
	})
	t.Run("ReturnsExportUrls", func(t *testing.T) {
		a := &Artifact{exportUrl: []string{"http://x/a.qcow2", "http://x/b.qcow2"}}
		assert.Equal(t, []string{"http://x/a.qcow2", "http://x/b.qcow2"}, a.Files())
	})
}

func TestArtifact_Id(t *testing.T) {
	a := &Artifact{config: Config{ImageConfig: ImageConfig{ImageUuid: "uuid-1"}}}
	assert.Equal(t, "uuid-1", a.Id())
}

func TestArtifact_String(t *testing.T) {
	a := &Artifact{config: Config{ExportImageResult: ExportImageResult{ImageUrl: "http://x/a.qcow2"}}}
	assert.Equal(t, "http://x/a.qcow2", a.String())
}

func TestArtifact_State(t *testing.T) {
	a := &Artifact{}
	assert.Nil(t, a.State("anything"))
}

func TestArtifact_Destroy(t *testing.T) {
	t.Run("NoDriverReturnsNil", func(t *testing.T) {
		a := &Artifact{config: Config{ImageConfig: ImageConfig{ImageUuid: "u"}}}
		assert.NoError(t, a.Destroy())
	})
	t.Run("EmptyImageUuidReturnsNil", func(t *testing.T) {
		driver := &MockDriver{}
		a := &Artifact{driver: driver}
		assert.NoError(t, a.Destroy())
		assert.False(t, driver.DeleteImageCalled)
	})
	t.Run("SuccessDeletesAndExpunges", func(t *testing.T) {
		driver := &MockDriver{}
		a := &Artifact{
			driver: driver,
			config: Config{ImageConfig: ImageConfig{ImageUuid: "img-uuid"}},
		}
		assert.NoError(t, a.Destroy())
		assert.True(t, driver.DeleteImageCalled)
		assert.Equal(t, "img-uuid", driver.DeleteImageUuid)
		assert.True(t, driver.ExpungeImageCalled)
		assert.Equal(t, "img-uuid", driver.ExpungeImageUuid)
	})
	t.Run("DeleteImageErrorStopsBeforeExpunge", func(t *testing.T) {
		driver := &MockDriver{DeleteImageErr: errors.New("delete boom")}
		a := &Artifact{
			driver: driver,
			config: Config{ImageConfig: ImageConfig{ImageUuid: "img-uuid"}},
		}
		err := a.Destroy()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete image")
		assert.True(t, driver.DeleteImageCalled)
		assert.False(t, driver.ExpungeImageCalled)
	})
	t.Run("ExpungeImageErrorReturned", func(t *testing.T) {
		driver := &MockDriver{ExpungeImageErr: errors.New("expunge boom")}
		a := &Artifact{
			driver: driver,
			config: Config{ImageConfig: ImageConfig{ImageUuid: "img-uuid"}},
		}
		err := a.Destroy()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to expunge image")
		assert.True(t, driver.DeleteImageCalled)
		assert.True(t, driver.ExpungeImageCalled)
	})
}
