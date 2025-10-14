package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type AnalyticsRecord struct {
	MatricNo  string
	Batch     *int
	Level     *string
	Timestamp time.Time
}

func main() {
	var (
		sourceDB = flag.String("source", "", "Source LibSQL/SQLite database path or connection string")
		targetDB = flag.String("target", "", "Target PostgreSQL connection string")
		dryRun   = flag.Bool("dry-run", false, "Show what would be migrated without actually doing it")
		help     = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	if *sourceDB == "" || *targetDB == "" {
		fmt.Println("Error: Both -source and -target parameters are required")
		printHelp()
		os.Exit(1)
	}

	fmt.Println("🔄 GoMa'luum Analytics Migration Tool")
	fmt.Println("=====================================")
	fmt.Printf("Source: %s\n", *sourceDB)
	fmt.Printf("Target: %s\n", *targetDB)
	fmt.Printf("Dry Run: %t\n\n", *dryRun)

	// Connect to source database (LibSQL/SQLite)
	fmt.Println("📂 Connecting to source database...")
	sourceConn, err := sql.Open("libsql", *sourceDB)
	if err != nil {
		log.Fatalf("Failed to connect to source database: %v", err)
	}
	defer sourceConn.Close()

	// Test source connection
	if err := sourceConn.Ping(); err != nil {
		log.Fatalf("Failed to ping source database: %v", err)
	}
	fmt.Println("✅ Connected to source database")

	// Connect to target database (PostgreSQL)
	fmt.Println("🐘 Connecting to target database...")
	targetConn, err := sql.Open("postgres", *targetDB)
	if err != nil {
		log.Fatalf("Failed to connect to target database: %v", err)
	}
	defer targetConn.Close()

	// Test target connection
	if err := targetConn.Ping(); err != nil {
		log.Fatalf("Failed to ping target database: %v", err)
	}
	fmt.Println("✅ Connected to target database")

	// Ensure target schema exists
	if !*dryRun {
		fmt.Println("🏗️  Creating target schema...")
		if err := createTargetSchema(targetConn); err != nil {
			log.Fatalf("Failed to create target schema: %v", err)
		}
		fmt.Println("✅ Target schema ready")
	}

	// Read data from source
	fmt.Println("📖 Reading data from source database...")
	records, err := readSourceData(sourceConn)
	if err != nil {
		log.Fatalf("Failed to read source data: %v", err)
	}
	fmt.Printf("📊 Found %d records to migrate\n", len(records))

	if len(records) == 0 {
		fmt.Println("ℹ️  No data to migrate")
		return
	}

	// Show sample data
	fmt.Println("\n📋 Sample records:")
	for i, record := range records {
		if i >= 5 { // Show only first 5 records
			fmt.Printf("... and %d more records\n", len(records)-5)
			break
		}
		fmt.Printf("  - %s (Batch: %v, Level: %v, Time: %s)\n",
			record.MatricNo,
			ptrToString(record.Batch),
			ptrToString(record.Level),
			record.Timestamp.Format("2006-01-02 15:04:05"))
	}

	if *dryRun {
		fmt.Println("\n🔍 DRY RUN: Would migrate these records (no actual changes made)")
		return
	}

	// Confirm migration
	fmt.Print("\n❓ Proceed with migration? (y/N): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		fmt.Println("❌ Migration cancelled")
		return
	}

	// Migrate data
	fmt.Println("\n🚀 Starting migration...")
	migrated, skipped, err := migrateData(targetConn, records)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Println("\n🎉 Migration completed!")
	fmt.Printf("✅ Migrated: %d records\n", migrated)
	fmt.Printf("⏭️  Skipped: %d records (already exist)\n", skipped)
	fmt.Printf("📊 Total processed: %d records\n", migrated+skipped)
}

func printHelp() {
	fmt.Println("GoMa'luum Analytics Migration Tool")
	fmt.Println("==================================")
	fmt.Println()
	fmt.Println("This tool migrates analytics data from LibSQL/SQLite to PostgreSQL.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run migrate_analytics.go -source <source_db> -target <target_db> [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -source string    Source LibSQL/SQLite database path or connection string")
	fmt.Println("                    Examples:")
	fmt.Println("                      file:local.db")
	fmt.Println("                      libsql://your-database.turso.io?authToken=your_token")
	fmt.Println("  -target string    Target PostgreSQL connection string")
	fmt.Println("                    Example: postgresql://user:pass@localhost:5432/gomaluum")
	fmt.Println("  -dry-run          Show what would be migrated without actually doing it")
	fmt.Println("  -help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Local SQLite to PostgreSQL")
	fmt.Println("  go run migrate_analytics.go -source \"file:analytics.db\" -target \"postgresql://user:pass@localhost:5432/gomaluum\"")
	fmt.Println()
	fmt.Println("  # Turso to PostgreSQL (dry run)")
	fmt.Println("  go run migrate_analytics.go -source \"libsql://db.turso.io?authToken=token\" -target \"postgresql://user:pass@host:5432/db\" -dry-run")
}

func createTargetSchema(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS analytics (
			matric_no VARCHAR(10) NOT NULL PRIMARY KEY,
			batch INTEGER GENERATED ALWAYS AS (CAST(SUBSTRING(matric_no, 1, 2) AS INTEGER) + 2000) STORED,
			level VARCHAR(10) GENERATED ALWAYS AS (
				CASE LENGTH(matric_no)
					WHEN 7 THEN 'DEGREE'
					WHEN 6 THEN 'CFS'
				END
			) STORED,
			timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_batch ON analytics(batch)`,
		`CREATE INDEX IF NOT EXISTS idx_level ON analytics(level)`,
		`CREATE INDEX IF NOT EXISTS idx_batch_level ON analytics(batch, level)`,
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement: %v", err)
		}
	}

	return nil
}

func readSourceData(db *sql.DB) ([]AnalyticsRecord, error) {
	query := `SELECT matric_no, batch, level, timestamp FROM analytics ORDER BY timestamp`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query source data: %v", err)
	}
	defer rows.Close()

	var records []AnalyticsRecord
	for rows.Next() {
		var record AnalyticsRecord
		var timestampStr string

		err := rows.Scan(&record.MatricNo, &record.Batch, &record.Level, &timestampStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		// Parse timestamp (SQLite stores it as string)
		record.Timestamp, err = parseTimestamp(timestampStr)
		if err != nil {
			log.Printf("Warning: failed to parse timestamp '%s' for %s, using current time: %v",
				timestampStr, record.MatricNo, err)
			record.Timestamp = time.Now()
		}

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	return records, nil
}

func parseTimestamp(timestampStr string) (time.Time, error) {
	// Try common timestamp formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000-07:00",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

func migrateData(db *sql.DB, records []AnalyticsRecord) (migrated, skipped int, err error) {
	stmt, err := db.Prepare(`
		INSERT INTO analytics (matric_no, timestamp)
		VALUES ($1, $2)
		ON CONFLICT (matric_no) DO NOTHING
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer stmt.Close()

	for i, record := range records {
		result, err := stmt.Exec(record.MatricNo, record.Timestamp)
		if err != nil {
			return migrated, skipped, fmt.Errorf("failed to insert record %s: %v", record.MatricNo, err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return migrated, skipped, fmt.Errorf("failed to get rows affected: %v", err)
		}

		if rowsAffected > 0 {
			migrated++
		} else {
			skipped++
		}

		// Progress indicator
		if (i+1)%100 == 0 || i == len(records)-1 {
			fmt.Printf("\r⏳ Progress: %d/%d records processed", i+1, len(records))
		}
	}
	fmt.Println() // New line after progress

	return migrated, skipped, nil
}

func ptrToString(ptr interface{}) string {
	switch v := ptr.(type) {
	case *int:
		if v == nil {
			return "nil"
		}
		return fmt.Sprintf("%d", *v)
	case *string:
		if v == nil {
			return "nil"
		}
		return *v
	default:
		return fmt.Sprintf("%v", v)
	}
}
