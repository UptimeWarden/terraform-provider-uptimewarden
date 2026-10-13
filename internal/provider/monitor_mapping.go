package provider

// monitor_mapping.go — Terraform values ⇄ API JSON for the monitor fields.

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ---------------------------------------------------------------------------
// Terraform → API
// ---------------------------------------------------------------------------

// toAPI returns the JSON value of v, or ok=false when v is null or unknown
// (the field is then left out of the request). Inside objects only known,
// non-null attributes are sent.
func toAPI(f monitorField, v attr.Value) (any, bool) {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return nil, false
	}
	switch f.kind {
	case kString:
		return v.(types.String).ValueString(), true
	case kInt:
		return v.(types.Int64).ValueInt64(), true
	case kBool:
		return v.(types.Bool).ValueBool(), true
	case kStringList:
		return stringElems(v.(types.List).Elements()), true
	case kStringSet:
		return stringElems(v.(types.Set).Elements()), true
	case kStringMap:
		out := map[string]string{}
		for k, e := range v.(types.Map).Elements() {
			if s, ok := e.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
				out[k] = s.ValueString()
			}
		}
		return out, true
	case kBoolMapMap:
		out := map[string]map[string]bool{}
		for ch, inner := range v.(types.Map).Elements() {
			m, ok := inner.(types.Map)
			if !ok || m.IsNull() || m.IsUnknown() {
				continue
			}
			row := map[string]bool{}
			for k, e := range m.Elements() {
				if b, ok := e.(types.Bool); ok && !b.IsNull() && !b.IsUnknown() {
					row[k] = b.ValueBool()
				}
			}
			out[ch] = row
		}
		return out, true
	case kObject:
		return objectToAPI(f.sub, v.(types.Object).Attributes()), true
	case kObjectList:
		out := []map[string]any{}
		for _, e := range v.(types.List).Elements() {
			o, ok := e.(types.Object)
			if !ok || o.IsNull() || o.IsUnknown() {
				continue
			}
			out = append(out, objectToAPI(f.sub, o.Attributes()))
		}
		return out, true
	}
	return nil, false
}

func objectToAPI(sub []monitorField, attrs map[string]attr.Value) map[string]any {
	out := map[string]any{}
	for _, s := range sub {
		if val, ok := toAPI(s, attrs[s.tf]); ok {
			out[s.api] = val
		}
	}
	return out
}

func stringElems(elems []attr.Value) []string {
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		if s, ok := e.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			out = append(out, s.ValueString())
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// API → Terraform
// ---------------------------------------------------------------------------

// fromAPI converts an API JSON value into the field's Terraform value.
//
// hint is the value the configuration asked for (on create/update) or the
// prior state (on read); it may be nil. It settles two cases where the API
// answers null for a value Terraform knows:
//   - the empty value was sent ("" clears, 0 turns off, [] detaches): the
//     empty value stays in state;
//   - keepWhenNull fields (secrets the API only shows to editors): the value
//     already known stays.
func fromAPI(f monitorField, v any, hint attr.Value) attr.Value {
	hintKnown := hint != nil && !hint.IsNull() && !hint.IsUnknown()
	if v == nil && hintKnown && (f.keepWhenNull || isEmptyValue(hint)) {
		return hint
	}

	switch f.kind {
	case kString:
		s, ok := v.(string)
		if !ok {
			return types.StringNull()
		}
		return types.StringValue(s)
	case kInt:
		n, ok := v.(float64)
		if !ok || n != math.Trunc(n) {
			return types.Int64Null()
		}
		return types.Int64Value(int64(n))
	case kBool:
		b, ok := v.(bool)
		if !ok {
			return types.BoolNull()
		}
		return types.BoolValue(b)
	case kStringList:
		items, ok := v.([]any)
		if !ok {
			return types.ListNull(types.StringType)
		}
		return types.ListValueMust(types.StringType, stringValues(items))
	case kStringSet:
		items, ok := v.([]any)
		if !ok {
			return types.SetNull(types.StringType)
		}
		return types.SetValueMust(types.StringType, stringValues(items))
	case kStringMap:
		m, ok := v.(map[string]any)
		if !ok {
			return types.MapNull(types.StringType)
		}
		out := map[string]attr.Value{}
		for k, e := range m {
			if s, ok := e.(string); ok {
				out[k] = types.StringValue(s)
			}
		}
		return types.MapValueMust(types.StringType, out)
	case kBoolMapMap:
		inner := types.MapType{ElemType: types.BoolType}
		m, ok := v.(map[string]any)
		if !ok {
			return types.MapNull(inner)
		}
		out := map[string]attr.Value{}
		for ch, rowAny := range m {
			row, ok := rowAny.(map[string]any)
			if !ok {
				continue
			}
			vals := map[string]attr.Value{}
			for k, e := range row {
				// Only the boolean switches; anything else in the stored
				// object is not part of this attribute.
				if b, ok := e.(bool); ok {
					vals[k] = types.BoolValue(b)
				}
			}
			out[ch] = types.MapValueMust(types.BoolType, vals)
		}
		return types.MapValueMust(inner, out)
	case kObject:
		m, ok := v.(map[string]any)
		if !ok {
			return types.ObjectNull(subAttrTypes(f.sub))
		}
		var hintAttrs map[string]attr.Value
		if o, ok := hint.(types.Object); ok && hintKnown {
			hintAttrs = o.Attributes()
		}
		return objectFromAPI(f.sub, m, hintAttrs)
	case kObjectList:
		elemType := types.ObjectType{AttrTypes: subAttrTypes(f.sub)}
		items, ok := v.([]any)
		if !ok {
			return types.ListNull(elemType)
		}
		var hintElems []attr.Value
		if l, ok := hint.(types.List); ok && hintKnown {
			hintElems = l.Elements()
		}
		out := make([]attr.Value, 0, len(items))
		for i, it := range items {
			m, _ := it.(map[string]any)
			var hintAttrs map[string]attr.Value
			if i < len(hintElems) {
				if o, ok := hintElems[i].(types.Object); ok && !o.IsNull() && !o.IsUnknown() {
					hintAttrs = o.Attributes()
				}
			}
			out = append(out, objectFromAPI(f.sub, m, hintAttrs))
		}
		return types.ListValueMust(elemType, out)
	}
	panic("unknown field kind " + f.tf)
}

func objectFromAPI(sub []monitorField, m map[string]any, hint map[string]attr.Value) types.Object {
	attrs := map[string]attr.Value{}
	for _, s := range sub {
		var h attr.Value
		if hint != nil {
			h = hint[s.tf]
		}
		attrs[s.tf] = fromAPI(s, m[s.api], h)
	}
	return types.ObjectValueMust(subAttrTypes(sub), attrs)
}

func stringValues(items []any) []attr.Value {
	out := make([]attr.Value, 0, len(items))
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, types.StringValue(s))
		}
	}
	return out
}

