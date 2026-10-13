package provider_test

// fake_api_test.go — an in-memory /api/v1 that follows the published contract
// closely enough to catch mapping mistakes: strict field names (an unknown
// field is a 400 like the real API), server defaults on create, PATCH
// semantics (only sent fields change; "" / null / 0 / [] clear the fields
// that the contract says they clear), heartbeat tokens, the reference lists
// and the v1 error envelope.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

var createFields = map[string]bool{}
var patchFields = map[string]bool{}

func init() {
	for _, f := range strings.Fields(`name type url description interval timeout isActive tags externalId managedBySource
		region verificationRegions minConfirmingRegions agentId
		method requestBody expectedStatusCodes expectedBody customHeaders contentValidation followRedirects verifySsl enableSsl
		basicAuthUsername basicAuthPassword bearerToken hostname port grpcProtoFile grpcMethod dnsConfig
		retryCount responseTimeThreshold sslExpiryDays sslAlerts domainMonitoringEnabled domainExpiryThreshold
		securityAnalysisEnabled securityAnalysisMode
		alertTypes notifications emailAddresses phoneNumbers whatsappNumbers voiceNumbers voiceMode escalationPolicyId selectedWebhookIds
		heartbeatIntervalSeconds heartbeatGraceSeconds maintenanceWindows`) {
		createFields[f] = true
		patchFields[f] = true
	}
	patchFields["groups"] = true
}

const (
	policyID  = "01HWPXSY000000000000000001"
	policyID2 = "01HWPXSY000000000000000002"
	hookID    = "01HWHKWB000000000000000001"
	hookID2   = "01HWHKWB000000000000000002"
)

type fakeAPI struct {
	mu       sync.Mutex
	t        *testing.T
	monitors map[string]map[string]any
	next     int
	// ignore lists fields the fake accepts but does not store, to test the
	// provider's "did not apply" error.
	ignore map[string]bool
	posts  []map[string]any
	patch  []map[string]any
	dels   []string
}

func newFakeAPI(t *testing.T) (*fakeAPI, *httptest.Server) {
	t.Helper()
	api := &fakeAPI{t: t, monitors: map[string]map[string]any{}, ignore: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/monitors", api.create)
	mux.HandleFunc("GET /api/v1/monitors/{id}", api.get)
	mux.HandleFunc("PATCH /api/v1/monitors/{id}", api.update)
	mux.HandleFunc("DELETE /api/v1/monitors/{id}", api.remove)
	mux.HandleFunc("GET /api/v1/escalation-policies", func(w http.ResponseWriter, r *http.Request) {
		writeData(w, 200, []map[string]any{
			{"publicId": policyID, "name": "P1 on-call", "stepCount": 2, "createdAt": "2026-10-01T00:00:00.000Z", "updatedAt": "2026-10-02T00:00:00.000Z"},
			{"publicId": policyID2, "name": "Business hours", "stepCount": 1, "createdAt": "2026-10-01T00:00:00.000Z", "updatedAt": "2026-10-01T00:00:00.000Z"},
		})
	})
	mux.HandleFunc("GET /api/v1/webhooks", func(w http.ResponseWriter, r *http.Request) {
		writeData(w, 200, []map[string]any{
			{"publicId": hookID, "name": "Slack #ops", "description": nil, "messageFormat": "slack", "isActive": true,
				"enabledAlertTypes": []string{"down", "recovery"}, "enabledMonitorTypes": []string{}, "createdAt": "2026-10-01T00:00:00.000Z"},
			{"publicId": hookID2, "name": "PagerDuty", "description": "pd", "messageFormat": nil, "isActive": false,
				"enabledAlertTypes": []string{"down"}, "enabledMonitorTypes": []string{"http"}, "createdAt": "2026-10-01T00:00:00.000Z"},
		})
	})
	mux.HandleFunc("GET /api/v1/regions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"data":[
			{"code":"ae","name":"Sydney","description":"Australia","isDefault":false},
			{"code":"gwc","name":"Sweden Central","description":"Europe","isDefault":true},
			{"code":"sea","name":"East Asia","description":null,"isDefault":false}],
			"plan":{"regionSelection":true,"maxRegions":5,"maxVerificationRegions":-1,"defaultConfirmingRegions":3,"allowConsensusOverride":true}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return api, srv
}

func writeData(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
}

func writeErr(w http.ResponseWriter, status int, code, msg string, details ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := map[string]any{"code": code, "message": msg}
	if len(details) > 0 {
		body["details"] = details
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "error": body})
}

func (a *fakeAPI) decode(w http.ResponseWriter, r *http.Request, allowed map[string]bool) (map[string]any, bool) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, 400, "VALIDATION_ERROR", "request body must be a JSON object")
		return nil, false
	}
	for k := range body {
		if !allowed[k] {
			writeErr(w, 400, "VALIDATION_ERROR", fmt.Sprintf("unknown field %q", k))
			return nil, false
		}
	}
	if reg, ok := body["region"].(string); ok && reg != "" && reg != "ae" && reg != "gwc" && reg != "sea" {
		writeErr(w, 400, "VALIDATION_ERROR", "Invalid monitoring region", "Valid regions: ae, gwc, sea")
		return nil, false
	}
	return body, true
}

