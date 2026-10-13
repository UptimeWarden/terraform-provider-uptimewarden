package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

// Ensure uptimewardenMonitorResource implements resource.Resource and
// resource.ResourceWithImportState.
var _ resource.Resource = &uptimewardenMonitorResource{}
var _ resource.ResourceWithImportState = &uptimewardenMonitorResource{}

// NewMonitorResource is registered in provider.go.
func NewMonitorResource() resource.Resource {
	return &uptimewardenMonitorResource{}
}

// uptimewardenMonitorResource implements CRUD for uptimewarden_monitor.
type uptimewardenMonitorResource struct {
	client *client.Client
}

// monitorResourceModel maps the resource HCL block.
type monitorResourceModel struct {
	// id is the UptimeWarden publicId (ULID). Computed on Create, stable after.
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	URL      types.String `tfsdk:"url"`
	Interval types.Int64  `tfsdk:"interval"`
	Timeout  types.Int64  `tfsdk:"timeout"`
	Region   types.String `tfsdk:"region"`
	IsActive types.Bool   `tfsdk:"is_active"`
}

// validMonitorTypes mirrors allowedMonitorTypes in the Go API handler.
var validMonitorTypes = []string{
	"http", "tcp", "udp", "ssl", "websocket", "grpc", "mcp", "ping", "dns", "heartbeat",
}

func (r *uptimewardenMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *uptimewardenMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an UptimeWarden monitor. Monitors are the core resource — they define what to check, how often, and from which regions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Public ID of the monitor (ULID). Assigned by UptimeWarden on creation.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the monitor.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Protocol type. One of: `http`, `tcp`, `udp`, `ssl`, `websocket`, `grpc`, `mcp`, `ping`, `dns`, `heartbeat`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// Type changes require replacement — the monitor's check
					// behaviour is fundamentally different between types.
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(validMonitorTypes...),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "Target URL or address (e.g. `https://api.acme.com/health`, `tcp://db.example.com:5432`).",
				Required:            true,
			},
			"interval": schema.Int64Attribute{
				MarkdownDescription: "Check interval in seconds. Minimum: 30 (Enterprise). Default: 300.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(300),
				Validators: []validator.Int64{
					int64validator.Between(30, 86400),
				},
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Request timeout in seconds (5–300). Default: 30.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(30),
				Validators: []validator.Int64{
					int64validator.Between(5, 300),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Primary check region. One of: `gwc` (Frankfurt), `wus3` (Phoenix), `sea` (Singapore), `brs` (São Paulo), `ae` (Australia East), `uaen` (UAE North). Defaults to `gwc`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("gwc"),
			},
			"is_active": schema.BoolAttribute{
				MarkdownDescription: "Whether the monitor is actively checking. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

func (r *uptimewardenMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = c
}

func (r *uptimewardenMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	isActive := plan.IsActive.ValueBool()
	created, err := r.client.CreateMonitor(ctx, client.CreateMonitorRequest{
		Name:     plan.Name.ValueString(),
		Type:     plan.Type.ValueString(),
		URL:      plan.URL.ValueString(),
		Interval: int(plan.Interval.ValueInt64()),
		Timeout:  int(plan.Timeout.ValueInt64()),
		Region:   plan.Region.ValueString(),
		IsActive: &isActive,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor", err.Error())
		return
	}

	tflog.Trace(ctx, "created monitor", map[string]any{"id": created.PublicID})
	monitorToState(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *uptimewardenMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	m, err := r.client.GetMonitor(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFoundErr(err) {
			// Monitor was deleted outside Terraform — remove from state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading monitor", err.Error())
		return
	}

	monitorToState(m, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *uptimewardenMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan monitorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	isActive := plan.IsActive.ValueBool()
	updated, err := r.client.UpdateMonitor(ctx, plan.ID.ValueString(), client.UpdateMonitorRequest{
		Name:     plan.Name.ValueString(),
		URL:      plan.URL.ValueString(),
		Interval: int(plan.Interval.ValueInt64()),
		Timeout:  int(plan.Timeout.ValueInt64()),
		Region:   plan.Region.ValueString(),
		IsActive: &isActive,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating monitor", err.Error())
		return
	}

	monitorToState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *uptimewardenMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteMonitor(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFoundErr(err) {
			return // already gone
		}
		resp.Diagnostics.AddError("Error deleting monitor", err.Error())
		return
	}

	tflog.Trace(ctx, "deleted monitor", map[string]any{"id": state.ID.ValueString()})
}

// ImportState supports `terraform import uptimewarden_monitor.example <publicId>`.
func (r *uptimewardenMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	m, err := r.client.GetMonitor(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing monitor", err.Error())
		return
	}

	var state monitorResourceModel
	monitorToState(m, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// monitorToState maps an API Monitor to the Terraform state model.
func monitorToState(m *client.Monitor, s *monitorResourceModel) {
	s.ID = types.StringValue(m.PublicID)
	s.Name = types.StringValue(m.Name)
	s.Type = types.StringValue(m.Type)
	s.URL = types.StringValue(m.URL)
	s.Interval = types.Int64Value(int64(m.Interval))
	s.Timeout = types.Int64Value(int64(m.Timeout))
	s.Region = types.StringValue(m.Region)
	s.IsActive = types.BoolValue(m.IsActive)
}
