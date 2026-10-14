// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	vnc "github.com/mitchellh/go-vnc"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
)

func isoOperations(ctx context.Context, state multistep.StateBag) (isoDriver, error) {
	d, ok := stepDriver(ctx, state).(isoDriver)
	if !ok {
		return nil, fmt.Errorf("driver does not support ISO operations")
	}
	return d, nil
}

// StepAnswerISO publishes only the generated CD, never its temporary directory.
// The random URL is short-lived and is not included in the artifact or logs.
type StepAnswerISO struct {
	image    ownedResource
	server   *http.Server
	listener net.Listener
}

func (s *StepAnswerISO) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	state.Put("boot_iso_uuid", c.ImageUuid)
	path, _ := state.Get("cd_path").(string)
	if path == "" {
		return multistep.ActionContinue
	}
	// ZStack rejects image imports smaller than 1 MiB. ISO9660 permits
	// trailing zero padding, so keep small answer CDs above this threshold.
	if err := padAnswerISO(path); err != nil {
		return halt(state, err)
	}
	ip, err := isoHTTPIP(c)
	if err != nil {
		return halt(state, err)
	}
	s.listener, err = net.Listen("tcp", net.JoinHostPort(c.HTTPAddress, "0"))
	if err != nil {
		return halt(state, err)
	}
	s.image = newResource()
	token := newResource().UUID
	route := "/" + token + "/answer.iso"
	mux := http.NewServeMux()
	mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != route {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		f, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, "answer.iso", info.ModTime(), f)
	})
	s.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go s.server.Serve(s.listener)
	source := "http://" + net.JoinHostPort(ip, strconv.Itoa(s.listener.Addr().(*net.TCPAddr).Port)) + route
	d := stepDriver(ctx, state)
	img, err := d.AddImage(param.AddImageParam{Params: param.AddImageParamDetail{
		Name: "packer-answer-" + s.image.UUID, Url: source, Format: strPtr("iso"), MediaType: strPtr("ISO"), Platform: strPtr(c.Platform), Architecture: optionalString(c.Architecture), ResourceUuid: &s.image.UUID, BackupStorageUuids: []string{c.BackupStorageUuid},
	}})
	if err != nil {
		return halt(state, fmt.Errorf("import answer CD: %w", redactError(err, []string{source, token})))
	}
	if img == nil || img.UUID == "" {
		return halt(state, fmt.Errorf("answer CD import returned empty inventory"))
	}
	s.image.UUID = img.UUID
	s.image.Created = true
	readyCtx, cancel := context.WithTimeout(ctx, c.ImageReadyTimeout())
	defer cancel()
	d = stepDriver(readyCtx, state)
	for {
		img, err = d.GetImage(s.image.UUID)
		if err != nil {
			return halt(state, err)
		}
		if img.Status == "Ready" {
			break
		}
		if img.Status == "Failed" || img.Status == "Deleted" {
			return halt(state, fmt.Errorf("answer CD entered terminal status %s", img.Status))
		}
		select {
		case <-readyCtx.Done():
			return halt(state, readyCtx.Err())
		case <-time.After(c.PollInterval()):
		}
	}
	state.Put("answer_iso_uuid", s.image.UUID)
	return multistep.ActionContinue
}
func (s *StepAnswerISO) Cleanup(state multistep.StateBag) {
	if s.server != nil {
		s.server.Close()
	}
	cleanupImage(state, &s.image)
}
func padAnswerISO(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	const minSize = 1 << 20
	if info.Size() < minSize {
		return f.Truncate(minSize)
	}
	return nil
}

