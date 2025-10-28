# Fake Data Seeder

This script generates realistic fake data for testing the admin dashboard features of Logan's 3D Creations.

## What It Creates

### Users (25)
- **VIP Users**: Users with 5+ orders or $500+ lifetime spend (shows ⭐ VIP badge)
- **New Users**: Users registered within the last 30 days (shows 🆕 New badge)
- **Inactive Users**: Users with no activity in 90+ days (shows 💤 Inactive badge)
- **Regular Users**: Active users with various activity levels

### Orders (45)
- Distributed across users to create realistic spending patterns
- Various statuses: received, in_production, shipped, delivered, cancelled
- Orders spanning the last 6 months
- 1-4 items per order
- Realistic pricing and shipping costs

### Shopping Carts
- **Active Carts (8)**: Updated within the last 7 days
- **Abandoned Carts (15)**: Abandoned 8-60 days ago
- Mix of guest (session-based) and logged-in user carts
- 1-5 items per cart

### Contact Requests (20)
- Different statuses: new, in_progress, responded, resolved, spam
- Different priorities: low, normal, high, urgent
- Spanning the last 3 months
- Realistic subjects and messages

### User Engagement
- **Favorites**: Products saved by users
- **Collections**: User-created collections (some with quote requests)
- 2-6 items per collection

## Prerequisites

The database must already have products seeded. If you don't have products, the script will exit with an error.

## Usage

### Quick Run (Recommended)

From the **project root** directory:

```bash
# Run directly
go run scripts/seed-fake-data/main.go

# Or with explicit database path
DB_PATH=./data/database.db go run scripts/seed-fake-data/main.go
```

### Build and Run

From the **project root** directory:

```bash
cd scripts/seed-fake-data
go build -o seed
cd ../..
./scripts/seed-fake-data/seed
```

Or build and run with proper paths:

```bash
cd scripts/seed-fake-data
go build -o seed
DB_PATH=../../data/database.db ./seed
```

### Custom Database Path

By default, the script uses `./data/database.db` (relative to where you run it). To use a different database:

```bash
DB_PATH=/path/to/your/database.db go run scripts/seed-fake-data/main.go
```

**Note**: The database path is relative to where you execute the command, so run from the project root or use an absolute path.

## What Happens

1. **Clears existing fake data** (preserves admin users and products)
2. **Creates users** with different activity patterns
3. **Creates orders** distributed to create VIP users
4. **Creates cart items** (both active and abandoned)
5. **Creates contact requests** with various statuses
6. **Creates user favorites** for selected users
7. **Creates user collections** with items
8. **Prints summary** showing what was created

## Safety

- ✅ **Idempotent**: Safe to run multiple times - clears fake data before re-seeding
- ✅ **Preserves real data**: Only deletes non-admin users and related test data
- ✅ **Preserves products**: Uses existing products, doesn't modify them
- ✅ **Preserves admin users**: Admin users are never deleted

## Testing Dashboard Features

After running the seed script, you can test these admin features:

### Users Management (`/admin/users`)
- View user list with segment badges (VIP, New, Inactive)
- Search and filter users
- View lifetime spend statistics
- Click into user detail pages

### User Detail Pages (`/admin/users/:id`)
- See comprehensive user information
- View order history
- See cart activity (active and abandoned)
- View favorites and collections

### Dashboard (`/admin`)
- Recent orders
- User activity metrics
- Revenue charts

### Abandoned Carts (`/admin/abandoned-carts`)
- View abandoned cart list
- See cart value and item counts
- View abandonment timelines
- Test recovery email workflows

### Contact Requests (`/admin/contacts`)
- View requests by status
- Filter by priority
- Test status updates

## Customization

Edit the constants at the top of `main.go` to adjust the amount of data generated:

```go
const (
    numUsers           = 25  // Number of users to create
    numOrders          = 45  // Number of orders to create
    numActiveCarts     = 8   // Number of active carts
    numAbandonedCarts  = 15  // Number of abandoned carts
    numContactRequests = 20  // Number of contact requests
    numFavoritesPerUser = 3  // Max favorites per user
    numCollections     = 10  // Number of collections
)
```

## Troubleshooting

### "No products found in database"

You need to seed products first. The script requires existing products to create realistic orders and carts.

### Database locked

Make sure the development server (`make dev`) is not running, or close other database connections.

### Import errors

Run `go mod tidy` to download dependencies:

```bash
cd scripts/seed-fake-data
go mod tidy
```

## Example Output

```
🌱 Starting database seeding...

✓ Found 54 products in database

🧹 Clearing existing fake data...
✓ Cleared existing fake data

👥 Creating users...
✓ Created 25 users
📦 Creating orders...
✓ Created 45 orders
🛒 Creating active carts...
✓ Created 8 active carts
🛒 Creating abandoned carts...
✓ Created 15 abandoned carts
📧 Creating contact requests...
✓ Created 20 contact requests
⭐ Creating user favorites...
✓ Created 42 favorites
📚 Creating user collections...
✓ Created 10 collections

✅ Database seeding completed!

📊 Summary:

  Users:              25 (3 VIP, 5 New)
  Orders:             45 (87 items)
  Active Carts:       8
  Abandoned Carts:    15
  Contact Requests:   20
  Favorites:          42
  Collections:        10

🎉 You can now view the admin dashboard with realistic data!
   Navigate to: http://localhost:8000/admin
```
