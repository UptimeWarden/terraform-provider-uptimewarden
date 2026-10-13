// Package provider implements the UptimeWarden Terraform provider using the
// Terraform Plugin Framework (v1.x). The provider is configured with an API
// token and an optional base_url override (useful for staging environments).
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

// Ensure uptimewardenProvider implements the provider.Provider interface.
var _ provider.Provider = &uptimewardenProvider{}
var _ provider.ProviderWithFunctions = &uptimewardenProvider{}

// uptimewardenProvider is the root provider struct.
type uptimewardenProvider struct {
	version string
}

// New returns a provider.Provider factory. Called by main.go.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &uptimewardenProvider{version: version}
	}
}

// providerModel maps the provider HCL block.
type providerModel struct {
	APIToken types.String `tfsdk:"api_token"`
	BaseURL  types.String `tfsdk:"base_url"`
}

func (p *uptimewardenProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "uptimewarden"
	resp.Version = p.version
}

func (p *uptimewardenProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The UptimeWarden provider manages UptimeWarden monitors as Terraform resources. It authenticates with an API token (`uptw_...`), which the UptimeWarden API accepts on its `/api/v1` monitor endpoints.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "UptimeWarden API token. Can also be set via the `UW_API_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the UptimeWarden API. Defaults to `https://app.uptimewarden.online`. Can also be set via the `UW_API_URL` environment variable (`UW_BASE_URL` is still read as an alias).",
				Optional:            true,
			},
		},
	}
}

func (p *uptimewardenProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// api_token: HCL → env var fallback.
	apiToken := cfg.APIToken.ValueString()
	if apiToken == "" {
		apiToken = os.Getenv("UW_API_TOKEN")
	}
	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API token",
			"Set api_token in the provider block or the UW_API_TOKEN environment variable.",
		)
		return
	}

	// base_url: HCL → UW_API_URL → UW_BASE_URL (alias) → default.
	baseURL := cfg.BaseURL.ValueString()
	if baseURL == "" {
		baseURL = baseURLFromEnv()
	}

	c := client.New(baseURL, apiToken)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *uptimewardenProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMonitorResource,
	}
}

func (p *uptimewardenProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewMonitorDataSource,
		NewRegionsDataSource,
		NewEscalationPolicyDataSource,
		NewWebhookDataSource,
	}
}

func (p *uptimewardenProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}

// baseURLFromEnv returns the API base URL from the environment. UW_API_URL is
// the canonical name (the CLI and GitHub Actions read the same variable);
// UW_BASE_URL is kept as an alias for configurations written before.
func baseURLFromEnv() string {
	if v := os.Getenv("UW_API_URL"); v != "" {
		return v
	}
	return os.Getenv("UW_BASE_URL")
}
