// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"crypto/sha512"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

const defaultClientLoginTimeout = 30 * time.Second

type AccessConfig struct {
	Host            string `mapstructure:"zstack_host"`
	Port            int    `mapstructure:"port"`
	AccountName     string `mapstructure:"account_name"`
	AccountPassword string `mapstructure:"account_password"`
	AccessKeyId     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	ctx             interpolate.Context

	portEnvErr error
}

func getEnvOrDefault(envVar string, defaultValue string) string {
	if defaultValue != "" {
		return defaultValue
	}
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return defaultValue
}

func (c *AccessConfig) applyEnvDefaults() {
	// Reset transient env-parse state so repeated Prepare() calls don't see
	// a stale error from a previous invocation with a bad ZSTACK_PORT.
	c.portEnvErr = nil
	c.Host = getEnvOrDefault("ZSTACK_HOST", c.Host)
	if raw := os.Getenv("ZSTACK_PORT"); raw != "" && c.Port == 0 {
		p, err := strconv.Atoi(raw)
		if err != nil {
			c.portEnvErr = fmt.Errorf("ZSTACK_PORT is not a valid integer: %q", raw)
		} else {
			c.Port = p
		}
	}
	c.AccountName = getEnvOrDefault("ZSTACK_ACCOUNT_NAME", c.AccountName)
	c.AccountPassword = getEnvOrDefault("ZSTACK_ACCOUNT_PASSWORD", c.AccountPassword)
	c.AccessKeyId = getEnvOrDefault("ZSTACK_ACCESS_KEY_ID", c.AccessKeyId)
	c.AccessKeySecret = getEnvOrDefault("ZSTACK_ACCESS_KEY_SECRET", c.AccessKeySecret)
}

func (c *AccessConfig) validateCredentials() []error {
	var errs []error
	if c.portEnvErr != nil {
		errs = append(errs, c.portEnvErr)
	}
	if c.Host == "" {
		errs = append(errs, fmt.Errorf("host is required"))
	}

	if (c.AccountName == "" || c.AccountPassword == "") && (c.AccessKeyId == "" || c.AccessKeySecret == "") {
		errs = append(errs, fmt.Errorf("either account_name + account_password or access_key_id + access_key_secret is required"))
	}

	if c.Port < 0 || c.Port > 65535 {
		errs = append(errs, fmt.Errorf("port must be between 1 and 65535"))
	}
	if (c.AccountName != "" || c.AccountPassword != "") && (c.AccessKeyId != "" || c.AccessKeySecret != "") {
		errs = append(errs, fmt.Errorf("account and access key authentication are mutually exclusive"))
	}
	return errs
}

func (c *AccessConfig) Prepare(raws ...interface{}) []error {
	c.ctx = interpolate.Context{}

	var errs []error
	err := config.Decode(c, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"run_command",
			},
		},
	}, raws...)
	if err != nil {
		errs = append(errs, err)
	}

	c.applyEnvDefaults()
	errs = append(errs, c.validateCredentials()...)

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (c *AccessConfig) CreateClient(debug bool) (*apiClient, error) {
	return c.createClient(context.Background())
}
func (c *AccessConfig) createClient(parent context.Context) (*apiClient, error) {
	if errs := c.validateCredentials(); len(errs) > 0 {
		return nil, errs[0]
	}
	cli, err := newAPIClient(c)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(parent, defaultClientLoginTimeout)
	defer cancel()
	if c.AccountName != "" && c.AccountPassword != "" {
		if _, err := cli.Login(ctx); err != nil {
			return nil, redactError(err, c.secretValues())
		}
		return cli, nil
	}
	d := &ZStackDriver{client: cli, ctx: ctx, secrets: c.secretValues()}
	if err := d.ValidateCredentials(); err != nil {
		return nil, err
	}
	return cli, nil
}

func (c *AccessConfig) secretValues() []string {
	values := []string{c.AccountPassword, c.AccessKeyId, c.AccessKeySecret}
	if c.AccountPassword != "" {
		values = append(values, fmt.Sprintf("%x", sha512.Sum512([]byte(c.AccountPassword))))
	}
	return values
}
func (c *AccessConfig) Driver(debug bool) (*ZStackDriver, error) {
	return c.driver(context.Background(), defaultAPITimeout)
}
func (c *AccessConfig) driver(ctx context.Context, timeout time.Duration) (*ZStackDriver, error) {
	cli, err := c.createClient(ctx)
	if err != nil {
		return nil, err
	}
	return &ZStackDriver{client: cli, ctx: ctx, timeout: timeout, secrets: append(c.secretValues(), cli.session)}, nil
}
