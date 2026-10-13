package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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

// validRegions are the region codes the API accepts for new monitors and
// region changes. Display names follow the regions table the dashboard reads:
// ae Sydney, brs São Paulo, gwc Sweden Central, sea East Asia, wus3 Phoenix.
// The retired UAE North region is not listed: the API refuses it.
var validRegions = []string{"ae", "brs", "gwc", "sea", "wus3"}

// regionDescription is shared by the resource and data source schemas.
const regionDescription = "Home region that runs the checks. One of: `ae` (Sydney), `brs` (São Paulo), `gwc` (Sweden Central), `sea` (East Asia), `wus3` (Phoenix). Choosing a region needs a plan that allows it (Professional or Enterprise); when unset, UptimeWarden places the monitor automatically (`gwc`) and the attribute reads back what the API reports."

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
				MarkdownDescription: "Request timeout in seconds (5–300). When unset, the API default (30) applies and is read back into state. The API applies a timeout on update but not yet on create, so for a configured value other than 30 the provider sets it with an update right after the create.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.Int64{
					int64validator.Between(5, 300),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: regionDescription,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					// An unset region keeps whatever the API reported; the
					// API only changes a region when a request names one.
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(validRegions...),
				},
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
	createReq := client.CreateMonitorRequest{
		Name:     plan.Name.ValueString(),
		Type:     plan.Type.ValueString(),
		URL:      plan.URL.ValueString(),
		Interval: int(plan.Interval.ValueInt64()),
		IsActive: &isActive,
	}
	if wantTimeout, ok := knownInt64(plan.Timeout); ok {
		createReq.Timeout = int(wantTimeout)
	}
	if wantRegion, ok := knownString(plan.Region); ok {
		createReq.Region = wantRegion
	}
	created, err := r.client.CreateMonitor(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor", err.Error())
		return
	}
	tflog.Trace(ctx, "created monitor", map[string]any{"id": created.PublicID})

	// The API's create path applies its default timeout (30) whatever the
	// request says, while PATCH applies the value. Set a configured timeout
	// with a follow-up update so state matches the configuration.
	if wantTimeout, ok := knownInt64(plan.Timeout); ok && int64(created.Timeout) != wantTimeout {
		t := int(wantTimeout)
		updated, uErr := r.client.UpdateMonitor(ctx, created.PublicID, client.UpdateMonitorRequest{Timeout: &t})
		if uErr != nil {
			// The monitor exists: keep it in state (as created) so the next
			// apply retries the timeout instead of creating a duplicate.
			monitorToState(created, &plan)
			resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
			resp.Diagnostics.AddError("Error setting monitor timeout",
				fmt.Sprintf("The monitor was created (id %s) but setting timeout = %d failed: %s", created.PublicID, wantTimeout, uErr))
			return
		}
		created = updated
	}

	want := plan
	monitorToState(created, &plan)
	if diags := checkApplied(want, created); diags != "" {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("UptimeWarden did not apply the configuration", diags)
		return
	}
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
	updateReq := client.UpdateMonitorRequest{
		Name:     plan.Name.ValueString(),
		URL:      plan.URL.ValueString(),
		Interval: int(plan.Interval.ValueInt64()),
		IsActive: &isActive,
	}
	if wantTimeout, ok := knownInt64(plan.Timeout); ok {
		t := int(wantTimeout)
		updateReq.Timeout = &t
	}
	if wantRegion, ok := knownString(plan.Region); ok {
		updateReq.Region = &wantRegion
	}
	updated, err := r.client.UpdateMonitor(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating monitor", err.Error())
		return
	}

	want := plan
	monitorToState(updated, &plan)
	if diags := checkApplied(want, updated); diags != "" {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("UptimeWarden did not apply the configuration", diags)
		return
	}
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

// monitorToState maps an API Monitor to the Terraform state model. State is
// always what the API reports, never the request, so a value the API did not
// apply shows up instead of being hidden.
func monitorToState(m *client.Monitor, s *monitorResourceModel) {
	s.ID = types.StringValue(m.PublicID)
	s.Name = types.StringValue(m.Name)
	s.Type = types.StringValue(m.Type)
	s.URL = types.StringValue(m.URL)
	s.Interval = types.Int64Value(int64(m.Interval))
	s.Timeout = types.Int64Value(int64(m.Timeout))
	s.Region = regionValue(m.Region)
	s.IsActive = types.BoolValue(m.IsActive)
}

// regionValue maps the API's region to state. The API omits the field for a
// monitor placed automatically; that reads as null.
func regionValue(region string) types.String {
	if region == "" {
		return types.StringNull()
	}
	return types.StringValue(region)
}

func knownInt64(v types.Int64) (int64, bool) {
	if v.IsNull() || v.IsUnknown() {
		return 0, false
	}
	return v.ValueInt64(), true
}

func knownString(v types.String) (string, bool) {
	if v.IsNull() || v.IsUnknown() {
		return "", false
	}
	return v.ValueString(), true
}

// checkApplied compares the planned values that were known before the call
// with what the API returned. Terraform requires a configured value to come
// back unchanged; when the API silently ignores one, the generic "Provider
// produced inconsistent result after apply" error hides which field it was.
// This names the field and both values instead. Returns "" when all match.
func checkApplied(want monitorResourceModel, got *client.Monitor) string {
	var diffs []string
	if v, ok := knownString(want.Name); ok && v != got.Name {
		diffs = append(diffs, fmt.Sprintf("name: configured %q, API reports %q", v, got.Name))
	}
	if v, ok := knownString(want.URL); ok && v != got.URL {
		diffs = append(diffs, fmt.Sprintf("url: configured %q, API reports %q", v, got.URL))
	}
	if v, ok := knownInt64(want.Interval); ok && v != int64(got.Interval) {
		diffs = append(diffs, fmt.Sprintf("interval: configured %d, API reports %d", v, got.Interval))
	}
	if v, ok := knownInt64(want.Timeout); ok && v != int64(got.Timeout) {
		diffs = append(diffs, fmt.Sprintf("timeout: configured %d, API reports %d", v, got.Timeout))
	}
	if v, ok := knownString(want.Region); ok && v != got.Region {
		diffs = append(diffs, fmt.Sprintf("region: configured %q, API reports %q", v, got.Region))
	}
	if !want.IsActive.IsNull() && !want.IsActive.IsUnknown() && want.IsActive.ValueBool() != got.IsActive {
		diffs = append(diffs, fmt.Sprintf("is_active: configured %t, API reports %t", want.IsActive.ValueBool(), got.IsActive))
	}
	if len(diffs) == 0 {
		return ""
	}
	return "The API accepted the request but reports different values, so state records what the API reports:\n  - " +
		strings.Join(diffs, "\n  - ")
}
