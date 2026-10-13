package provider

// monitor_fields.go — the uptimewarden_monitor attributes, one table.
//
// Every attribute is the snake_case form of a /api/v1/monitors field with the
// same meaning (expected_status_codes is expectedStatusCodes). The table
// drives the resource schema, the data source schema, request bodies and the
// mapping of API responses back into state, so a field is added in one place.

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type fieldKind int

const (
	kString fieldKind = iota
	kInt
	kBool
	kStringList // ordered list of strings
	kStringSet  // unordered set of strings
	kStringMap  // map(string)
	kBoolMapMap // map(map(bool)), e.g. notifications: channel → alert type → on/off
	kObject     // single nested object (sub fields)
	kObjectList // list of nested objects (sub fields)
)

// fieldMode says who sets a field.
type fieldMode int

const (
	// mOptional: the user may set it; when unset, state reads back the API's
	// value (Optional + Computed, prior state kept in plans).
	mOptional fieldMode = iota
	// mRequired: the user must set it.
	mRequired
	// mComputed: only the API sets it.
	mComputed
)

type monitorField struct {
	tf   string // Terraform attribute name
	api  string // API JSON field name
	kind fieldKind
	mode fieldMode
	desc string

	sensitive bool
	// createOnly is false for fields the API accepts only on PATCH (groups):
	// they are set with a follow-up PATCH after the create.
	patchOnly bool
	// emptyIsNull: the API answers null after the empty value was sent
	// ("" detaches a policy, 0 turns a threshold off). The configured empty
	// value is kept in state so the plan stays clean.
	emptyIsNull bool
	// keepWhenNull: the API hides the value from callers who cannot edit;
	// a null answer keeps the value already in state.
	keepWhenNull bool

	stringValidators []validator.String
	int64Validators  []validator.Int64
	listValidators   []validator.List
	setValidators    []validator.Set

	sub []monitorField // kObject / kObjectList
}

var ulidPattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{26}$`)

var monitorTypes = []string{
	"http", "tcp", "udp", "ssl", "websocket", "mcp", "grpc", "ping", "dns", "heartbeat",
}

var regionCodePattern = regexp.MustCompile(`^[a-z0-9]{2,8}$`)

// monitorFields is the attribute table, in schema order.
var monitorFields = []monitorField{
	{tf: "public_id", api: "publicId", kind: kString, mode: mComputed,
		desc: "Public ID of the monitor (ULID). Same value as `id`."},
	{tf: "name", api: "name", kind: kString, mode: mRequired,
		desc: "Human-readable name."},
	{tf: "type", api: "type", kind: kString, mode: mRequired,
		desc: "Protocol type. One of: `http`, `tcp`, `udp`, `ssl`, `websocket`, `mcp`, `grpc`, `ping`, `dns`, `heartbeat`. " +
			"Changing between two non-heartbeat types updates the monitor in place; changing to or from `heartbeat` destroys it and creates a new one.",
		stringValidators: []validator.String{stringvalidator.OneOf(monitorTypes...)}},
	{tf: "url", api: "url", kind: kString, mode: mOptional,
		desc: "Target URL or address. Required for every type except `heartbeat`, where UptimeWarden sets it to `heartbeat://<token>` (do not set it)."},
	{tf: "description", api: "description", kind: kString, mode: mOptional, emptyIsNull: true,
		desc: "Free-text description."},
	{tf: "interval", api: "interval", kind: kInt, mode: mOptional,
		desc:            "Check interval in seconds. At least 30 and at least your plan's minimum. API default: 300.",
		int64Validators: []validator.Int64{int64validator.AtLeast(30)}},
	{tf: "timeout", api: "timeout", kind: kInt, mode: mOptional,
		desc:            "Request timeout in seconds (5–300). API default: 30.",
		int64Validators: []validator.Int64{int64validator.Between(5, 300)}},
	{tf: "is_active", api: "isActive", kind: kBool, mode: mOptional,
		desc: "Whether the monitor is checking. API default: `true`."},
	{tf: "tags", api: "tags", kind: kStringSet, mode: mOptional,
		desc: "Tags."},
	{tf: "groups", api: "groups", kind: kStringSet, mode: mOptional, patchOnly: true,
		desc: "Monitor groups (by name). The API sets groups on update only, so on create the provider sets them right after."},
	{tf: "external_id", api: "externalId", kind: kString, mode: mOptional, emptyIsNull: true,
		desc:             "Your stable handle for the monitor (at most 255 characters). `\"\"` clears it.",
		stringValidators: []validator.String{stringvalidator.LengthAtMost(255)}},
	{tf: "managed_by_source", api: "managedBySource", kind: kString, mode: mComputed,
		desc: "Who created the monitor: `terraform` for monitors this provider created, `manual`, `cli`, …"},

	// Where it is checked from.
	{tf: "region", api: "region", kind: kString, mode: mOptional,
		desc: "Home region code: `ae` (Sydney), `brs` (São Paulo), `gwc` (Sweden Central), `sea` (East Asia), `wus3` (Phoenix). " +
			"Choosing a region needs region selection on your plan; when unset the API places the monitor (`gwc`). " +
			"The `uptimewarden_regions` data source lists the active codes and your plan's allowance.",
		stringValidators: []validator.String{stringvalidator.RegexMatches(regionCodePattern, "must be a region code such as gwc")}},
	{tf: "verification_regions", api: "verificationRegions", kind: kStringList, mode: mOptional,
		desc:           "Regions that recheck a failure before it is alerted: active codes, no repeats, not the home region, at most your plan's verification regions minus one. When unset the API picks them from your plan.",
		listValidators: []validator.List{listvalidator.UniqueValues(), listvalidator.ValueStringsAre(stringvalidator.RegexMatches(regionCodePattern, "must be a region code such as sea"))}},
	{tf: "min_confirming_regions", api: "minConfirmingRegions", kind: kInt, mode: mOptional,
		desc:            "How many regions must see the failure to confirm it. A value other than your plan's default needs a plan that allows it.",
		int64Validators: []validator.Int64{int64validator.AtLeast(1)}},
	{tf: "agent_id", api: "agentId", kind: kString, mode: mOptional,
		desc: "Run the checks from one of your own agents (its id). `\"\"` or `server` means UptimeWarden's regions."},

	// Check settings.
	{tf: "method", api: "method", kind: kString, mode: mOptional,
		desc: "HTTP method."},
	{tf: "request_body", api: "requestBody", kind: kString, mode: mOptional,
		desc: "HTTP request body."},
	{tf: "expected_status_codes", api: "expectedStatusCodes", kind: kString, mode: mOptional,
		desc: "Accepted HTTP status codes, e.g. `200-299` or `200,201`."},
	{tf: "expected_body", api: "expectedBody", kind: kString, mode: mOptional,
		desc: "Text the response body must contain."},
	{tf: "custom_headers", api: "customHeaders", kind: kStringMap, mode: mOptional,
		desc: "Extra request headers."},
	{tf: "content_validation", api: "contentValidation", kind: kObject, mode: mOptional,
		desc: "Response content validation.",
		sub: []monitorField{
			{tf: "enabled", api: "enabled", kind: kBool, mode: mOptional, desc: "Whether content validation runs."},
			{tf: "type", api: "type", kind: kString, mode: mOptional, desc: "`json`, `xml`, `text` or `regex`.",
				stringValidators: []validator.String{stringvalidator.OneOf("json", "xml", "text", "regex")}},
			{tf: "schema", api: "schema", kind: kString, mode: mOptional, desc: "JSON Schema or XML Schema."},
			{tf: "expected_content", api: "expectedContent", kind: kString, mode: mOptional, desc: "Expected text or regex."},
			{tf: "case_sensitive", api: "caseSensitive", kind: kBool, mode: mOptional, desc: "Case-sensitive text/regex match."},
		}},
	{tf: "follow_redirects", api: "followRedirects", kind: kBool, mode: mOptional,
		desc: "Follow HTTP redirects."},
	{tf: "verify_ssl", api: "verifySsl", kind: kBool, mode: mOptional,
		desc: "Verify the TLS certificate."},
	{tf: "enable_ssl", api: "enableSsl", kind: kBool, mode: mOptional,
		desc: "Use TLS (TCP and similar checks)."},
	{tf: "basic_auth_username", api: "basicAuthUsername", kind: kString, mode: mOptional,
		desc: "HTTP basic auth user name."},
	{tf: "basic_auth_password", api: "basicAuthPassword", kind: kString, mode: mOptional, sensitive: true, keepWhenNull: true,
		desc: "HTTP basic auth password."},
	{tf: "bearer_token", api: "bearerToken", kind: kString, mode: mOptional, sensitive: true, keepWhenNull: true,
		desc: "Bearer token sent in the Authorization header."},
	{tf: "hostname", api: "hostname", kind: kString, mode: mOptional,
		desc: "Host name (TCP, UDP, ping and similar checks)."},
	{tf: "port", api: "port", kind: kInt, mode: mOptional,
		desc:            "Port (1–65535).",
		int64Validators: []validator.Int64{int64validator.Between(1, 65535)}},
	{tf: "grpc_proto_file", api: "grpcProtoFile", kind: kString, mode: mOptional,
		desc: "gRPC proto file."},
	{tf: "grpc_method", api: "grpcMethod", kind: kString, mode: mOptional,
		desc: "gRPC method."},
	{tf: "dns_config", api: "dnsConfig", kind: kObject, mode: mOptional,
		desc: "DNS check settings (type `dns`).",
		sub: []monitorField{
			{tf: "record_type", api: "recordType", kind: kString, mode: mOptional, desc: "`A`, `AAAA`, `CNAME`, `MX`, `NS` or `TXT`.",
				stringValidators: []validator.String{stringvalidator.OneOf("A", "AAAA", "CNAME", "MX", "NS", "TXT")}},
			{tf: "resolver", api: "resolver", kind: kString, mode: mOptional, desc: "Resolver to query; the check's default resolver when unset."},
			{tf: "expected_answers", api: "expectedAnswers", kind: kStringList, mode: mOptional, desc: "Values that must all appear in the answer."},
		}},

	// Thresholds.
	{tf: "retry_count", api: "retryCount", kind: kInt, mode: mOptional,
		desc:            "Retries before a failure counts.",
		int64Validators: []validator.Int64{int64validator.AtLeast(0)}},
	{tf: "response_time_threshold", api: "responseTimeThreshold", kind: kInt, mode: mOptional, emptyIsNull: true,
		desc:            "Slow-response alert threshold in milliseconds (50–5000). `0` turns it off.",
		int64Validators: []validator.Int64{int64validator.Any(int64validator.OneOf(0), int64validator.Between(50, 5000))}},
	{tf: "ssl_expiry_days", api: "sslExpiryDays", kind: kInt, mode: mOptional,
		desc:            "Days before certificate expiry to alert. API default: 30.",
		int64Validators: []validator.Int64{int64validator.AtLeast(1)}},
	{tf: "ssl_alerts", api: "sslAlerts", kind: kObject, mode: mOptional,
		desc: "Which certificate-expiry reminders to send.",
		sub: []monitorField{
			{tf: "thirty_days", api: "thirtyDays", kind: kBool, mode: mOptional, desc: "30 days before expiry."},
			{tf: "fifteen_days", api: "fifteenDays", kind: kBool, mode: mOptional, desc: "15 days before expiry."},
			{tf: "seven_days", api: "sevenDays", kind: kBool, mode: mOptional, desc: "7 days before expiry."},
			{tf: "one_day", api: "oneDay", kind: kBool, mode: mOptional, desc: "1 day before expiry."},
		}},
	{tf: "domain_monitoring_enabled", api: "domainMonitoringEnabled", kind: kBool, mode: mOptional,
		desc: "Watch the domain's registration expiry. A plan feature with a count limit."},
	{tf: "domain_expiry_threshold", api: "domainExpiryThreshold", kind: kInt, mode: mOptional,
		desc:            "Days before domain expiry to alert.",
		int64Validators: []validator.Int64{int64validator.AtLeast(1)}},
	{tf: "security_analysis_enabled", api: "securityAnalysisEnabled", kind: kBool, mode: mOptional,
		desc: "Run security analysis. A plan feature."},
	{tf: "security_analysis_mode", api: "securityAnalysisMode", kind: kString, mode: mOptional,
		desc: "Security analysis mode."},

	// Alert routing.
	{tf: "alert_types", api: "alertTypes", kind: kStringSet, mode: mOptional,
		desc: "Alert types to send, e.g. `down`, `recovery`."},
	{tf: "notifications", api: "notifications", kind: kBoolMapMap, mode: mOptional,
		desc: "Channel × alert type switches, e.g. `{ email = { down = true, recovery = true } }`."},
	{tf: "email_addresses", api: "emailAddresses", kind: kString, mode: mOptional,
		desc: "Comma-separated e-mail recipients."},
	{tf: "phone_numbers", api: "phoneNumbers", kind: kString, mode: mOptional,
		desc: "Comma-separated SMS recipients."},
	{tf: "whatsapp_numbers", api: "whatsappNumbers", kind: kString, mode: mOptional,
		desc: "Comma-separated WhatsApp recipients."},
	{tf: "voice_numbers", api: "voiceNumbers", kind: kString, mode: mOptional,
		desc: "Comma-separated voice-call recipients."},
	{tf: "voice_mode", api: "voiceMode", kind: kString, mode: mOptional,
		desc:             "`sequential` (one number at a time, stops when someone answers) or `all`. API default: `sequential`.",
		stringValidators: []validator.String{stringvalidator.OneOf("sequential", "all")}},
	{tf: "escalation_policy_id", api: "escalationPolicyId", kind: kString, mode: mOptional, emptyIsNull: true,
		desc:             "Public id of one of your escalation policies (see the `uptimewarden_escalation_policy` data source). `\"\"` detaches it. On-call escalation is a plan feature.",
		stringValidators: []validator.String{stringvalidator.Any(stringvalidator.OneOf(""), stringvalidator.RegexMatches(ulidPattern, "must be an escalation policy public id (ULID)"))}},
	{tf: "selected_webhook_ids", api: "selectedWebhookIds", kind: kStringSet, mode: mOptional,
		desc:          "Public ids of your webhooks (see the `uptimewarden_webhook` data source). `[]` detaches all.",
		setValidators: []validator.Set{setvalidator.ValueStringsAre(stringvalidator.RegexMatches(ulidPattern, "must be a webhook public id (ULID)"))}},

	// Heartbeat and maintenance.
	{tf: "heartbeat_interval_seconds", api: "heartbeatIntervalSeconds", kind: kInt, mode: mOptional,
		desc:            "Heartbeat monitors: expected seconds between pings (60–2592000). Defaults to `interval`.",
		int64Validators: []validator.Int64{int64validator.Between(60, 2592000)}},
	{tf: "heartbeat_grace_seconds", api: "heartbeatGraceSeconds", kind: kInt, mode: mOptional,
		desc:            "Heartbeat monitors: slack after a missed ping before alerting (0–2592000). API default: 60.",
		int64Validators: []validator.Int64{int64validator.Between(0, 2592000)}},
	{tf: "heartbeat_token", api: "heartbeatToken", kind: kString, mode: mComputed, sensitive: true, keepWhenNull: true,
		desc: "Heartbeat monitors: the ping token."},
	{tf: "heartbeat_url", api: "heartbeatUrl", kind: kString, mode: mComputed, sensitive: true, keepWhenNull: true,
		desc: "Heartbeat monitors: the success ping URL. Replace `/ping` with `/fail` to report a failure."},
	{tf: "maintenance_windows", api: "maintenanceWindows", kind: kObjectList, mode: mOptional,
		desc: "Maintenance windows. Windows opened by deploys (GitHub Actions, CLI) show up here too; a one-off window that has ended is dropped by the API.",
		sub: []monitorField{
			{tf: "id", api: "id", kind: kString, mode: mOptional, desc: "Window id (set by the deploy endpoints)."},
			{tf: "type", api: "type", kind: kString, mode: mOptional, desc: "`recurring` or `once`.",
				stringValidators: []validator.String{stringvalidator.OneOf("recurring", "once")}},
			{tf: "day", api: "day", kind: kString, mode: mOptional, desc: "Recurring windows: weekday, e.g. `monday`."},
			{tf: "start", api: "start", kind: kString, mode: mRequired, desc: "Start: `HH:MM` (recurring) or ISO 8601 (once)."},
			{tf: "end", api: "end", kind: kString, mode: mRequired, desc: "End: `HH:MM` (recurring) or ISO 8601 (once)."},
			{tf: "timezone", api: "timezone", kind: kString, mode: mOptional, desc: "IANA time zone, e.g. `Europe/Istanbul`."},
			{tf: "source", api: "source", kind: kString, mode: mOptional, desc: "Who opened the window."},
			{tf: "reason", api: "reason", kind: kString, mode: mOptional, desc: "Why."},
		}},

	{tf: "created_at", api: "createdAt", kind: kString, mode: mComputed,
		desc: "Creation time (RFC 3339)."},
}

