package zstack

import (
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"zstack.io/zstack-sdk-go/pkg/client"
)

type AccessConfig struct {
	Host            string `mapstructure:"zstack_host"`
	Port            int    `mapstructure:"port"`
	AccountName     string `mapstructure:"account_name"`
	AccountPassword string `mapstructure:"account_password"`
	AccessKeyId     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	ctx             interpolate.Context
}

func getEnvOrDefault(envVar string, defaultValue string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return defaultValue
}

func (c *AccessConfig) Prepare(raws ...interface{}) []error {
	// Initialize the context
	c.ctx = interpolate.Context{}

	// Decode the configuration
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

	c.Host = getEnvOrDefault("ZSTACK_HOST", c.Host)
	c.Port, _ = strconv.Atoi(getEnvOrDefault("ZSTACK_PORT", strconv.Itoa(c.Port)))
	c.AccountName = getEnvOrDefault("ZSTACK_ACCOUNTNAME", c.AccountName)
	c.AccountPassword = getEnvOrDefault("ZSTACK_ACCOUNTPASSWORD", c.AccountPassword)
	c.AccessKeyId = getEnvOrDefault("ZSTACK_ACCESSKEYID", c.AccessKeyId)
	c.AccessKeySecret = getEnvOrDefault("ZSTACK_ACCESSKEYSECRET", c.AccessKeySecret)

	// Validate required fields
	if c.Host == "" {
		errs = append(errs, fmt.Errorf("host is required"))
	}

	if (c.AccountName == "" || c.AccountPassword == "") && (c.AccessKeyId == "" || c.AccessKeySecret == "") {
		errs = append(errs, fmt.Errorf("either accountname + accountpassword or accesskeyid + accesskeysecret is required"))
	}
	if err != nil {

		return errs
	}
	return nil
}

func (c *AccessConfig) CreateClient() (*client.ZSClient, error) {
	var cli *client.ZSClient

	if c.AccountName != "" && c.AccountPassword != "" {
		cli = client.NewZSClient(client.NewZSConfig(c.Host, c.Port, "zstack").LoginAccount(c.AccountName, c.AccountPassword).ReadOnly(false).Debug(true))
		_, err := cli.Login()
		if err != nil {
			return nil, fmt.Errorf("unable to create ZStack API client: %s", err)
		}
	} else if c.AccessKeyId != "" && c.AccessKeySecret != "" {
		cli = client.NewZSClient(client.NewZSConfig(c.Host, c.Port, "zstack").AccessKey(c.AccessKeyId, c.AccessKeySecret).ReadOnly(false).Debug(true))
	}

	// Check if client was created successfully
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
