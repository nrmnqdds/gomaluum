# Migration Scripts

This directory contains scripts to help migrate analytics data from the old LibSQL/SQLite storage to the new PostgreSQL storage.

## 🚀 Quick Start

### Option 1: Using the Shell Script (Recommended)

```bash
# Make the script executable (if not already)
chmod +x scripts/migrate.sh

# Run migration with command line arguments
./scripts/migrate.sh -s "file:old_analytics.db" -t "postgresql://user:pass@localhost:5432/gomaluum"

# Or use environment variables
export OLD_DB_PATH="file:old_analytics.db"
export DATABASE_URL="postgresql://user:pass@localhost:5432/gomaluum"
./scripts/migrate.sh

# Dry run to see what would be migrated
./scripts/migrate.sh -s "file:old_analytics.db" -t "postgresql://user:pass@localhost:5432/gomaluum" --dry-run
```

### Option 2: Using the Go Script Directly

```bash
# From the project root directory
go run scripts/migrate_analytics.go -source "file:old_analytics.db" -target "postgresql://user:pass@localhost:5432/gomaluum"

# Dry run
go run scripts/migrate_analytics.go -source "file:old_analytics.db" -target "postgresql://user:pass@localhost:5432/gomaluum" -dry-run
```

## 📁 Files

- **`migrate.sh`** - Shell script wrapper with better UX and error handling
- **`migrate_analytics.go`** - Core migration logic in Go
- **`README.md`** - This documentation

## 🔧 Prerequisites

- Go 1.23 or higher
- Access to your old LibSQL/SQLite database
- PostgreSQL database set up and accessible
- Both database drivers (installed via `go mod tidy`)

## 📋 Supported Source Formats

### Local SQLite Files
```bash
# Absolute path
-s "file:/path/to/analytics.db"

# Relative path
-s "file:./data/analytics.db"
```

### Turso (LibSQL)
```bash
# With auth token
-s "libsql://your-database.turso.io?authToken=your_auth_token"
```

### In-Memory SQLite
```bash
# For testing purposes
-s ":memory:"
```

## 🐘 PostgreSQL Connection String Format

```
postgresql://[user[:password]@][host][:port][/dbname][?param1=value1&...]
```

### Examples:
```bash
# Local development
postgresql://postgres:password@localhost:5432/gomaluum

# Production with SSL
postgresql://user:pass@prod-db.example.com:5432/gomaluum?sslmode=require

# With connection pool settings
postgresql://user:pass@localhost:5432/gomaluum?pool_max_conns=10&pool_min_conns=2
```

## 🔄 Migration Process

The migration script will:

1. **Connect** to both source and target databases
2. **Create schema** in PostgreSQL (if it doesn't exist)
3. **Read data** from the LibSQL/SQLite analytics table
4. **Transform data** to match PostgreSQL schema
5. **Insert records** using `ON CONFLICT DO NOTHING` (no duplicates)
6. **Report results** showing migrated vs skipped records

### Data Transformation

The script handles these differences between SQLite and PostgreSQL:

- **Timestamps**: Parses various SQLite timestamp formats to PostgreSQL TIMESTAMPTZ
- **Generated Columns**: PostgreSQL automatically calculates `batch` and `level` from `matric_no`
- **Conflict Handling**: Uses `ON CONFLICT (matric_no) DO NOTHING` to avoid duplicates

## 📊 What Gets Migrated

From the LibSQL/SQLite `analytics` table:
- `matric_no` → PostgreSQL `matric_no` (PRIMARY KEY)
- `timestamp` → PostgreSQL `timestamp` (converted to TIMESTAMPTZ)
- `batch` and `level` are automatically generated in PostgreSQL

## 🛡️ Safety Features

### Dry Run Mode
Use `-d` or `--dry-run` to see what would be migrated without making changes:

```bash
./scripts/migrate.sh -s "file:old.db" -t "postgresql://user:pass@host/db" --dry-run
```

### Duplicate Handling
The migration uses `ON CONFLICT DO NOTHING`, so:
- ✅ Safe to run multiple times
- ✅ Won't overwrite existing data
- ✅ Only new records are inserted

### Connection Validation
- Tests database connections before starting
- Validates connection string formats
- Provides helpful error messages

## 📝 Usage Examples

### Migrate from Local SQLite
```bash
./scripts/migrate.sh \
  -s "file:./data/analytics.db" \
  -t "postgresql://postgres:password@localhost:5432/gomaluum"
```

### Migrate from Turso to Production
```bash
./scripts/migrate.sh \
  -s "libsql://myapp-db.turso.io?authToken=eyJ..." \
  -t "postgresql://user:pass@prod-db.amazonaws.com:5432/gomaluum?sslmode=require"
```

### Using Environment Variables
```bash
# Set environment variables
export OLD_DB_PATH="libsql://old-db.turso.io?authToken=token"
export DATABASE_URL="postgresql://user:pass@localhost:5432/gomaluum"

# Run migration
./scripts/migrate.sh

# Or dry run
./scripts/migrate.sh --dry-run
```

### Check What Would Be Migrated
```bash
# See migration plan without changes
./scripts/migrate.sh \
  -s "file:analytics.db" \
  -t "postgresql://user:pass@host/db" \
  --dry-run
```

## 🚨 Troubleshooting

### Common Issues

1. **"Failed to connect to source database"**
   - Check if the SQLite file exists and is readable
   - Verify Turso auth token is valid and not expired
   - Ensure the connection string format is correct

2. **"Failed to connect to target database"**
   - Verify PostgreSQL is running and accessible
   - Check credentials and database name
   - Ensure the user has CREATE TABLE permissions

3. **"go.mod not found"**
   - Run the script from the project root directory
   - Ensure you're in the correct gomaluum project folder

4. **"No data to migrate"**
   - Check if the source database has an `analytics` table
   - Verify the table has data
   - Check connection permissions

### Getting Help

```bash
# Show detailed help
./scripts/migrate.sh --help

# Show Go script help
go run scripts/migrate_analytics.go -help
```

## 🔍 Verification

After migration, you can verify the data was transferred correctly:

```sql
-- Check total record count
SELECT COUNT(*) FROM analytics;

-- Check batch and level distribution
SELECT level, batch, COUNT(*) as count 
FROM analytics 
GROUP BY level, batch 
ORDER BY level, batch;

-- Check recent records
SELECT matric_no, batch, level, timestamp 
FROM analytics 
ORDER BY timestamp DESC 
LIMIT 10;
```

## 🔒 Security Considerations

- **Connection Strings**: Contain sensitive credentials - don't commit them to version control
- **Environment Variables**: Use `.env` files or secure environment variable management
- **Database Permissions**: Migration user only needs INSERT permissions, not full admin access
- **Dry Run First**: Always test with `--dry-run` before actual migration

## 📞 Support

If you encounter issues:
1. Check the troubleshooting section above
2. Run with `--dry-run` to diagnose issues
3. Verify your connection strings and permissions
4. Check the application logs for detailed error messages