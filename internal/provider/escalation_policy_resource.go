package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

var _ resource.Resource = &escalationPolicyResource{}
var _ resource.ResourceWithImportState = &escalationPolicyResource{}

func NewEscalationPolicyResource() resource.Resource { return &escalationPolicyResource{} }

type escalationPolicyResource struct{ client *client.Client }

type escalationPolicyModel struct {
	ID    types.String          `tfsdk:"id"`
	Name  types.String          `tfsdk:"name"`
	Steps []escalationStepModel `tfsdk:"steps"`
}

type escalationStepModel struct {
	StepOrder   types.Int64    `tfsdk:"step_order"`
	TargetType  types.String   `tfsdk:"target_type"`
	TargetID    types.Int64    `tfsdk:"target_id"`
	WaitMinutes types.Int64    `tfsdk:"wait_minutes"`
	Channels    []types.String `tfsdk:"channels"`
}

func (r *escalationPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_escalation_policy"
}

func (r *escalationPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an on-call escalation policy. A policy defines an ordered chain of targets (users or schedules) that get paged when an incident is not acknowledged.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{Required: true},
			"steps": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Ordered escalation steps. Up to 10 steps.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"step_order": schema.Int64Attribute{
							Required:   true,
							Validators: []validator.Int64{int64validator.Between(0, 9)},
						},
						"target_type": schema.StringAttribute{
							Required:   true,
							Validators: []validator.String{stringvalidator.OneOf("user", "schedule")},
						},
						"target_id": schema.Int64Attribute{
							MarkdownDescription: "Internal ID of the user or schedule to page.",
							Required:            true,
							Validators:          []validator.Int64{int64validator.AtLeast(1)},
						},
						"wait_minutes": schema.Int64Attribute{
							MarkdownDescription: "Minutes to wait before escalating to the next step (0–1440). Default: 5.",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(5),
							Validators:          []validator.Int64{int64validator.Between(0, 1440)},
						},
						"channels": schema.ListAttribute{
							ElementType:         types.StringType,
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Notification channels for this step. Defaults to `[\"email\",\"sms\",\"push\"]`.",
						},
					},
				},
			},
		},
	}
}

func (r *escalationPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildPolicyRequest(m *escalationPolicyModel) client.EscalationPolicyRequest {
	steps := make([]client.EscalationPolicyStepInput, 0, len(m.Steps))
	for _, s := range m.Steps {
		wait := int(s.WaitMinutes.ValueInt64())
		channels := make([]string, 0, len(s.Channels))
		for _, ch := range s.Channels {
			channels = append(channels, ch.ValueString())
		}
		steps = append(steps, client.EscalationPolicyStepInput{
			StepOrder:   int(s.StepOrder.ValueInt64()),
			TargetType:  s.TargetType.ValueString(),
			TargetID:    s.TargetID.ValueInt64(),
			WaitMinutes: &wait,
			Channels:    channels,
		})
	}
	return client.EscalationPolicyRequest{Name: m.Name.ValueString(), Steps: steps}
}

func escalationPolicyToState(p *client.EscalationPolicy, m *escalationPolicyModel) {
	m.ID = types.StringValue(p.PublicID)
	m.Name = types.StringValue(p.Name)
	steps := make([]escalationStepModel, 0, len(p.Steps))
	for _, s := range p.Steps {
		channels := make([]types.String, 0, len(s.Channels))
		for _, ch := range s.Channels {
			channels = append(channels, types.StringValue(ch))
		}
		steps = append(steps, escalationStepModel{
			StepOrder:   types.Int64Value(int64(s.StepOrder)),
			TargetType:  types.StringValue(s.TargetType),
			TargetID:    types.Int64Value(s.TargetID),
			WaitMinutes: types.Int64Value(int64(s.WaitMinutes)),
			Channels:    channels,
		})
	}
	m.Steps = steps
}

func (r *escalationPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan escalationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateEscalationPolicy(ctx, buildPolicyRequest(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating escalation policy", err.Error())
		return
	}
	escalationPolicyToState(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *escalationPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state escalationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.GetEscalationPolicy(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFoundErr(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading escalation policy", err.Error())
		return
	}
	escalationPolicyToState(p, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *escalationPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan escalationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UpdateEscalationPolicy(ctx, plan.ID.ValueString(), buildPolicyRequest(&plan)); err != nil {
		resp.Diagnostics.AddError("Error updating escalation policy", err.Error())
		return
	}
	// PUT returns 204; re-fetch to confirm.
	p, err := r.client.GetEscalationPolicy(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading escalation policy after update", err.Error())
		return
	}
	escalationPolicyToState(p, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *escalationPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state escalationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteEscalationPolicy(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting escalation policy", err.Error())
	}
}

func (r *escalationPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	p, err := r.client.GetEscalationPolicy(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing escalation policy", err.Error())
		return
	}
	var state escalationPolicyModel
	escalationPolicyToState(p, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