func isoHTTPIP(c *Config) (string, error) {
	if c.HTTPIP != "" {
		return c.HTTPIP, nil
	}
	if ip := net.ParseIP(c.HTTPAddress); ip != nil && !ip.IsUnspecified() {
		return ip.String(), nil
	}
	if c.HTTPInterface != "" {
		i, err := net.InterfaceByName(c.HTTPInterface)
		if err != nil {
			return "", err
		}
		addrs, err := i.Addrs()
		if err != nil {
			return "", err
		}
		for _, a := range addrs {
			if ip, n, e := net.ParseCIDR(a.String()); e == nil && n != nil && ip.To4() != nil && !ip.IsLoopback() {
				return ip.String(), nil
			}
		}
		return "", fmt.Errorf("http_interface has no usable IPv4 address")
	}
	client, err := newAPIClient(&c.AccessConfig)
	if err != nil {
		return "", err
	}
	conn, err := net.Dial("udp", net.JoinHostPort(client.base.Hostname(), "80"))
	if err != nil {
		return "", fmt.Errorf("set http_ip to an address reachable by the VM and backup storage: %w", err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

type StepISOBoot struct{}

func (*StepISOBoot) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	d, err := isoOperations(ctx, state)
	if err != nil {
		return halt(state, err)
	}
	if err = d.SetBootOrder(c.InstanceUuid, []string{"CdRom", "HardDisk"}); err != nil {
		return halt(state, err)
	}
	if _, err = d.StartVM(c.InstanceUuid); err != nil {
		return halt(state, err)
	}
	return multistep.ActionContinue
}
func (*StepISOBoot) Cleanup(multistep.StateBag) {}

type StepISOBootCommand struct{}

func (*StepISOBootCommand) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	if len(c.BootCommand) == 0 {
		return multistep.ActionContinue
	}
	ctx, cancel := context.WithTimeout(ctx, c.VmRunningTimeout())
	defer cancel()
	if c.BootWait > 0 {
		select {
		case <-ctx.Done():
			return halt(state, ctx.Err())
		case <-time.After(c.BootWait):
		}
	}
	ip, err := isoHTTPIP(c)
	if err != nil {
		return halt(state, err)
	}
	port, _ := state.Get("http_port").(int)
	tpl := c.ctx
	tpl.Data = &struct {
		HTTPIP   string
		HTTPPort int
		Name     string
	}{ip, port, c.InstanceName}
	bootCommand := c.FlatBootCommand()
	if c.BootGroupInterval > 0 {
		bootCommand = strings.Join(c.BootCommand, "<wait"+c.BootGroupInterval.String()+">")
	}
	command, err := interpolate.Render(bootCommand, &tpl)
	if err != nil {
		return halt(state, fmt.Errorf("render boot_command: %w", err))
	}
	seq, err := bootcommand.GenerateExpressionSequence(command)
	if err != nil {
		return halt(state, fmt.Errorf("invalid boot_command expression"))
	}
	d, err := isoOperations(ctx, state)
	if err != nil {
		return halt(state, err)
	}
	address, err := d.Console(c.InstanceUuid)
	if err != nil {
		return halt(state, err)
	}
	if address.Protocol != "vnc" || address.HostIP == "" || address.Port <= 0 {
		return halt(state, fmt.Errorf("ISO boot_command requires a reachable TCP VNC console"))
	}
	conn, err := (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(address.HostIP, strconv.Itoa(address.Port)))
	if err != nil {
		return halt(state, fmt.Errorf("connect VNC: %w", err))
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	auth := []vnc.ClientAuth{new(vnc.ClientAuthNone)}
	if address.Password != "" {
		auth = []vnc.ClientAuth{&vnc.PasswordAuth{Password: address.Password}}
	}
	client, err := vnc.Client(conn, &vnc.ClientConfig{Auth: auth})
	if err != nil {
		return halt(state, fmt.Errorf("VNC handshake failed"))
	}
	defer client.Close()
	state.Get("ui").(packersdk.Ui).Say("Sending ISO installer boot command...")
	if err = seq.Do(ctx, bootcommand.NewVNCDriver(client, c.BootKeyInterval)); err != nil {
		return halt(state, fmt.Errorf("send installer boot command: %w", err))
	}
	return multistep.ActionContinue
}
func (*StepISOBootCommand) Cleanup(multistep.StateBag) {}

type StepVerifyInstallation struct{}

func (*StepVerifyInstallation) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	comm, ok := state.Get("communicator").(packersdk.Communicator)
	if !ok {
		return halt(state, fmt.Errorf("installed system communicator is missing"))
	}
	cmd := &packersdk.RemoteCmd{Command: c.InstallationCheckCommand}
	if err := cmd.RunWithUi(ctx, comm, state.Get("ui").(packersdk.Ui)); err != nil {
		return halt(state, fmt.Errorf("installation check: %w", err))
	}
	if cmd.ExitStatus() != 0 {
		return halt(state, fmt.Errorf("installation_check_command failed with exit status %d", cmd.ExitStatus()))
	}
	return multistep.ActionContinue
}
func (*StepVerifyInstallation) Cleanup(multistep.StateBag) {}

type StepISOHardDiskBoot struct{ Connect *communicator.StepConnect }

func (s *StepISOHardDiskBoot) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	if s.Connect != nil {
		s.Connect.Cleanup(state)
	}
	if (&StepStopVmInstance{}).Run(ctx, state) != multistep.ActionContinue {
		return multistep.ActionHalt
	}
	d, err := isoOperations(ctx, state)
	if err != nil {
		return halt(state, err)
	}
	for _, key := range []string{"boot_iso_uuid", "answer_iso_uuid"} {
		if id, ok := state.Get(key).(string); ok && id != "" {
			if err = d.DetachISO(c.InstanceUuid, id); err != nil {
				return halt(state, err)
			}
		}
	}
	if err = d.SetBootOrder(c.InstanceUuid, []string{"HardDisk"}); err != nil {
		return halt(state, err)
	}
	if _, err = d.StartVM(c.InstanceUuid); err != nil {
		return halt(state, err)
	}
	return multistep.ActionContinue
}
func (*StepISOHardDiskBoot) Cleanup(multistep.StateBag) {}
