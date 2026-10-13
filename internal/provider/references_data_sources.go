package provider

// Data sources over the read-only reference lists of the v1 API:
// GET /api/v1/regions, /api/v1/escalation-policies and /api/v1/webhooks.

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

// ---------------------------------------------------------------------------
// uptimewarden_regions
// ---------------------------------------------------------------------------

func NewRegionsDataSource() datasource.DataSource { return &regionsDataSource{} }

type regionsDataSource struct{ client *client.Client }

type regionsModel struct {
	Codes   types.List   `tfsdk:"codes"`
	Default types.String `tfsdk:"default"`
	Regions types.List   `tfsdk:"regions"`
	Plan    types.Object `tfsdk:"plan"`
}

var regionAttrTypes = map[string]attr.Type{
	"code": types.StringType, "name": types.StringType, "description": types.StringType, "is_default": types.BoolType,
}

var allowanceAttrTypes = map[string]attr.Type{
	"region_selection":           types.BoolType,
	"max_regions":                types.Int64Type,
	"max_verification_regions":   types.Int64Type,
	"default_confirming_regions": types.Int64Type,
	"allow_consensus_override":   types.BoolType,
}

func (d *regionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

func (d *regionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The active monitoring regions and what your plan allows (`GET /api/v1/regions`).",
		Attributes: map[string]schema.Attribute{
			"codes":   schema.ListAttribute{MarkdownDescription: "Active region codes.", ElementType: types.StringType, Computed: true},
			"default": schema.StringAttribute{MarkdownDescription: "The region new monitors get when none is chosen.", Computed: true},
			"regions": schema.ListNestedAttribute{
				MarkdownDescription: "Active regions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"code":        schema.StringAttribute{MarkdownDescription: "Region code, e.g. `gwc`.", Computed: true},
					"name":        schema.StringAttribute{MarkdownDescription: "Display name, e.g. `Sweden Central`.", Computed: true},
					"description": schema.StringAttribute{MarkdownDescription: "Area, e.g. `Europe`.", Computed: true},
					"is_default":  schema.BoolAttribute{MarkdownDescription: "Whether this is the default region.", Computed: true},
				}},
			},
			"plan": schema.SingleNestedAttribute{
				MarkdownDescription: "Your plan's region allowance. `-1` means unlimited. Null when the API does not report it.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"region_selection":           schema.BoolAttribute{MarkdownDescription: "Whether you can choose a monitor's region.", Computed: true},
					"max_regions":                schema.Int64Attribute{MarkdownDescription: "Regions your monitors may use in total.", Computed: true},
					"max_verification_regions":   schema.Int64Attribute{MarkdownDescription: "Regions (home + verification) one monitor may vote across.", Computed: true},
					"default_confirming_regions": schema.Int64Attribute{MarkdownDescription: "Default `min_confirming_regions`.", Computed: true},
					"allow_consensus_override":   schema.BoolAttribute{MarkdownDescription: "Whether `min_confirming_regions` may differ from the default.", Computed: true},
				},
			},
		},
	}
}

func (d *regionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *regionsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	r, err := d.client.ListRegions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing regions", err.Error())
		return
	}
	codes := make([]string, 0, len(r.Regions))
	rows := make([]attr.Value, 0, len(r.Regions))
	def := types.StringNull()
	for _, reg := range r.Regions {
		codes = append(codes, reg.Code)
		if reg.IsDefault {
			def = types.StringValue(reg.Code)
		}
		rows = append(rows, types.ObjectValueMust(regionAttrTypes, map[string]attr.Value{
			"code": types.StringValue(reg.Code), "name": types.StringValue(reg.Name),
			"description": optString(reg.Description), "is_default": types.BoolValue(reg.IsDefault),
		}))
	}
	plan := types.ObjectNull(allowanceAttrTypes)
	if r.Plan != nil {
		plan = types.ObjectValueMust(allowanceAttrTypes, map[string]attr.Value{
			"region_selection":           types.BoolValue(r.Plan.RegionSelection),
			"max_regions":                types.Int64Value(r.Plan.MaxRegions),
			"max_verification_regions":   types.Int64Value(r.Plan.MaxVerificationRegions),
			"default_confirming_regions": types.Int64Value(r.Plan.DefaultConfirmingRegions),
			"allow_consensus_override":   types.BoolValue(r.Plan.AllowConsensusOverride),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, regionsModel{
		Codes:   stringList(codes),
		Default: def,
		Regions: types.ListValueMust(types.ObjectType{AttrTypes: regionAttrTypes}, rows),
		Plan:    plan,
	})...)
}

// ---------------------------------------------------------------------------
// uptimewarden_escalation_policy
// ---------------------------------------------------------------------------

func NewEscalationPolicyDataSource() datasource.DataSource { return &escalationPolicyDataSource{} }

type escalationPolicyDataSource struct{ client *client.Client }

