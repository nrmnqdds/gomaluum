# Migration from LibSQL/SQLite to PostgreSQL

This document summarizes the migration of GoMa'luum's analytics storage from LibSQL/SQLite to PostgreSQL.

## 🎯 Summary of Changes

GoMa'luum has been updated to use PostgreSQL for analytics storage while maintaining backward compatibility. **The database is now optional** - the application runs perfectly without it, but analytics functionality will be disabled.

### Key Changes Made

1. **Database Driver**: Replaced `github.com/tursodatabase/libsql-client-go` with `github.com/lib/pq`
2. **Environment Variable**: Changed from `DB_PATH` to `DATABASE_URL`
3. **SQL Compatibility**: Updated queries and schema for PostgreSQL
4. **Optional Database**: App no longer panics when database is unavailable
5. **Migration Tools**: Added automated scripts to transfer existing data

## 🔧 Technical Changes

### Dependencies
```diff
- github.com/tursodatabase/libsql-client-go v0.0.0-20240902231107-85af5b9d094d
+ github.com/lib/pq v1.10.9
```

### Database Schema Migration
```sql
-- Old SQLite Schema
CREATE TABLE IF NOT EXISTS analytics (
    matric_no TEXT NOT NULL PRIMARY KEY,
    batch AS (substr(matric_no, 1, 2) + 2000) STORED,
    level AS (
        CASE length(matric_no)
            WHEN 7 THEN 'DEGREE'
            WHEN 6 THEN 'CFS'
        END
    ) STORED,
    timestamp DATETIME DEFAULT current_timestamp
);

-- New PostgreSQL Schema
CREATE TABLE IF NOT EXISTS analytics (
    matric_no VARCHAR(10) NOT NULL PRIMARY KEY,
    batch INTEGER GENERATED ALWAYS AS (CAST(SUBSTRING(matric_no, 1, 2) AS INTEGER) + 2000) STORED,
    level VARCHAR(10) GENERATED ALWAYS AS (
        CASE LENGTH(matric_no)
            WHEN 7 THEN 'DEGREE'
            WHEN 6 THEN 'CFS'
        END
    ) STORED,
    timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
```

### Environment Variables
```diff
- DB_PATH=file:analytics.db
+ DATABASE_URL=postgresql://user:pass@host:5432/gomaluum
```

### Code Changes
- **Connection**: `sql.Open("libsql", ...)` → `sql.Open("postgres", ...)`
- **Parameters**: `?` placeholders → `$1, $2, ...`
- **Functions**: `substr()` → `SUBSTRING()`, `length()` → `LENGTH()`
- **Types**: `TEXT` → `VARCHAR()`, `DATETIME` → `TIMESTAMPTZ`

## 🚀 For Users

### Without Database (Default Behavior)
```bash
# Just run the app - no database needed
go run main.go
# or
docker run -p 1323:1323 gomaluum
```

### With PostgreSQL Analytics
```bash
# Set the database URL and run
export DATABASE_URL=postgresql://user:pass@localhost:5432/gomaluum
go run main.go
# or
docker run -p 1323:1323 -e DATABASE_URL=postgresql://user:pass@localhost:5432/gomaluum gomaluum
```

## 📊 Migrating Existing Data

If you have existing analytics data in LibSQL/SQLite, use our automated migration tools:

### Quick Migration
```bash
# Using the shell script (recommended)
./scripts/migrate.sh -s "file:old_analytics.db" -t "postgresql://user:pass@localhost:5432/gomaluum"

# Check what would be migrated first
./scripts/migrate.sh -s "file:old_analytics.db" -t "postgresql://user:pass@localhost:5432/gomaluum" --dry-run
```

### Supported Source Formats
- **Local SQLite**: `file:path/to/database.db`
- **Turso (LibSQL)**: `libsql://database.turso.io?authToken=your_token`
- **Memory**: `:memory:` (for testing)

For detailed migration instructions, see:
- [`scripts/README.md`](scripts/README.md) - Migration script documentation
- [`docs/DATABASE.md`](docs/DATABASE.md) - Database configuration guide

## 🔄 Application Behavior

### Before Migration (LibSQL/SQLite)
- ❌ Required `DB_PATH` environment variable
- ❌ App would panic if database unavailable
- ❌ Used SQLite-specific SQL syntax
- ✅ Analytics functionality worked

### After Migration (PostgreSQL Optional)
- ✅ Optional `DATABASE_URL` environment variable
- ✅ App runs gracefully without database
- ✅ Uses standard PostgreSQL syntax
- ✅ Analytics work when database available
- ✅ All core functionality works regardless

## 🛡️ Backward Compatibility

### Environment Variables
- **Old**: `DB_PATH` (no longer used)
- **New**: `DATABASE_URL` (optional)

### Database Requirements
- **Before**: SQLite/LibSQL database required
- **After**: PostgreSQL database optional

### API Endpoints
- **Core APIs**: Work with or without database
- **Analytics API** (`/api/analytics`): Returns error if no database

## 🔍 Verification Steps

After migration, verify everything works:

1. **Test without database**:
   ```bash
   unset DATABASE_URL
   go run main.go
   # Should start successfully
   ```

2. **Test with database**:
   ```bash
   export DATABASE_URL=postgresql://user:pass@localhost:5432/gomaluum
   go run main.go
   # Should start and create schema
   ```

3. **Test analytics endpoint**:
   ```bash
   curl http://localhost:1323/api/analytics
   # Should return data or appropriate error
   ```

## 📋 Checklist for Upgrading

- [ ] Set up PostgreSQL database (if you want analytics)
- [ ] Update environment variables (`DB_PATH` → `DATABASE_URL`)
- [ ] Run migration script (if you have existing data)
- [ ] Test application startup
- [ ] Verify analytics functionality (if using database)
- [ ] Update deployment scripts/docker-compose
- [ ] Update documentation/README files

## 🆘 Troubleshooting

### Common Issues

1. **App won't start**: Check if you're still using `DB_PATH` instead of `DATABASE_URL`
2. **Migration fails**: Verify both source and target database connections
3. **Analytics not working**: Ensure `DATABASE_URL` is set and PostgreSQL is accessible
4. **Schema errors**: Make sure PostgreSQL user has CREATE TABLE permissions

### Getting Help

- Check [Database Documentation](docs/DATABASE.md)
- Review [Migration Scripts Documentation](scripts/README.md)
- Verify connection strings and credentials
- Test with dry-run mode first

## 🎉 Benefits of Migration

1. **More Robust**: PostgreSQL is more suitable for production analytics
2. **Scalable**: Better performance and concurrent access
3. **Optional**: App works without database dependency
4. **Standard**: Uses widely-adopted PostgreSQL instead of niche LibSQL
5. **Flexible**: Easy to integrate with existing PostgreSQL infrastructure
6. **Safe**: Automated migration with conflict handling and dry-run mode

## 📚 Additional Resources

- [Database Configuration Guide](docs/DATABASE.md)
- [Migration Scripts Documentation](scripts/README.md)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Connection String Reference](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)