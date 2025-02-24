package zstack

import "fmt"

// packersdk.Artifact implementation
type Artifact struct {
	config    Config
	exportUrl []string
}

func (*Artifact) BuilderId() string {
	return BuilderId
}

func (a *Artifact) Files() []string {
	if len(a.exportUrl) > 0 {
		return a.exportUrl
	} else {
		return []string{}
	}

}

func (a *Artifact) Id() string {
	return a.config.ImageUuid
}

func (a *Artifact) String() string {
	return a.config.ImageUrl
}

func (a *Artifact) State(name string) interface{} {
	return fmt.Sprintf("State: name - %s", name)
}

func (a *Artifact) Destroy() error {
	return nil
}
