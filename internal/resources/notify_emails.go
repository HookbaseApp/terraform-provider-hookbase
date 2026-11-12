package resources

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf16"

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

// The API's limits (MAX_NOTIFY_EMAILS and MAX_EMAIL_LENGTH in api/src/utils/notify-emails.ts).
const (
	// maxNotifyEmails is the most addresses a route may list, counted after repeats are dropped.
	maxNotifyEmails = 20
	// maxEmailLength is the RFC 5321 path limit, in UTF-16 code units as JavaScript counts length.
	maxEmailLength = 254
)

// jsLength is len(s) as JavaScript counts it: UTF-16 code units.
func jsLength(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

// notifyEmailsValidator checks notify_emails as the API will: at most maxNotifyEmails addresses,
// each an email address of at most maxEmailLength characters.
type notifyEmailsValidator struct{}

func (notifyEmailsValidator) Description(_ context.Context) string {
	return fmt.Sprintf("at most %d comma-separated entries, each an email address of at most %d characters",
		maxNotifyEmails, maxEmailLength)
}

func (v notifyEmailsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (notifyEmailsValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	entries := client.ParseEmailList(req.ConfigValue.ValueString())
	if len(entries) > maxNotifyEmails {
		resp.Diagnostics.AddAttributeError(req.Path, "Too many email addresses",
			fmt.Sprintf("notify_emails lists %d addresses; a route can notify at most %d. An address "+
				"listed twice (in any letter case) counts once.", len(entries), maxNotifyEmails))
	}
	for _, entry := range entries {
		if n := jsLength(entry); n > maxEmailLength {
			shown := entry
			if r := []rune(shown); len(r) > 64 {
				shown = string(r[:64]) + "..."
			}
			resp.Diagnostics.AddAttributeError(req.Path, "Email address too long",
				fmt.Sprintf("%q is %d characters long; an address can be at most %d.", shown, n, maxEmailLength))
			continue
		}
		if !emailAddressPattern.MatchString(entry) {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid email address",
				fmt.Sprintf("%q is not an email address. notify_emails is a comma-separated list of "+
					"addresses, for example \"oncall@example.com, team@example.com\".", entry))
		}
	}
}
