package resources

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Helpers for String attributes that hold JSON (a destination's config, a route's
// filter_conditions and expected_response). The API hands these back re-encoded: keys in another
// order (a jsonb column sorts them), different whitespace, or the whole value as its stored JSON
// text, i.e. a JSON string. Comparing the text would show a diff on every plan, so state keeps
// the configured text whenever it says the same thing as the API's value.

// redactedPrefix starts a credential the API masks in a queue or storage destination's config
// (redactWarehouseConfig in api/src/routes/destinations.ts). The four characters after it come
// from the stored ciphertext, not the secret, so they can't be checked against configuration.
const redactedPrefix = "••••"

// unwrapJSONText returns the JSON text that a JSON string holds, and raw otherwise. The API returns
// some values (a route's filter_conditions) as their stored JSON text, i.e. a JSON string, and
// provider 0.6.x stored that verbatim. A string holding nothing unwraps to null.
func unwrapJSONText(raw []byte) []byte {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '"' {
		return raw
	}
	var inner string
	if json.Unmarshal(raw, &inner) != nil {
		return raw
	}
	inner = strings.TrimSpace(inner)
	switch {
	case inner == "":
		return []byte("null")
	case json.Valid([]byte(inner)):
		return []byte(inner)
	}
	return raw
}

// jsonText returns the API's value as compact JSON text, and false when it has none (absent or
// null). A value sent as its stored JSON text (a JSON string holding JSON) is unwrapped first.
func jsonText(raw json.RawMessage) (string, bool) {
	raw = unwrapJSONText(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		return string(raw), true
	}
	return out.String(), true
}

// jsonToState returns the state value for a JSON attribute the API reports as raw: prior when it
// is equivalent to the API's value (so a refresh shows no diff for a value the API only
// re-encoded), the API's value otherwise (so real drift shows in the next plan), and null when the
// API has none. With redacted set, a masked credential in the API's value matches any non-empty
// string in prior.
func jsonToState(prior types.String, raw json.RawMessage, redacted bool) types.String {
	api, ok := jsonText(raw)
	if !ok {
		// The API stores an empty list or object as nothing (route filter conditions, for one), so
		// a configured [] or {} reads back as null; keep it rather than fail the apply.
		if !prior.IsNull() && !prior.IsUnknown() && jsonEmptyContainer(prior.ValueString()) {
			return prior
		}
		return types.StringNull()
	}
	if prior.IsNull() || prior.IsUnknown() || !jsonEquivalent(prior.ValueString(), api, redacted) {
		return types.StringValue(api)
	}
	// State written by provider 0.6.x can hold the API's value verbatim, which for a route's
	// filter_conditions is a JSON string holding the JSON. No configuration says that, so the
	// value is stored the way jsonencode writes it, which a jsonencode() configuration matches
	// without a diff.
	if text := strings.TrimSpace(prior.ValueString()); strings.HasPrefix(text, `"`) {
		if canonical, ok := jsonencodeText(unwrapJSONText([]byte(text))); ok {
			return types.StringValue(canonical)
		}
	}
	return prior
}

// jsonencodeText returns JSON text as Terraform's jsonencode writes it: compact, object keys
// sorted, and <, >, & escaped. Numbers keep their text.
func jsonencodeText(text []byte) (string, bool) {
	dec := json.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil {
		return "", false
	}
	out, err := json.Marshal(v)
	if err != nil {
		return "", false
	}
	return string(out), true
}

// jsonEquivalent reports whether two JSON texts hold the same value, ignoring key order,
// whitespace and number formatting (1 and 1.0). A prior holding its JSON as a JSON string (state
// from provider 0.6.x) is unwrapped first, as the API's value is. See jsonToState for redacted.
func jsonEquivalent(prior, api string, redacted bool) bool {
	var p, a any
	if json.Unmarshal(unwrapJSONText([]byte(prior)), &p) != nil || json.Unmarshal([]byte(api), &a) != nil {
		return false
	}
	return jsonValuesMatch(p, a, redacted)
}

func jsonValuesMatch(prior, api any, redacted bool) bool {
	switch a := api.(type) {
	case map[string]any:
		p, ok := prior.(map[string]any)
		if !ok || len(p) != len(a) {
			return false
		}
		for k, av := range a {
			pv, ok := p[k]
			if !ok || !jsonValuesMatch(pv, av, redacted) {
				return false
			}
		}
		return true
	case []any:
		p, ok := prior.([]any)
		if !ok || len(p) != len(a) {
			return false
		}
		for i := range a {
			if !jsonValuesMatch(p[i], a[i], redacted) {
				return false
			}
		}
		return true
	case string:
		if redacted && strings.HasPrefix(a, redactedPrefix) {
			p, ok := prior.(string)
			return ok && p != ""
		}
		return prior == api
	default:
		// float64, bool or nil: comparable as interface values.
		return prior == api
	}
}

// jsonEmptyContainer reports whether text is the JSON [] or {}.
func jsonEmptyContainer(text string) bool {
	var v any
	if json.Unmarshal([]byte(text), &v) != nil {
		return false
	}
	switch x := v.(type) {
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}
