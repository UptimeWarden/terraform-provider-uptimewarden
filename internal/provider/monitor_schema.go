package provider

import (
	"context"

	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// resourceAttributes builds the uptimewarden_monitor resource schema from
// monitorFields. Top-level optional fields keep their prior state in plans
// (UseStateForUnknown): leaving one out of the configuration keeps whatever
// the API holds instead of showing "(known after apply)" on every change.
func resourceAttributes() map[string]rschema.Attribute {
	out := map[string]rschema.Attribute{
		"id": rschema.StringAttribute{
			MarkdownDescription: "Public ID of the monitor (ULID). Assigned by UptimeWarden on creation.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
	for _, f := range monitorFields {
		out[f.tf] = resourceAttribute(f, true)
	}
	return out
}

func resourceAttribute(f monitorField, topLevel bool) rschema.Attribute {
	required := f.mode == mRequired
	optional := f.mode == mOptional
	computed := f.mode != mRequired
	keep := topLevel && computed

	switch f.kind {
	case kString:
		a := rschema.StringAttribute{MarkdownDescription: f.desc, Required: required, Optional: optional, Computed: computed,
			Sensitive: f.sensitive, Validators: f.stringValidators}
		if keep {
			a.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
		}
		if topLevel && f.tf == "type" {
			a.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplaceIf(heartbeatTypeChange,
				"Changing to or from heartbeat replaces the monitor.", "Changing to or from `heartbeat` replaces the monitor.")}
		}
		return a
	case kInt:
		a := rschema.Int64Attribute{MarkdownDescription: f.desc, Required: required, Optional: optional, Computed: computed,
			Sensitive: f.sensitive, Validators: f.int64Validators}
		if keep {
			a.PlanModifiers = []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
		}
		return a
	case kBool:
		a := rschema.BoolAttribute{MarkdownDescription: f.desc, Required: required, Optional: optional, Computed: computed, Sensitive: f.sensitive}
		if keep {
			a.PlanModifiers = []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}
		}
		return a
	case kStringList:
		a := rschema.ListAttribute{MarkdownDescription: f.desc, ElementType: types.StringType, Required: required, Optional: optional,
			Computed: computed, Sensitive: f.sensitive, Validators: f.listValidators}
		if keep {
			a.PlanModifiers = []planmodifier.List{listplanmodifier.UseStateForUnknown()}
		}
		return a
	case kStringSet:
		a := rschema.SetAttribute{MarkdownDescription: f.desc, ElementType: types.StringType, Required: required, Optional: optional,
			Computed: computed, Sensitive: f.sensitive, Validators: f.setValidators}
		if keep {
			a.PlanModifiers = []planmodifier.Set{setplanmodifier.UseStateForUnknown()}
		}
		return a
	case kStringMap:
		a := rschema.MapAttribute{MarkdownDescription: f.desc, ElementType: types.StringType, Required: required, Optional: optional,
			Computed: computed, Sensitive: f.sensitive}
		if keep {
			a.PlanModifiers = []planmodifier.Map{mapplanmodifier.UseStateForUnknown()}
		}
		return a
	case kBoolMapMap:
		a := rschema.MapAttribute{MarkdownDescription: f.desc, ElementType: types.MapType{ElemType: types.BoolType}, Required: required,
			Optional: optional, Computed: computed, Sensitive: f.sensitive}
		if keep {
			a.PlanModifiers = []planmodifier.Map{mapplanmodifier.UseStateForUnknown()}
		}
		return a
	case kObject:
		sub := map[string]rschema.Attribute{}
		for _, s := range f.sub {
			sub[s.tf] = resourceAttribute(s, false)
		}
		a := rschema.SingleNestedAttribute{MarkdownDescription: f.desc, Attributes: sub, Required: required, Optional: optional,
			Computed: computed, Sensitive: f.sensitive}
		if keep {
			a.PlanModifiers = []planmodifier.Object{objectplanmodifier.UseStateForUnknown()}
		}
		return a
	case kObjectList:
		sub := map[string]rschema.Attribute{}
		for _, s := range f.sub {
			sub[s.tf] = resourceAttribute(s, false)
		}
		a := rschema.ListNestedAttribute{MarkdownDescription: f.desc, NestedObject: rschema.NestedAttributeObject{Attributes: sub},
			Required: required, Optional: optional, Computed: computed, Sensitive: f.sensitive}
		if keep {
			a.PlanModifiers = []planmodifier.List{listplanmodifier.UseStateForUnknown()}
		}
		return a
	}
	panic("unknown field kind " + f.tf)
}

// heartbeatTypeChange: the API can change a monitor's type in place except to
// or from heartbeat.
func heartbeatTypeChange(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.StateValue.IsNull() || req.PlanValue.IsUnknown() || req.StateValue.Equal(req.PlanValue) {
		return
	}
	from, to := req.StateValue.ValueString(), req.PlanValue.ValueString()
	resp.RequiresReplace = from == "heartbeat" || to == "heartbeat"
}

// dataSourceAttributes is every monitor field as a computed attribute, plus
// the read-only fields the resource leaves out. id is the lookup key.
func dataSourceAttributes() map[string]dsschema.Attribute {
	out := map[string]dsschema.Attribute{
		"id": dsschema.StringAttribute{MarkdownDescription: "Public ID of the monitor to look up.", Required: true},
	}
	for _, f := range append(append([]monitorField{}, monitorFields...), dataSourceOnlyFields...) {
		out[f.tf] = dataSourceAttribute(f)
	}
	return out
}

func dataSourceAttribute(f monitorField) dsschema.Attribute {
	switch f.kind {
	case kString:
		return dsschema.StringAttribute{MarkdownDescription: f.desc, Computed: true, Sensitive: f.sensitive}
	case kInt:
		return dsschema.Int64Attribute{MarkdownDescription: f.desc, Computed: true, Sensitive: f.sensitive}
	case kBool:
		return dsschema.BoolAttribute{MarkdownDescription: f.desc, Computed: true, Sensitive: f.sensitive}
	case kStringList:
		return dsschema.ListAttribute{MarkdownDescription: f.desc, ElementType: types.StringType, Computed: true, Sensitive: f.sensitive}
	case kStringSet:
		return dsschema.SetAttribute{MarkdownDescription: f.desc, ElementType: types.StringType, Computed: true, Sensitive: f.sensitive}
	case kStringMap:
		return dsschema.MapAttribute{MarkdownDescription: f.desc, ElementType: types.StringType, Computed: true, Sensitive: f.sensitive}
	case kBoolMapMap:
		return dsschema.MapAttribute{MarkdownDescription: f.desc, ElementType: types.MapType{ElemType: types.BoolType}, Computed: true, Sensitive: f.sensitive}
	case kObject:
		sub := map[string]dsschema.Attribute{}
		for _, s := range f.sub {
			sub[s.tf] = dataSourceAttribute(s)
		}
		return dsschema.SingleNestedAttribute{MarkdownDescription: f.desc, Attributes: sub, Computed: true, Sensitive: f.sensitive}
	case kObjectList:
		sub := map[string]dsschema.Attribute{}
		for _, s := range f.sub {
			sub[s.tf] = dataSourceAttribute(s)
		}
		return dsschema.ListNestedAttribute{MarkdownDescription: f.desc, NestedObject: dsschema.NestedAttributeObject{Attributes: sub},
			Computed: true, Sensitive: f.sensitive}
	}
	panic("unknown field kind " + f.tf)
}
