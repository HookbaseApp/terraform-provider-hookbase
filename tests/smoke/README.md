# Smoke Test

Manual end-to-end test against a running Hookbase API.

## Prerequisites

1. Build and install the provider locally:
   ```bash
   cd ../.. && make install
   ```

2. Start the Hookbase API dev server:
   ```bash
   cd ../../../api && npm run dev
   ```

3. Have an API key and org ID ready.

## Run

```bash
# Set variables
export TF_VAR_api_key="whr_your_api_key_here"
export TF_VAR_org_id="your-org-uuid-here"

# Init, plan, apply
terraform init
terraform plan
terraform apply

# Verify outputs
terraform output

# Clean up
terraform destroy
```

## What it tests

Creates the full inbound + outbound pipeline:
- **Inbound**: source → transform → filter → destination → route
- **Outbound**: event type → application → endpoint → subscription

All 9 resource types are exercised.
