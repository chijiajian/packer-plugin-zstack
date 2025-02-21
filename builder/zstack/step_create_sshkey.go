package zstack

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"golang.org/x/crypto/ssh"
)

type StepCreateSSHKey struct {
	Password     string
	Publicfile   string
	Debug        bool
	DebugKeyPath string
}

func (s *StepCreateSSHKey) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	//driver := state.Get("driver").(Driver)
	ui.Say("start create sshkey for zstack")

	if s.Password != "" {
		ui.Message("Using SSH password")
		return multistep.ActionContinue
	}

	if config.Comm.SSHPrivateKeyFile != "" {
		ui.Message("Using existing SSH private key")
		privateKeyBytes, err := os.ReadFile(config.Comm.SSHPrivateKeyFile)
		if err != nil {
			state.Put("error", fmt.Errorf("error loading configured private key file: %s", err))
			return multistep.ActionHalt
		}
		config.Comm.SSHPrivateKey = privateKeyBytes
		state.Put("config", config)
		return multistep.ActionContinue
	}

	ui.Say("Createing temporary SSH key for instance...")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		err := fmt.Errorf("error creating temporary ssh key: %s", err)
		state.Put("error", err)
		ui.Errorf(err.Error())
		return multistep.ActionHalt
	}

	privateKeyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	}

	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		err := fmt.Errorf("error creating temporary ssh key: %s", err)
		state.Put("error", err)
		ui.Errorf(err.Error())
		return multistep.ActionHalt
	}

	config.Comm.SSHPrivateKey = pem.EncodeToMemory(&privateKeyBlock)
	config.Comm.SSHPublicKey = ssh.MarshalAuthorizedKey(pub)

	if config.Comm.SSHPublicKey[len(config.Comm.SSHPublicKey)-1] == '\n' {
		config.Comm.SSHPublicKey = config.Comm.SSHPublicKey[:len(config.Comm.SSHPublicKey)-1]
	}
	state.Put("config", config)

	if s.Debug {
		ui.Message(fmt.Sprintf("Saving key for debug purposes: %s", s.DebugKeyPath))
		f, err := os.Create(s.DebugKeyPath)
		if err != nil {
			state.Put("error", fmt.Errorf("error saving debug key: %s", err))
			return multistep.ActionHalt
		}

		err = pem.Encode(f, &privateKeyBlock)
		f.Close()
		if err != nil {
			state.Put("error", fmt.Errorf("error saving debug key: %s", err))
			return multistep.ActionHalt
		}
	}
	return multistep.ActionContinue
}
