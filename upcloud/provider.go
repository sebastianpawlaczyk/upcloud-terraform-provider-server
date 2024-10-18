package upcloud

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/upcloud-terraform-provider-server/internal/server"

	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/client"
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	usernameDescription       = "UpCloud username with API access. Can also be configured using the `UPCLOUD_USERNAME` environment variable."
	passwordDescription       = "Password for UpCloud API user. Can also be configured using the `UPCLOUD_PASSWORD` environment variable."
	requestTimeoutDescription = "The duration (in seconds) that the provider waits for an HTTP request towards UpCloud API to complete. Defaults to 120 seconds"
)

type upcloudProviderModel struct {
	Username          types.String `tfsdk:"username"`
	Password          types.String `tfsdk:"password"`
	RetryWaitMinSec   types.Int64  `tfsdk:"retry_wait_min_sec"`
	RetryWaitMaxSec   types.Int64  `tfsdk:"retry_wait_max_sec"`
	RetryMax          types.Int64  `tfsdk:"retry_max"`
	RequestTimeoutSec types.Int64  `tfsdk:"request_timeout_sec"`
}

type upcloudProvider struct {
	userAgent string
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &upcloudProvider{
			userAgent: defaultUserAgent(),
		}
	}
}

func (p *upcloudProvider) Metadata(ctx context.Context, request provider.MetadataRequest, response *provider.MetadataResponse) {
	response.TypeName = "upcloud"
	response.Version = "0.1.0"
}

func (p *upcloudProvider) Schema(ctx context.Context, request provider.SchemaRequest, response *provider.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: usernameDescription,
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: passwordDescription,
				Optional:    true,
			},
			"retry_wait_min_sec": schema.Int64Attribute{
				Optional:    true,
				Description: "Minimum time to wait between retries",
			},
			"retry_wait_max_sec": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum time to wait between retries",
			},
			"retry_max": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of retries",
			},
			"request_timeout_sec": schema.Int64Attribute{
				Optional:    true,
				Description: requestTimeoutDescription,
			},
		},
	}
}

func withInt64Default(val types.Int64, def int64) int64 {
	if val.IsNull() {
		return def
	}
	return val.ValueInt64()
}

func withStringDefault(val types.String, def string) string {
	if val.IsNull() {
		return def
	}
	return val.ValueString()
}

func withEnvDefault(val types.String, env string) string {
	return withStringDefault(val, os.Getenv(env))
}

func (p *upcloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

	var model upcloudProviderModel
	if diags := req.Config.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics = append(resp.Diagnostics, diags...)
		return
	}

	requestTimeout := time.Duration(withInt64Default(model.RequestTimeoutSec, 120)) * time.Second
	config := Config{
		Username: withEnvDefault(model.Username, "UPCLOUD_USERNAME"),
		Password: withEnvDefault(model.Password, "UPCLOUD_PASSWORD"),
	}

	httpClient := retryablehttp.NewClient()
	httpClient.RetryWaitMin = time.Duration(withInt64Default(model.RetryWaitMinSec, 1)) * time.Second
	httpClient.RetryWaitMax = time.Duration(withInt64Default(model.RetryWaitMaxSec, 30)) * time.Second
	httpClient.RetryMax = int(withInt64Default(model.RetryMax, 4))

	service := newUpCloudServiceConnection(
		config.Username,
		config.Password,
		httpClient.HTTPClient,
		requestTimeout,
		p.userAgent,
	)

	_, err := config.checkLogin(service)
	if err != nil {
		resp.Diagnostics.AddError("Authentication failed", "Failed to authenticate to UpCloud API with given credentials")
	}

	tflog.Info(ctx, "UpCloud service connection configured for plugin framework provider", map[string]interface{}{"http_client": fmt.Sprintf("%#v", httpClient), "request_timeout": requestTimeout})

	resp.ResourceData = service
	resp.DataSourceData = service
}

func (p *upcloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *upcloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		server.NewServerResource,
	}
}

func newUpCloudServiceConnection(username, password string, httpClient *http.Client, requestTimeout time.Duration, userAgents ...string) *service.Service {
	providerClient := client.New(
		username,
		password,
		client.WithHTTPClient(httpClient),
		client.WithTimeout(requestTimeout),
	)

	if len(userAgents) == 0 {
		userAgents = []string{defaultUserAgent()}
	}
	providerClient.UserAgent = strings.Join(userAgents, " ")

	return service.New(providerClient)
}

func defaultUserAgent() string {
	return fmt.Sprintf("upcloud-terraform-provider-server/%s", "dev")
}
