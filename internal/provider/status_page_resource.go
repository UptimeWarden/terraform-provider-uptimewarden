package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

var _ resource.Resource = &statusPageResource{}
var _ resource.ResourceWithImportState = &statusPageResource{}

func NewStatusPageResource() resource.Resource { return &statusPageResource{} }

type statusPageResource struct{ client *client.Client }

type statusPageModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Title              types.String `tfsdk:"title"`
	Slug               types.String `tfsdk:"slug"`
	Description        types.String `tfsdk:"description"`
	Domain             types.String `tfsdk:"domain"`
	LogoURL            types.String `tfsdk:"logo_url"`
	FaviconURL         types.String `tfsdk:"favicon_url"`
	IsPublic           types.Bool   `tfsdk:"is_public"`
	AllowSubscriptions types.Bool   `tfsdk:"allow_subscriptions"`
	ShowHistory        types.Bool   `tfsdk:"show_history"`
	HistoryDays        types.Int64  `tfsdk:"history_days"`
	ShowResponseTime   types.Bool   `tfsdk:"show_response_time"`
	ShowUptime         types.Bool   `tfsdk:"show_uptime"`
	HidePoweredBy      types.Bool   `tfsdk:"hide_powered_by"`
}

func (r *statusPageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (r *statusPageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a public or private UptimeWarden status page. Custom domain support requires Professional or Enterprise plan.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Internal name for the status page (not shown publicly).",
				Required:            true,
			},
			"title": schema.StringAttribute{
				MarkdownDescription: "Public-facing title displayed on the status page.",
				Required:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "URL slug (e.g. `acme` → `status.uptimewarden.online/acme`). Must be unique.",
				Required:            true,
			},
			"description":   schema.StringAttribute{Optional: true, Computed: true},
			"domain":        schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: "Custom domain (e.g. `status.acme.com`). Requires Professional+."},
			"logo_url":      schema.StringAttribute{Optional: true, Computed: true},
			"favicon_url":   schema.StringAttribute{Optional: true, Computed: true},
			"is_public":     schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"allow_subscriptions": schema.BoolAttribute{
				MarkdownDescription: "Allow visitors to subscribe to email updates.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"show_history":       schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"history_days":       schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(90)},
			"show_response_time": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"show_uptime":        schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"hide_powered_by":    schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		},
	}
}

func (r *statusPageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *statusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan statusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	isPublic := plan.IsPublic.ValueBool()
	allowSubs := plan.AllowSubscriptions.ValueBool()
	showHist := plan.ShowHistory.ValueBool()
	histDays := int(plan.HistoryDays.ValueInt64())
	showRT := plan.ShowResponseTime.ValueBool()
	showUp := plan.ShowUptime.ValueBool()
	hideBy := plan.HidePoweredBy.ValueBool()

	createReq := client.CreateStatusPageRequest{
		Name:               plan.Name.ValueString(),
		Title:              plan.Title.ValueString(),
		Slug:               plan.Slug.ValueString(),
		IsPublic:           &isPublic,
		AllowSubscriptions: &allowSubs,
		ShowHistory:        &showHist,
		HistoryDays:        &histDays,
		ShowResponseTime:   &showRT,
		ShowUptime:         &showUp,
		HidePoweredBy:      &hideBy,
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		createReq.Description = &v
	}
	if !plan.Domain.IsNull() {
		v := plan.Domain.ValueString()
		createReq.Domain = &v
	}

	created, err := r.client.CreateStatusPage(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating status page", err.Error())
		return
	}
	statusPageToState(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *statusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state statusPageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := r.client.GetStatusPage(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFoundErr(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading status page", err.Error())
		return
	}
	statusPageToState(s, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *statusPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan statusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()
	title := plan.Title.ValueString()
	slug := plan.Slug.ValueString()
	isPublic := plan.IsPublic.ValueBool()
	allowSubs := plan.AllowSubscriptions.ValueBool()
	showHist := plan.ShowHistory.ValueBool()
	histDays := int(plan.HistoryDays.ValueInt64())
	showRT := plan.ShowResponseTime.ValueBool()
	showUp := plan.ShowUptime.ValueBool()
	hideBy := plan.HidePoweredBy.ValueBool()

	updated, err := r.client.UpdateStatusPage(ctx, plan.ID.ValueString(), client.UpdateStatusPageRequest{
		Name:               &name,
		Title:              &title,
		Slug:               &slug,
		IsPublic:           &isPublic,
		AllowSubscriptions: &allowSubs,
		ShowHistory:        &showHist,
		HistoryDays:        &histDays,
		ShowResponseTime:   &showRT,
		ShowUptime:         &showUp,
		HidePoweredBy:      &hideBy,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating status page", err.Error())
		return
	}
	statusPageToState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *statusPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state statusPageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteStatusPage(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting status page", err.Error())
	}
}

func (r *statusPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	s, err := r.client.GetStatusPage(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing status page", err.Error())
		return
	}
	var state statusPageModel
	statusPageToState(s, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func statusPageToState(s *client.StatusPage, m *statusPageModel) {
	m.ID = types.StringValue(s.PublicID)
	m.Name = types.StringValue(s.Name)
	m.Title = types.StringValue(s.Title)
	m.Slug = types.StringValue(s.Slug)
	m.IsPublic = types.BoolValue(s.IsPublic)
	m.AllowSubscriptions = types.BoolValue(s.AllowSubscriptions)
	m.ShowHistory = types.BoolValue(s.ShowHistory)
	m.HistoryDays = types.Int64Value(int64(s.HistoryDays))
	m.ShowResponseTime = types.BoolValue(s.ShowResponseTime)
	m.ShowUptime = types.BoolValue(s.ShowUptime)
	m.HidePoweredBy = types.BoolValue(s.HidePoweredBy)
	if s.Description != nil {
		m.Description = types.StringValue(*s.Description)
	} else {
		m.Description = types.StringNull()
	}
	if s.Domain != nil {
		m.Domain = types.StringValue(*s.Domain)
	} else {
		m.Domain = types.StringNull()
	}
	if s.LogoURL != nil {
		m.LogoURL = types.StringValue(*s.LogoURL)
	} else {
		m.LogoURL = types.StringNull()
	}
	if s.FaviconURL != nil {
		m.FaviconURL = types.StringValue(*s.FaviconURL)
	} else {
		m.FaviconURL = types.StringNull()
	}
}
