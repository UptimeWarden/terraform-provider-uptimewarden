package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

var _ resource.Resource = &apiTokenResource{}

// apiTokenResource intentionally does NOT implement ResourceWithImportState.
// The plainToken value is only returned at creation and cannot be recovered —
// importing a token would leave the sensitive value empty in state, which is
// misleading. Users should create new tokens via Terraform instead.

func NewAPITokenResource() resource.Resource { return &apiTokenResource{} }

type apiTokenResource struct{ client *client.Client }

type apiTokenModel struct {
	ID          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Permissions []types.String `tfsdk:"permissions"`
	ExpiresAt   types.String   `tfsdk:"expires_at"`
	IsActive    types.Bool     `tfsdk:"is_active"`
	Prefix      types.String   `tfsdk:"prefix"`
	// token is the plain-text token returned only on create (sensitive).
	Token types.String `tfsdk:"token"`
}

func (r *apiTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_token"
}

func (r *apiTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates a UptimeWarden API token with granular permissions. The plain token value (`token`) is returned **only at creation** and stored in Terraform state as a sensitive value — treat it like a secret. Destroying and re-creating this resource generates a new token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{Required: true},
			"permissions": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Permission strings (e.g. `monitors:read`, `incidents:read`). Empty list grants no permissions.",
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "ISO 8601 expiry timestamp. Omit for no expiry.",
				Optional:            true,
				Computed:            true,
			},
			"is_active": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"prefix": schema.StringAttribute{
				MarkdownDescription: "First 12 characters of the token (e.g. `uptw_XXXXXXX`). Safe to log.",
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "Plain-text API token. Available only immediately after creation. Stored in state as a sensitive value — do not log.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *apiTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *apiTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	perms := make([]string, 0, len(plan.Permissions))
	for _, p := range plan.Permissions {
		perms = append(perms, p.ValueString())
	}
	createReq := client.CreateAPITokenRequest{
		Name:        plan.Name.ValueString(),
		Permissions: perms,
	}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		v := plan.ExpiresAt.ValueString()
		createReq.ExpiresAt = &v
	}

	result, err := r.client.CreateAPIToken(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating API token", err.Error())
		return
	}

	apiTokenToState(&result.Token, &plan)
	plan.Token = types.StringValue(result.PlainToken)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *apiTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	t, err := r.client.GetAPIToken(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFoundErr(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading API token", err.Error())
		return
	}
	apiTokenToState(t, &state)
	// token (plainToken) is not refreshable — keep the value already in state.
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *apiTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	isActive := plan.IsActive.ValueBool()
	name := plan.Name.ValueString()
	updateReq := client.UpdateAPITokenRequest{Name: &name, IsActive: &isActive}
	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		v := plan.ExpiresAt.ValueString()
		updateReq.ExpiresAt = &v
	}
	t, err := r.client.UpdateAPIToken(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating API token", err.Error())
		return
	}
	// Update permissions separately.
	perms := make([]string, 0, len(plan.Permissions))
	for _, p := range plan.Permissions {
		perms = append(perms, p.ValueString())
	}
	if err := r.client.UpdateAPITokenPermissions(ctx, plan.ID.ValueString(), perms); err != nil {
		resp.Diagnostics.AddError("Error updating API token permissions", err.Error())
		return
	}
	apiTokenToState(t, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *apiTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAPIToken(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting API token", err.Error())
	}
}

func apiTokenToState(t *client.APIToken, m *apiTokenModel) {
	m.ID = types.StringValue(t.PublicID)
	m.Name = types.StringValue(t.Name)
	m.Prefix = types.StringValue(t.Prefix)
	m.IsActive = types.BoolValue(t.IsActive)
	if t.ExpiresAt != nil {
		m.ExpiresAt = types.StringValue(*t.ExpiresAt)
	} else {
		m.ExpiresAt = types.StringNull()
	}
	perms := make([]types.String, 0, len(t.Permissions))
	for _, p := range t.Permissions {
		perms = append(perms, types.StringValue(p))
	}
	m.Permissions = perms
}
