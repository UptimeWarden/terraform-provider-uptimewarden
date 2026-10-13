package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

// Ensure uptimewardenMonitorDataSource implements datasource.DataSource.
var _ datasource.DataSource = &uptimewardenMonitorDataSource{}

// NewMonitorDataSource is registered in provider.go.
func NewMonitorDataSource() datasource.DataSource {
	return &uptimewardenMonitorDataSource{}
}

// uptimewardenMonitorDataSource reads an existing monitor by publicId.
type uptimewardenMonitorDataSource struct {
	client *client.Client
}

// monitorDataSourceModel maps the data source HCL block.
type monitorDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	URL      types.String `tfsdk:"url"`
	Interval types.Int64  `tfsdk:"interval"`
	Timeout  types.Int64  `tfsdk:"timeout"`
	Region   types.String `tfsdk:"region"`
	IsActive types.Bool   `tfsdk:"is_active"`
}

func (d *uptimewardenMonitorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (d *uptimewardenMonitorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads an existing UptimeWarden monitor by its public ID. Use this data source to reference a monitor managed outside of Terraform (e.g. created via the UI or API).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Public ID of the monitor to look up.",
				Required:            true,
			},
			"name":      schema.StringAttribute{MarkdownDescription: "Monitor name.", Computed: true},
			"type":      schema.StringAttribute{MarkdownDescription: "Monitor protocol type.", Computed: true},
			"url":       schema.StringAttribute{MarkdownDescription: "Target URL or address.", Computed: true},
			"interval":  schema.Int64Attribute{MarkdownDescription: "Check interval in seconds.", Computed: true},
			"timeout":   schema.Int64Attribute{MarkdownDescription: "Request timeout in seconds.", Computed: true},
			"region":    schema.StringAttribute{MarkdownDescription: "Home region code (`ae`, `brs`, `gwc`, `sea` or `wus3`). Null when the monitor is placed automatically.", Computed: true},
			"is_active": schema.BoolAttribute{MarkdownDescription: "Whether the monitor is actively checking.", Computed: true},
		},
	}
}

func (d *uptimewardenMonitorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *uptimewardenMonitorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state monitorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m, err := d.client.GetMonitor(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading monitor", err.Error())
		return
	}

	state.Name = types.StringValue(m.Name)
	state.Type = types.StringValue(m.Type)
	state.URL = types.StringValue(m.URL)
	state.Interval = types.Int64Value(int64(m.Interval))
	state.Timeout = types.Int64Value(int64(m.Timeout))
	state.Region = regionValue(m.Region)
	state.IsActive = types.BoolValue(m.IsActive)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
