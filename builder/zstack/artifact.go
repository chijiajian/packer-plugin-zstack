package zstack

import "fmt"

// packersdk.Artifact implementation
type Artifact struct {
	config    Config
	exportUrl []string
	driver    Driver
}

func (*Artifact) BuilderId() string {
	return BuilderId
}

func (a *Artifact) Files() []string {
	if len(a.exportUrl) > 0 {
		return a.exportUrl
	}
	return []string{}
}

func (a *Artifact) Id() string {
	return a.config.ImageUuid
}

func (a *Artifact) String() string {
	return a.config.ImageUrl
}

func (a *Artifact) State(name string) interface{} {
	return nil
}

func (a *Artifact) Destroy() error {
	if a.driver == nil || a.config.ImageUuid == "" {
		return nil
	}
	if err := a.driver.DeleteImage(a.config.ImageUuid); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	if err := a.driver.ExpungeImage(a.config.ImageUuid); err != nil {
		return fmt.Errorf("failed to expunge image: %w", err)
	}
	return nil
}