// isEmptyValue reports "", 0, false-free empties: the values a client sends
// to clear a field.
func isEmptyValue(v attr.Value) bool {
	switch t := v.(type) {
	case types.String:
		return t.ValueString() == ""
	case types.Int64:
		return t.ValueInt64() == 0
	case types.List:
		return len(t.Elements()) == 0
	case types.Set:
		return len(t.Elements()) == 0
	case types.Map:
		return len(t.Elements()) == 0
	}
	return false
}

// ---------------------------------------------------------------------------
// Whole-monitor helpers
// ---------------------------------------------------------------------------

// monitorObjectType is the Terraform type of the whole resource.
func monitorObjectType() types.ObjectType {
	t := map[string]attr.Type{"id": types.StringType}
	for _, f := range monitorFields {
		t[f.tf] = f.attrType()
	}
	return types.ObjectType{AttrTypes: t}
}

// monitorStateFromAPI maps an API monitor to resource state attributes.
func monitorStateFromAPI(m map[string]any, hint map[string]attr.Value) map[string]attr.Value {
	out := map[string]attr.Value{}
	for _, f := range monitorFields {
		var h attr.Value
		if hint != nil {
			h = hint[f.tf]
		}
		out[f.tf] = fromAPI(f, m[f.api], h)
	}
	pid, _ := m["publicId"].(string)
	out["id"] = types.StringValue(pid)
	return out
}

// createBody is every configured, writable field.
func createBody(plan map[string]attr.Value) map[string]any {
	body := map[string]any{}
	for _, f := range monitorFields {
		if f.mode == mComputed || f.patchOnly {
			continue
		}
		if v, ok := toAPI(f, plan[f.tf]); ok {
			body[f.api] = v
		}
	}
	return body
}

// updateBody is every writable field whose planned value differs from state.
// The API changes only the fields it is sent.
func updateBody(plan, state map[string]attr.Value) map[string]any {
	body := map[string]any{}
	for _, f := range monitorFields {
		if f.mode == mComputed {
			continue
		}
		p := plan[f.tf]
		if p == nil || p.IsUnknown() || p.IsNull() || p.Equal(state[f.tf]) {
			continue
		}
		if v, ok := toAPI(f, p); ok {
			body[f.api] = v
		}
	}
	return body
}

// unappliedFields compares the known planned values with what the API
// returned and describes each one the API did not take. Unknown parts of the
// plan (computed attributes) are skipped.
func unappliedFields(plan, got map[string]attr.Value) []string {
	var diffs []string
	for _, f := range monitorFields {
		p := plan[f.tf]
		if p == nil || p.IsUnknown() {
			continue
		}
		if !knownPartsEqual(p, got[f.tf]) {
			if f.sensitive {
				diffs = append(diffs, fmt.Sprintf("%s: the API returned a different (sensitive) value", f.tf))
			} else {
				diffs = append(diffs, fmt.Sprintf("%s: configured %s, API returned %s", f.tf, p, got[f.tf]))
			}
		}
	}
	sort.Strings(diffs)
	return diffs
}

// knownPartsEqual compares want and got, treating unknown parts of want as
// matching anything.
func knownPartsEqual(want, got attr.Value) bool {
	if want == nil || want.IsUnknown() {
		return true
	}
	if got == nil {
		return want.IsNull()
	}
	switch w := want.(type) {
	case basetypes.ObjectValue:
		g, ok := got.(basetypes.ObjectValue)
		if !ok || w.IsNull() != g.IsNull() {
			return false
		}
		if w.IsNull() {
			return true
		}
		ga := g.Attributes()
		for k, v := range w.Attributes() {
			if !knownPartsEqual(v, ga[k]) {
				return false
			}
		}
		return true
	case basetypes.ListValue:
		g, ok := got.(basetypes.ListValue)
		if !ok || w.IsNull() != g.IsNull() {
			return false
		}
		we, ge := w.Elements(), g.Elements()
		if len(we) != len(ge) {
			return false
		}
		for i := range we {
			if !knownPartsEqual(we[i], ge[i]) {
				return false
			}
		}
		return true
	}
	return want.Equal(got)
}

func joinDiffs(diffs []string) string {
	return "  - " + strings.Join(diffs, "\n  - ")
}
