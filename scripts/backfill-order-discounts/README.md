# Backfill Order Discounts Script

This script backfills discount and promotion code information for existing orders by querying the Stripe API for historical checkout session data.

## Purpose

When promotion codes are used during Stripe Checkout, the discount is applied but wasn't previously being stored in the orders table. This script:

1. Queries all orders with a `stripe_checkout_session_id`
2. Retrieves the full Stripe checkout session details for each order
3. Extracts discount amount and promotion code information
4. Updates the order with:
   - `original_subtotal_cents` - The subtotal before discount
   - `discount_cents` - The discount amount applied
   - `promotion_code` - The promotion code used (if any)
   - `promotion_code_id` - Link to promotion_codes table (if code exists in our system)

## Prerequisites

- Stripe secret key (from `.envrc` or provided via flag)
- Access to the production/staging database
- Go 1.21 or later

## Usage

### Dry Run (Recommended First)

Always run in dry-run mode first to see what would be updated without making changes:

```bash
cd scripts/backfill-order-discounts
go run main.go --dry-run
```

### Production Run

After verifying the dry run output, run without --dry-run to update the database:

```bash
go run main.go
```

### Custom Options

```bash
# Use a different database file
go run main.go --db /path/to/database.db

# Provide Stripe key explicitly
go run main.go --stripe-key sk_live_xxxxx

# Dry run with custom database
go run main.go --db ./data/database.db --dry-run
```

## Rate Limiting

The script includes automatic rate limiting:
- 100ms delay between successful Stripe API calls
- 500ms delay after errors to prevent rate limit issues

For large numbers of orders, the script may take some time to complete. Stripe's rate limit is 100 requests per second, so expect approximately:
- 10 orders per second
- 600 orders per minute
- 36,000 orders per hour

## Output

The script provides detailed logging:
- `INFO` - Progress updates and successful operations
- `DEBUG` - Detailed information about each order (visible with slog debug level)
- `ERROR` - Failed operations

Example output:
```
INFO starting backfill total_orders=150 dry_run=true
INFO processing order order_id=abc123 session_id=cs_test_xxx progress=1/150
INFO found discount for order order_id=abc123 discount_cents=1500 promotion_code=SAVE15
INFO backfill complete total_orders=150 processed=150 updated=25 skipped=125 errors=0
INFO DRY RUN - no changes were made to the database
```

## Safety Features

1. **Dry Run Mode**: Test the script without making changes
2. **Idempotent**: Safe to run multiple times - skips orders that already have discount data
3. **Error Handling**: Continues processing even if individual orders fail
4. **Rate Limiting**: Built-in delays to respect Stripe API limits
5. **Detailed Logging**: Full visibility into what the script is doing

## Troubleshooting

### "Failed to retrieve stripe session" errors

This is normal for orders where:
- The Stripe session has expired (sessions expire after 24 hours)
- The session was created in a different Stripe account
- The session ID is invalid

The script will log these errors and continue processing other orders.

### Rate limit errors

If you see rate limit errors from Stripe:
- The script should handle these automatically with delays
- If issues persist, you can manually increase the delay in the code
- Consider running the script during off-peak hours

### No orders updated

If no orders are updated:
- Check that orders have `stripe_checkout_session_id` set
- Verify the Stripe secret key is correct and has access to the sessions
- Check that orders don't already have discount data (script skips these)
- Run with `--dry-run` to see detailed logging

## Database Schema

The script updates these columns in the `orders` table:
- `original_subtotal_cents` (INTEGER) - Subtotal before discount
- `discount_cents` (INTEGER) - Discount amount in cents
- `promotion_code` (TEXT) - Promotion code used
- `promotion_code_id` (TEXT) - Foreign key to promotion_codes table
- `updated_at` (DATETIME) - Automatically updated

## Notes

- Orders without a Stripe checkout session ID are skipped
- Orders that already have discount data are skipped
- Promotion codes are matched to the `promotion_codes` table when possible
- The original subtotal is calculated as: `subtotal_cents + discount_cents`