// dataSourceOnlyFields are read-only fields that change on their own; the
// resource leaves them out so state does not churn on every refresh.
var dataSourceOnlyFields = []monitorField{
	{tf: "last_checked", api: "lastChecked", kind: kString, mode: mComputed, desc: "Time of the last check."},
	{tf: "heartbeat_last_ping_at", api: "heartbeatLastPingAt", kind: kString, mode: mComputed, desc: "Heartbeat monitors: time of the last ping."},
}

// attrType is the Terraform type of a field.
func (f monitorField) attrType() attr.Type {
	switch f.kind {
	case kString:
		return types.StringType
	case kInt:
		return types.Int64Type
	case kBool:
		return types.BoolType
	case kStringList:
		return types.ListType{ElemType: types.StringType}
	case kStringSet:
		return types.SetType{ElemType: types.StringType}
	case kStringMap:
		return types.MapType{ElemType: types.StringType}
	case kBoolMapMap:
		return types.MapType{ElemType: types.MapType{ElemType: types.BoolType}}
	case kObject:
		return types.ObjectType{AttrTypes: subAttrTypes(f.sub)}
	case kObjectList:
		return types.ListType{ElemType: types.ObjectType{AttrTypes: subAttrTypes(f.sub)}}
	}
	panic("unknown field kind")
}

func subAttrTypes(fields []monitorField) map[string]attr.Type {
	out := make(map[string]attr.Type, len(fields))
	for _, f := range fields {
		out[f.tf] = f.attrType()
	}
	return out
}
