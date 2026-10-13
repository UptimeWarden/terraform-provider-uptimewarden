package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

var _ resource.Resource = &uptimewardenMonitorResource{}
var _ resource.ResourceWithImportState = &uptimewardenMonitorResource{}
var _ resource.ResourceWithValidateConfig = &uptimewardenMonitorResource{}

// NewMonitorResource is registered in provider.go.
func NewMonitorResource() resource.Resource {
	return &uptimewardenMonitorResource{}
}

// uptimewardenMonitorResource implements CRUD for uptimewarden_monitor over
// /api/v1/monitors. The attributes come from monitorFields.
type uptimewardenMonitorResource struct {
	client *client.Client
}

func (r *uptimewardenMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *uptimewardenMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an UptimeWarden monitor: what it checks, from where, its thresholds and who is alerted. " +
			"Each attribute is the snake_case form of the `/api/v1/monitors` field. An optional attribute left out of the " +
			"configuration keeps the value UptimeWarden holds (its default on create).",
		Attributes: resourceAttributes(),
	}
}

func (r *uptimewardenMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

// ValidateConfig: url is required except for heartbeat monitors, where the
// API sets it.
func (r *uptimewardenMonitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var typ, url types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &typ)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("url"), &url)...)
	if resp.Diagnostics.HasError() || typ.IsUnknown() || typ.IsNull() || url.IsUnknown() {
		return
	}
	if typ.ValueString() == "heartbeat" && !url.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "url is set by UptimeWarden for heartbeat monitors",
			"Remove url: a heartbeat monitor's url is heartbeat://<token>. Use heartbeat_url for the ping URL.")
	}
	if typ.ValueString() != "heartbeat" && url.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Missing url",
			fmt.Sprintf("url is required for %s monitors.", typ.ValueString()))
	}
}

// getAttrs reads the whole plan/state/config as attribute values.
func getAttrs(ctx context.Context, src interface {
	Get(context.Context, any) diag.Diagnostics
}) (map[string]attr.Value, diag.Diagnostics) {
	var obj types.Object
	diags := src.Get(ctx, &obj)
	if diags.HasError() {
		return nil, diags
	}
	return obj.Attributes(), diags
}

func setState(ctx context.Context, state *tfsdk.State, attrs map[string]attr.Value) diag.Diagnostics {
	obj, diags := types.ObjectValue(monitorObjectType().AttrTypes, attrs)
	if diags.HasError() {
		return diags
	}
	return state.Set(ctx, obj)
}

func (r *uptimewardenMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan, diags := getAttrs(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateMonitor(ctx, createBody(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor", err.Error())
		return
	}
	id := created.PublicID()
	tflog.Trace(ctx, "created monitor", map[string]any{"id": id})

	// Fields the API takes on update only (groups), and anything the create
	// did not apply (a timeout on an API older than the 2026-10 contract),
	// are set with one follow-up PATCH.
	state := monitorStateFromAPI(created, plan)
	followUp := map[string]any{}
	for _, f := range monitorFields {
		if f.mode == mComputed {
			continue
		}
		if f.patchOnly || (f.tf == "timeout" && !knownPartsEqual(plan[f.tf], state[f.tf])) {
			if v, ok := toAPI(f, plan[f.tf]); ok {
				followUp[f.api] = v
			}
		}
	}
	if len(followUp) > 0 && id != "" {
		updated, uErr := r.client.UpdateMonitor(ctx, id, followUp)
		if uErr != nil {
			// The monitor exists: keep it in state so the next apply retries
			// instead of creating a duplicate.
			resp.Diagnostics.Append(setState(ctx, &resp.State, state)...)
			resp.Diagnostics.AddError("Error finishing monitor creation",
				fmt.Sprintf("The monitor was created (id %s) but the follow-up update failed: %s", id, uErr))
			return
		}
		state = monitorStateFromAPI(updated, plan)
	}

	resp.Diagnostics.Append(setState(ctx, &resp.State, state)...)
	if diffs := unappliedFields(plan, state); len(diffs) > 0 {
		resp.Diagnostics.AddError("UptimeWarden did not apply the configuration",
			"The API accepted the request but returned different values; state records what the API returned:\n"+joinDiffs(diffs))
	}
}

func (r *uptimewardenMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	prior, diags := getAttrs(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := prior["id"].(types.String).ValueString()

	m, err := r.client.GetMonitor(ctx, id)
	if err != nil {
		if client.IsNotFoundErr(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading monitor", err.Error())
		return
	}
	resp.Diagnostics.Append(setState(ctx, &resp.State, monitorStateFromAPI(m, prior))...)
}

func (r *uptimewardenMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan, diags := getAttrs(ctx, req.Plan)
	resp.Diagnostics.Append(diags...)
	prior, diags := getAttrs(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := prior["id"].(types.String).ValueString()

	body := updateBody(plan, prior)
	var m client.Monitor
	var err error
	if len(body) == 0 {
		m, err = r.client.GetMonitor(ctx, id)
	} else {
		m, err = r.client.UpdateMonitor(ctx, id, body)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating monitor", err.Error())
		return
	}

	state := monitorStateFromAPI(m, plan)
	resp.Diagnostics.Append(setState(ctx, &resp.State, state)...)
	if diffs := unappliedFields(plan, state); len(diffs) > 0 {
		resp.Diagnostics.AddError("UptimeWarden did not apply the configuration",
			"The API accepted the request but returned different values; state records what the API returned:\n"+joinDiffs(diffs))
	}
}

func (r *uptimewardenMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	prior, diags := getAttrs(ctx, req.State)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := prior["id"].(types.String).ValueString()
	if err := r.client.DeleteMonitor(ctx, id); err != nil {
		if client.IsNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting monitor", err.Error())
		return
	}
	tflog.Trace(ctx, "deleted monitor", map[string]any{"id": id})
}

// ImportState supports `terraform import uptimewarden_monitor.example <publicId>`.
func (r *uptimewardenMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	m, err := r.client.GetMonitor(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing monitor", err.Error())
		return
	}
	resp.Diagnostics.Append(setState(ctx, &resp.State, monitorStateFromAPI(m, nil))...)
}
