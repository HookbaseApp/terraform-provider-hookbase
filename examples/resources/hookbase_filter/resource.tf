# Filter for high-value transactions only
resource "hookbase_filter" "high_value" {
  name  = "high-value-orders"
  logic = "AND"
  conditions = jsonencode([
    {
      field    = "data.object.amount"
      operator = "greater_than"
      value    = "10000"
    },
    {
      field    = "data.object.currency"
      operator = "equals"
      value    = "usd"
    }
  ])
}
