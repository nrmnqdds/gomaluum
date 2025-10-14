# Database Configuration (Optional)

GoMa'luum has migrated from SQLite/LibSQL to PostgreSQL for analytics storage. **The database is optional** - the application will run without it, but analytics functionality will be disabled.

## Environment Variables

### DATABASE_URL (Optional)
The PostgreSQL connection string used to connect to your database. If not provided, the application will run without analytics functionality.

**Format:**
```
postgresql://[user[:password]@][netloc][:port][/dbname][?param1=value1&...]
```

**Examples:**
```bash
# Local development
DATABASE_URL=postgresql://username:password@localhost:5432/gomaluum

# Production with SSL
DATABASE_URL=postgresql://user:pass@prod-db.example.com:5432/gomaluum?sslmode=require

# With connection pool settings
DATABASE_URL=postgresql://user:pass@localhost:5432/gomaluum?pool_max_conns=10&pool_min_conns=2
```

## Database Schema

The application automatically creates the following analytics table on startup:

```sql
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

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_batch ON analytics(batch);
CREATE INDEX IF NOT EXISTS idx_level ON analytics(level);
CREATE INDEX IF NOT EXISTS idx_batch_level ON analytics(batch, level);
```

## Application Behavior

### Without Database
- Application starts normally
- All core functionality (login, profile, schedule, results) works
- Analytics endpoints return errors
- No analytics data is collected

### With Database
- Application starts with database connection
- All functionality works including analytics
- Analytics data is collected automatically
- Analytics endpoints return data

## Migration from SQLite/LibSQL

If you're migrating from the previous SQLite/LibSQL implementation and want to keep analytics functionality:

1. **Set up PostgreSQL**: Install and configure a PostgreSQL database
2. **Update environment**: Replace `DB_PATH` with `DATABASE_URL`
3. **Migrate data** (if needed): Export existing analytics data and import into PostgreSQL

### Automated Data Migration

We provide automated migration scripts to transfer your existing analytics data:

```bash
# Using the shell script (recommended)
./scripts/migrate.sh -s "file:old_analytics.db" -t "postgresql://user:pass@localhost:5432/gomaluum"

# Or directly with Go
go run scripts/migrate_analytics.go -source "file:old_analytics.db" -target "postgresql://user:pass@localhost:5432/gomaluum"

# Dry run to see what would be migrated
./scripts/migrate.sh -s "file:old_analytics.db" -t "postgresql://user:pass@localhost:5432/gomaluum" --dry-run
```

The migration script handles:
- ✅ Automatic schema creation
- ✅ Data format conversion (SQLite → PostgreSQL)
- ✅ Duplicate prevention (`ON CONFLICT DO NOTHING`)
- ✅ Progress reporting
- ✅ Dry run mode for testing

For detailed migration instructions, see [`scripts/README.md`](../scripts/README.md).

### Manual Data Migration (if needed)

If you prefer manual migration:

```sql
-- Example manual migration query
INSERT INTO analytics (matric_no, timestamp)
SELECT matric_no, timestamp FROM your_sqlite_backup_table
ON CONFLICT (matric_no) DO NOTHING;
```

## Local Development Setup

### Using Docker Compose

Create a `docker-compose.yml` file for local development:

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: gomaluum
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  gomaluum:
    build: .
    environment:
      DATABASE_URL: postgresql://postgres:password@postgres:5432/gomaluum
    ports:
      - "1323:1323"
    depends_on:
      - postgres

volumes:
  postgres_data:
```

### Manual Setup

1. Install PostgreSQL
2. Create database:
   ```sql
   CREATE DATABASE gomaluum;
   ```
3. Set environment variable (optional):
   ```bash
   export DATABASE_URL=postgresql://username:password@localhost:5432/gomaluum
   ```

### Running Without Database
Simply don't set the `DATABASE_URL` environment variable and the application will run without analytics functionality.

## Production Considerations

### Connection Pooling
Configure connection pool parameters in your DATABASE_URL:
- `pool_max_conns`: Maximum number of connections
- `pool_min_conns`: Minimum number of connections
- `pool_max_conn_lifetime`: Maximum connection lifetime

### SSL/TLS
For production deployments, always use SSL:
```
DATABASE_URL=postgresql://user:pass@host:5432/db?sslmode=require
```

### Monitoring
Monitor your PostgreSQL instance for:
- Connection count
- Query performance
- Index usage
- Disk space

## Troubleshooting

### Common Issues

1. **Connection refused**: Check if PostgreSQL is running and accessible
2. **Authentication failed**: Verify credentials in DATABASE_URL
3. **Database not found**: Ensure the database exists
4. **SSL errors**: Check SSL configuration and certificates

### Debugging Connection Issues

Enable detailed logging by setting PostgreSQL log level:
```sql
SET log_statement = 'all';
SET log_min_duration_statement = 0;
```

## Analytics Functionality

The analytics system tracks:
- **Student matriculation numbers** (anonymized for privacy)
- **Academic level** (CFS/DEGREE based on matric number length)
- **Batch year** (calculated from matric number prefix)
- **Access timestamp** (when the student last used the API)

The system provides aggregate statistics via the `/api/analytics` endpoint, grouped by academic level and batch year.