package zstack

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/client"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
)

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
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return defaultValue
}

func (c *AccessConfig) applyEnvDefaults() {
	c.Host = getEnvOrDefault("ZSTACK_HOST", c.Host)
	if raw := os.Getenv("ZSTACK_PORT"); raw != "" {
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

func (c *AccessConfig) CreateClient() (*client.ZSClient, error) {
	var cli *client.ZSClient

	// PKR-001 Bug 1: Only set default port when not specified (was overwriting custom port)
	if c.Port == 0 {
		c.Port = 8080
	}

	if c.AccountName != "" && c.AccountPassword != "" {
		cli = client.NewZSClient(client.NewZSConfig(c.Host, c.Port, "zstack").LoginAccount(c.AccountName, c.AccountPassword).ReadOnly(false).Debug(false))
		_, err := cli.Login(context.Background())
		if err != nil {
			return nil, fmt.Errorf("unable to create ZStack API client: %s", err)
		}
	} else if c.AccessKeyId != "" && c.AccessKeySecret != "" {
		cli = client.NewZSClient(client.NewZSConfig(c.Host, c.Port, "zstack").AccessKey(c.AccessKeyId, c.AccessKeySecret).ReadOnly(false).Debug(false))
		probe := param.NewQueryParam()
		if _, err := cli.QueryZone(&probe); err != nil {
			return nil, fmt.Errorf("unable to validate ZStack access key credentials: %s", err)
		}
	}

	if cli == nil {
		return nil, fmt.Errorf("failed to create ZStack client: client is nil")
	}

	return cli, nil
}

func (c *AccessConfig) Driver() (*ZStackDriver, error) {
	cli, err := c.CreateClient()
	if err != nil {
		return nil, err
	}

	return &ZStackDriver{client: cli}, nil
}
