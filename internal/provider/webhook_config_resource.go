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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

var _ resource.Resource = &webhookConfigResource{}
var _ resource.ResourceWithImportState = &webhookConfigResource{}

func NewWebhookConfigResource() resource.Resource { return &webhookConfigResource{} }

type webhookConfigResource struct{ client *client.Client }

type webhookConfigModel struct {
	ID          types.String `tfsdk:"id"`
	TypeID      types.Int64  `tfsdk:"type_id"`
	Name        types.String `tfsdk:"name"`
	URL         types.String `tfsdk:"url"`
	HTTPMethod  types.String `tfsdk:"http_method"`
	Description types.String `tfsdk:"description"`
	AuthMethod  types.String `tfsdk:"auth_method"`
	Timeout     types.Int64  `tfsdk:"timeout"`
	RetryCount  types.Int64  `tfsdk:"retry_count"`
	RetryDelay  types.Int64  `tfsdk:"retry_delay"`
	IsActive    types.Bool   `tfsdk:"is_active"`
}

func (r *webhookConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_config"
}

func (r *webhookConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an outbound webhook configuration. Webhook configs define HTTP endpoints that receive alert notifications. Use `type_id = 1` for a generic HTTP webhook; query `GET /api/webhooks/types` for the full list.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type_id": schema.Int64Attribute{
				MarkdownDescription: "Webhook type ID. Use `1` for a generic HTTP webhook. See `GET /api/webhooks/types` for available IDs.",
				Required:            true,
				Validators:          []validator.Int64{int64validator.AtLeast(1)},
			},
			"name": schema.StringAttribute{Required: true},
			"url": schema.StringAttribute{
				MarkdownDescription: "Destination URL (must be http or https).",
				Required:            true,
			},
			"http_method": schema.StringAttribute{
				MarkdownDescription: "HTTP method. One of: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`. Default: `POST`.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("GET", "POST", "PUT", "PATCH", "DELETE"),
				},
			},
			"description": schema.StringAttribute{Optional: true, Computed: true},
			"auth_method": schema.StringAttribute{
				MarkdownDescription: "Authentication method (e.g. `bearer`, `basic`). Optional.",
				Optional:            true,
				Computed:            true,
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Request timeout in seconds. Default: 30.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(30),
			},
			"retry_count": schema.Int64Attribute{
				MarkdownDescription: "Number of retry attempts on failure. Default: 3.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3),
			},
			"retry_delay": schema.Int64Attribute{
				MarkdownDescription: "Delay between retries in seconds. Default: 60.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(60),
			},
			"is_active": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
		},
	}
}

func (r *webhookConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *webhookConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	isActive := plan.IsActive.ValueBool()
	created, err := r.client.CreateWebhookConfig(ctx, client.CreateWebhookConfigRequest{
		TypeID:     plan.TypeID.ValueInt64(),
		Name:       plan.Name.ValueString(),
		URL:        plan.URL.ValueString(),
		HTTPMethod: plan.HTTPMethod.ValueString(),
		Timeout:    int(plan.Timeout.ValueInt64()),
		RetryCount: int(plan.RetryCount.ValueInt64()),
		RetryDelay: int(plan.RetryDelay.ValueInt64()),
		IsActive:   &isActive,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook config", err.Error())
		return
	}
	webhookConfigToState(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *webhookConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	w, err := r.client.GetWebhookConfig(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFoundErr(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading webhook config", err.Error())
		return
	}
	webhookConfigToState(w, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *webhookConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	isActive := plan.IsActive.ValueBool()
	name := plan.Name.ValueString()
	url := plan.URL.ValueString()
	method := plan.HTTPMethod.ValueString()
	to := int(plan.Timeout.ValueInt64())
	rc := int(plan.RetryCount.ValueInt64())
	rd := int(plan.RetryDelay.ValueInt64())
	updated, err := r.client.UpdateWebhookConfig(ctx, plan.ID.ValueString(), client.UpdateWebhookConfigRequest{
		Name:       &name,
		URL:        &url,
		HTTPMethod: &method,
		Timeout:    &to,
		RetryCount: &rc,
		RetryDelay: &rd,
		IsActive:   &isActive,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook config", err.Error())
		return
	}
	webhookConfigToState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *webhookConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteWebhookConfig(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting webhook config", err.Error())
	}
}

func (r *webhookConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	w, err := r.client.GetWebhookConfig(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing webhook config", err.Error())
		return
	}
	var state webhookConfigModel
	webhookConfigToState(w, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func webhookConfigToState(w *client.WebhookConfig, s *webhookConfigModel) {
	s.ID = types.StringValue(w.PublicID)
	s.TypeID = types.Int64Value(w.TypeID)
	s.Name = types.StringValue(w.Name)
	s.URL = types.StringValue(w.URL)
	s.HTTPMethod = types.StringValue(w.HTTPMethod)
	s.Timeout = types.Int64Value(int64(w.Timeout))
	s.RetryCount = types.Int64Value(int64(w.RetryCount))
	s.RetryDelay = types.Int64Value(int64(w.RetryDelay))
	s.IsActive = types.BoolValue(w.IsActive)
	if w.Description != nil {
		s.Description = types.StringValue(*w.Description)
	} else {
		s.Description = types.StringNull()
	}
	if w.AuthMethod != nil {
		s.AuthMethod = types.StringValue(*w.AuthMethod)
	} else {
		s.AuthMethod = types.StringNull()
	}
}