type escalationPolicyModel struct {
	Name      types.String `tfsdk:"name"`
	ID        types.String `tfsdk:"id"`
	StepCount types.Int64  `tfsdk:"step_count"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (d *escalationPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_escalation_policy"
}

func (d *escalationPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up one of your escalation policies by name, for `uptimewarden_monitor.escalation_policy_id`. " +
			"Policies are created in the dashboard. The name must match exactly one policy.",
		Attributes: map[string]schema.Attribute{
			"name":       schema.StringAttribute{MarkdownDescription: "Policy name (exact match).", Required: true},
			"id":         schema.StringAttribute{MarkdownDescription: "Public id (ULID).", Computed: true},
			"step_count": schema.Int64Attribute{MarkdownDescription: "Number of escalation steps.", Computed: true},
			"created_at": schema.StringAttribute{MarkdownDescription: "Creation time.", Computed: true},
			"updated_at": schema.StringAttribute{MarkdownDescription: "Last update time.", Computed: true},
		},
	}
}

func (d *escalationPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *escalationPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg escalationPolicyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	policies, err := d.client.ListEscalationPolicies(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing escalation policies", err.Error())
		return
	}
	name := cfg.Name.ValueString()
	var match []client.EscalationPolicy
	var names []string
	for _, p := range policies {
		names = append(names, p.Name)
		if p.Name == name {
			match = append(match, p)
		}
	}
	if msg := matchError("escalation policy", name, len(match), names); msg != "" {
		resp.Diagnostics.AddError("Escalation policy not found", msg)
		return
	}
	p := match[0]
	resp.Diagnostics.Append(resp.State.Set(ctx, escalationPolicyModel{
		Name: cfg.Name, ID: types.StringValue(p.PublicID), StepCount: types.Int64Value(p.StepCount),
		CreatedAt: types.StringValue(p.CreatedAt), UpdatedAt: types.StringValue(p.UpdatedAt),
	})...)
}

// ---------------------------------------------------------------------------
// uptimewarden_webhook
// ---------------------------------------------------------------------------

func NewWebhookDataSource() datasource.DataSource { return &webhookDataSource{} }

type webhookDataSource struct{ client *client.Client }

type webhookModel struct {
	Name                types.String `tfsdk:"name"`
	ID                  types.String `tfsdk:"id"`
	Description         types.String `tfsdk:"description"`
	MessageFormat       types.String `tfsdk:"message_format"`
	IsActive            types.Bool   `tfsdk:"is_active"`
	EnabledAlertTypes   types.List   `tfsdk:"enabled_alert_types"`
	EnabledMonitorTypes types.List   `tfsdk:"enabled_monitor_types"`
	CreatedAt           types.String `tfsdk:"created_at"`
}

func (d *webhookDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (d *webhookDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up one of your webhook configurations by name, for `uptimewarden_monitor.selected_webhook_ids`. " +
			"Webhooks are created in the dashboard. The name must match exactly one webhook.",
		Attributes: map[string]schema.Attribute{
			"name":                  schema.StringAttribute{MarkdownDescription: "Webhook name (exact match).", Required: true},
			"id":                    schema.StringAttribute{MarkdownDescription: "Public id (ULID).", Computed: true},
			"description":           schema.StringAttribute{MarkdownDescription: "Description.", Computed: true},
			"message_format":        schema.StringAttribute{MarkdownDescription: "Message format.", Computed: true},
			"is_active":             schema.BoolAttribute{MarkdownDescription: "Whether the webhook is active.", Computed: true},
			"enabled_alert_types":   schema.ListAttribute{MarkdownDescription: "Alert types it sends.", ElementType: types.StringType, Computed: true},
			"enabled_monitor_types": schema.ListAttribute{MarkdownDescription: "Monitor types it applies to.", ElementType: types.StringType, Computed: true},
			"created_at":            schema.StringAttribute{MarkdownDescription: "Creation time.", Computed: true},
		},
	}
}

func (d *webhookDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDataSourceClient(req, resp)
}

func (d *webhookDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg webhookModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	hooks, err := d.client.ListWebhooks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing webhooks", err.Error())
		return
	}
	name := cfg.Name.ValueString()
	var match []client.Webhook
	var names []string
	for _, h := range hooks {
		names = append(names, h.Name)
		if h.Name == name {
			match = append(match, h)
		}
	}
	if msg := matchError("webhook", name, len(match), names); msg != "" {
		resp.Diagnostics.AddError("Webhook not found", msg)
		return
	}
	h := match[0]
	resp.Diagnostics.Append(resp.State.Set(ctx, webhookModel{
		Name: cfg.Name, ID: types.StringValue(h.PublicID), Description: optString(h.Description),
		MessageFormat: optString(h.MessageFormat), IsActive: types.BoolValue(h.IsActive),
		EnabledAlertTypes: stringList(h.EnabledAlertTypes), EnabledMonitorTypes: stringList(h.EnabledMonitorTypes),
		CreatedAt: types.StringValue(h.CreatedAt),
	})...)
}

// matchError explains a lookup by name that did not find exactly one item.
func matchError(kind, name string, n int, available []string) string {
	switch {
	case n == 1:
		return ""
	case n > 1:
		return fmt.Sprintf("%d %ss are named %q; rename them in the dashboard so the name is unique.", n, kind, name)
	case len(available) == 0:
		return fmt.Sprintf("No %s named %q: this account has none.", kind, name)
	}
	return fmt.Sprintf("No %s named %q. Available: %s.", kind, name, strings.Join(quoteAll(available), ", "))
}

func quoteAll(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = fmt.Sprintf("%q", s)
	}
	return out
}
