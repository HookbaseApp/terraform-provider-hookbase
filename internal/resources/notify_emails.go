package resources

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

// A route's notify_emails is a comma-separated string in configuration. The API stores it as a
// list and answers it joined with ", ", so the text read back differs from the configured text
// whenever the spacing does. State keeps the configured text when it lists the same addresses.

// notifyEmailsToState returns the state value for notify_emails. It is prior when prior lists the
// same addresses as the API, in the same order, read the way the API reads them (entries trimmed,
// empty ones and repeats dropped; see client.ParseEmailList). This way
// "a@example.com,b@example.com" shows no diff against the API's "a@example.com, b@example.com".
// Otherwise it is the API's list joined the way the API joins it, so real drift shows in the next
// plan, or null when the API has no addresses.
func notifyEmailsToState(prior types.String, api client.EmailList) types.String {
	// A prior of "" (or only commas and spaces) lists no addresses, which matches an API with
	// none: the API stores "" as null, and the configured "" is kept rather than failing the apply.
	if !prior.IsNull() && !prior.IsUnknown() && slices.Equal(client.ParseEmailList(prior.ValueString()), api) {
		return prior
	}
	if len(api) == 0 {
		return types.StringNull()
	}
	return types.StringValue(strings.Join(api, ", "))
}

// notifyEmailsBody is the notifyEmails value to send for a configured notify_emails: the entries
// read as client.ParseEmailList reads them, joined with ", ". A value that lists no address sends "", which
// the API stores as no addresses.
func notifyEmailsBody(configured string) *string {
	v := strings.Join(client.ParseEmailList(configured), ", ")
	return &v
}

// emailAddressPattern is a loose check: something@domain.tld, with no spaces. The API applies its
// own, stricter check and answers 400 for an address it refuses; this one only catches the obvious
// mistakes (a semicolon as the separator, a missing @) at plan time, without refusing an address
// the API would take.
var emailAddressPattern = regexp.MustCompile(`^[^\s@]+@[^\s@.]+(\.[^\s@.]+)+$`)

// notifyEmailsValidator checks each comma-separated entry of notify_emails.
type notifyEmailsValidator struct{}

func (notifyEmailsValidator) Description(_ context.Context) string {
	return "each comma-separated entry must be an email address"
}

func (v notifyEmailsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (notifyEmailsValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, entry := range client.ParseEmailList(req.ConfigValue.ValueString()) {
		if !emailAddressPattern.MatchString(entry) {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid email address",
				fmt.Sprintf("%q is not an email address. notify_emails is a comma-separated list of "+
					"addresses, for example \"oncall@example.com, team@example.com\".", entry))
		}
	}
}