func (a *fakeAPI) create(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	body, ok := a.decode(w, r, createFields)
	if !ok {
		return
	}
	a.posts = append(a.posts, body)
	if body["type"] != "heartbeat" && body["url"] == nil {
		writeErr(w, 400, "VALIDATION_ERROR", "url is required")
		return
	}
	a.next++
	id := fmt.Sprintf("01HWTEST%018d", a.next)
	m := map[string]any{
		"publicId": id, "description": nil, "interval": 300.0, "timeout": 30.0, "isActive": true,
		"tags": []any{}, "groups": []any{}, "externalId": nil, "managedBySource": "manual",
		"region": "gwc", "verificationRegions": []any{"sea"}, "minConfirmingRegions": 2.0, "agentId": nil,
		"method": "GET", "requestBody": nil, "expectedStatusCodes": "200", "expectedBody": nil,
		"customHeaders": map[string]any{}, "contentValidation": nil, "followRedirects": true, "verifySsl": true,
		"enableSsl": nil, "basicAuthUsername": nil, "basicAuthPassword": nil, "bearerToken": nil,
		"hostname": nil, "port": nil, "grpcProtoFile": nil, "grpcMethod": nil, "dnsConfig": nil,
		"retryCount": 1.0, "responseTimeThreshold": nil, "sslExpiryDays": 30.0,
		"sslAlerts":               map[string]any{"thirtyDays": true, "fifteenDays": true, "sevenDays": true, "oneDay": true},
		"domainMonitoringEnabled": false, "domainExpiryThreshold": 30.0, "securityAnalysisEnabled": false,
		"securityAnalysisMode": nil, "alertTypes": []any{"down", "recovery"},
		"notifications":  map[string]any{"email": map[string]any{"down": true, "recovery": true}, "sms": map[string]any{"down": false}},
		"emailAddresses": "", "phoneNumbers": "", "whatsappNumbers": "", "voiceNumbers": "", "voiceMode": "sequential",
		"escalationPolicyId": nil, "selectedWebhookIds": []any{}, "maintenanceWindows": []any{},
		"heartbeatToken": nil, "heartbeatUrl": nil, "heartbeatIntervalSeconds": nil, "heartbeatGraceSeconds": nil,
		"heartbeatLastPingAt": nil, "createdAt": "2026-10-13T10:00:00.000Z", "lastChecked": nil,
	}
	for k, v := range body {
		if !a.ignore[k] {
			m[k] = v
		}
	}
	if m["type"] == "heartbeat" {
		tok := "hbtok" + id[len(id)-6:]
		m["url"] = "heartbeat://" + tok
		m["heartbeatToken"] = tok
		m["heartbeatUrl"] = "https://app.example/api/heartbeat/" + tok + "/ping"
		if m["heartbeatIntervalSeconds"] == nil {
			m["heartbeatIntervalSeconds"] = m["interval"]
		}
		if m["heartbeatGraceSeconds"] == nil {
			m["heartbeatGraceSeconds"] = 60.0
		}
	}
	a.normalise(m)
	a.monitors[id] = m
	writeData(w, http.StatusCreated, m)
}

// normalise applies the clearing rules of the contract.
func (a *fakeAPI) normalise(m map[string]any) {
	if m["escalationPolicyId"] == "" {
		m["escalationPolicyId"] = nil
	}
	if n, ok := m["responseTimeThreshold"].(float64); ok && n == 0 {
		m["responseTimeThreshold"] = nil
	}
	if m["externalId"] == "" {
		m["externalId"] = nil
	}
	if m["description"] == "" {
		m["description"] = nil
	}
}

func (a *fakeAPI) get(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	m, ok := a.monitors[r.PathValue("id")]
	if !ok {
		writeErr(w, 404, "MONITOR_NOT_FOUND", "Monitor not found")
		return
	}
	writeData(w, 200, m)
}

func (a *fakeAPI) update(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	m, ok := a.monitors[r.PathValue("id")]
	if !ok {
		writeErr(w, 404, "MONITOR_NOT_FOUND", "Monitor not found")
		return
	}
	body, ok := a.decode(w, r, patchFields)
	if !ok {
		return
	}
	a.patch = append(a.patch, body)
	if t, ok := body["type"].(string); ok && t != m["type"] && (t == "heartbeat" || m["type"] == "heartbeat") {
		writeErr(w, 400, "VALIDATION_ERROR", "type cannot change to or from heartbeat")
		return
	}
	for k, v := range body {
		if a.ignore[k] {
			continue
		}
		if v == nil && k != "escalationPolicyId" && k != "responseTimeThreshold" {
			continue // null = no change
		}
		m[k] = v
	}
	a.normalise(m)
	writeData(w, 200, m)
}

func (a *fakeAPI) remove(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	id := r.PathValue("id")
	if _, ok := a.monitors[id]; !ok {
		writeErr(w, 404, "MONITOR_NOT_FOUND", "Monitor not found")
		return
	}
	delete(a.monitors, id)
	a.dels = append(a.dels, id)
	writeData(w, 200, map[string]any{"message": "Monitor deleted"})
}
