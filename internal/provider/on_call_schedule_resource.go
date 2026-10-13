package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

var _ resource.Resource = &onCallScheduleResource{}
var _ resource.ResourceWithImportState = &onCallScheduleResource{}

func NewOnCallScheduleResource() resource.Resource { return &onCallScheduleResource{} }

type onCallScheduleResource struct{ client *client.Client }

type onCallScheduleModel struct {
	ID       types.String             `tfsdk:"id"`
	Name     types.String             `tfsdk:"name"`
	Timezone types.String             `tfsdk:"timezone"`
	Members  []onCallScheduleMember   `tfsdk:"members"`
}

type onCallScheduleMember struct {
	UserID    types.Int64 `tfsdk:"user_id"`
	DayOfWeek types.Int64 `tfsdk:"day_of_week"`
}

func (r *onCallScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_on_call_schedule"
}

func (r *onCallScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a weekly on-call schedule. Each member is assigned to one or more days of the week. Schedules are referenced by escalation policy steps with `target_type = \"schedule\"`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{Required: true},
			"timezone": schema.StringAttribute{
				MarkdownDescription: "IANA timezone for the schedule (e.g. `Europe/Istanbul`). Default: `UTC`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("UTC"),
			},
			"members": schema.ListNestedAttribute{
				Optional:            true,
				MarkdownDescription: "On-call rotation members. Each maps a user to a day of the week (0=Sunday … 6=Saturday). Maximum 7 members (one per day).",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id": schema.Int64Attribute{
							MarkdownDescription: "Internal ID of the user on call for this day.",
							Required:            true,
							Validators:          []validator.Int64{int64validator.AtLeast(1)},
						},
						"day_of_week": schema.Int64Attribute{
							MarkdownDescription: "Day of week (0=Sunday, 1=Monday … 6=Saturday).",
							Required:            true,
							Validators:          []validator.Int64{int64validator.Between(0, 6)},
						},
					},
				},
			},
		},
	}
}

func (r *onCallScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildScheduleRequest(m *onCallScheduleModel) client.OnCallScheduleRequest {
	members := make([]client.OnCallScheduleMemberInput, 0, len(m.Members))
	for _, mb := range m.Members {
		members = append(members, client.OnCallScheduleMemberInput{
			UserID:    mb.UserID.ValueInt64(),
			DayOfWeek: int(mb.DayOfWeek.ValueInt64()),
		})
	}
	return client.OnCallScheduleRequest{
		Name:     m.Name.ValueString(),
		Timezone: m.Timezone.ValueString(),
		Members:  members,
	}
}

func onCallScheduleToState(s *client.OnCallSchedule, m *onCallScheduleModel) {
	m.ID = types.StringValue(s.PublicID)
	m.Name = types.StringValue(s.Name)
	m.Timezone = types.StringValue(s.Timezone)
	members := make([]onCallScheduleMember, 0, len(s.Members))
	for _, mb := range s.Members {
		members = append(members, onCallScheduleMember{
			UserID:    types.Int64Value(mb.UserID),
			DayOfWeek: types.Int64Value(int64(mb.DayOfWeek)),
		})
	}
	m.Members = members
}

func (r *onCallScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan onCallScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateOnCallSchedule(ctx, buildScheduleRequest(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating on-call schedule", err.Error())
		return
	}
	onCallScheduleToState(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *onCallScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state onCallScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	s, err := r.client.GetOnCallSchedule(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFoundErr(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading on-call schedule", err.Error())
		return
	}
	onCallScheduleToState(s, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *onCallScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan onCallScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UpdateOnCallSchedule(ctx, plan.ID.ValueString(), buildScheduleRequest(&plan)); err != nil {
		resp.Diagnostics.AddError("Error updating on-call schedule", err.Error())
		return
	}
	s, err := r.client.GetOnCallSchedule(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading on-call schedule after update", err.Error())
		return
	}
	onCallScheduleToState(s, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *onCallScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state onCallScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteOnCallSchedule(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting on-call schedule", err.Error())
	}
}

func (r *onCallScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	s, err := r.client.GetOnCallSchedule(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error importing on-call schedule", err.Error())
		return
	}
	var state onCallScheduleModel
	onCallScheduleToState(s, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}
