#!/bin/bash

# GoMa'luum Analytics Migration Script
# ===================================
# This script helps migrate analytics data from LibSQL/SQLite to PostgreSQL

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

print_banner() {
    echo -e "${BLUE}🔄 GoMa'luum Analytics Migration Tool${NC}"
    echo -e "${BLUE}====================================${NC}"
    echo
}

print_help() {
    print_banner
    echo "This script helps migrate analytics data from LibSQL/SQLite to PostgreSQL."
    echo
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0 [options]"
    echo
    echo -e "${YELLOW}Options:${NC}"
    echo "  -s, --source <db>     Source LibSQL/SQLite database path or connection string"
    echo "  -t, --target <db>     Target PostgreSQL connection string"
    echo "  -d, --dry-run         Show what would be migrated without actually doing it"
    echo "  -h, --help            Show this help message"
    echo
    echo -e "${YELLOW}Environment Variables:${NC}"
    echo "  OLD_DB_PATH          Source database path (alternative to -s)"
    echo "  DATABASE_URL         Target database URL (alternative to -t)"
    echo
    echo -e "${YELLOW}Examples:${NC}"
    echo
    echo "  # Using command line arguments:"
    echo "  $0 -s \"file:analytics.db\" -t \"postgresql://user:pass@localhost:5432/gomaluum\""
    echo
    echo "  # Using environment variables:"
    echo "  export OLD_DB_PATH=\"file:analytics.db\""
    echo "  export DATABASE_URL=\"postgresql://user:pass@localhost:5432/gomaluum\""
    echo "  $0"
    echo
    echo "  # Dry run to see what would be migrated:"
    echo "  $0 -s \"libsql://db.turso.io?authToken=token\" -t \"postgresql://user:pass@host:5432/db\" --dry-run"
    echo
    echo -e "${YELLOW}Common Source Database Formats:${NC}"
    echo "  Local SQLite:    file:path/to/database.db"
    echo "  Turso (LibSQL):  libsql://your-database.turso.io?authToken=your_token"
    echo "  Memory SQLite:   :memory:"
    echo
    echo -e "${YELLOW}PostgreSQL Connection String Format:${NC}"
    echo "  postgresql://[user[:password]@][host][:port][/dbname][?param1=value1&...]"
    echo
}

check_dependencies() {
    echo -e "${BLUE}🔍 Checking dependencies...${NC}"

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go is not installed. Please install Go 1.23 or higher.${NC}"
        exit 1
    fi

    # Check Go version
    go_version=$(go version | cut -d' ' -f3 | sed 's/go//')
    echo -e "${GREEN}✅ Go ${go_version} found${NC}"

    # Check if migration go.mod exists
    if [ ! -f "${SCRIPT_DIR}/go.mod" ]; then
        echo -e "${RED}❌ Migration go.mod not found in scripts directory.${NC}"
        exit 1
    fi

    echo -e "${GREEN}✅ Dependencies check passed${NC}"
    echo
}

validate_connection_string() {
    local db_type="$1"
    local connection_string="$2"

    if [ "$db_type" = "source" ]; then
        # Validate source (LibSQL/SQLite)
        if [[ ! "$connection_string" =~ ^(file:|libsql://|:memory:) ]]; then
            echo -e "${YELLOW}⚠️  Warning: Source database format may not be valid. Expected formats:${NC}"
            echo -e "   - file:path/to/database.db"
            echo -e "   - libsql://database.turso.io?authToken=token"
            echo -e "   - :memory:"
        fi
    elif [ "$db_type" = "target" ]; then
        # Validate target (PostgreSQL)
        if [[ ! "$connection_string" =~ ^postgresql:// ]]; then
            echo -e "${YELLOW}⚠️  Warning: Target database should be a PostgreSQL connection string${NC}"
            echo -e "   Expected format: postgresql://user:pass@host:port/dbname"
        fi
    fi
}

run_migration() {
    local source_db="$1"
    local target_db="$2"
    local dry_run="$3"

    echo -e "${BLUE}🚀 Running migration...${NC}"
    echo

    # Build the Go command
    local go_cmd="go run migrate_analytics.go -source \"$source_db\" -target \"$target_db\""

    if [ "$dry_run" = "true" ]; then
        go_cmd="$go_cmd -dry-run"
    fi

    # Change to the scripts directory to use the migration go.mod
    cd "${SCRIPT_DIR}"

    # Run the migration
    echo -e "${BLUE}Executing: $go_cmd${NC}"
    echo
    eval $go_cmd
}

# Parse command line arguments
SOURCE_DB=""
TARGET_DB=""
DRY_RUN="false"

while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--source)
            SOURCE_DB="$2"
            shift 2
            ;;
        -t|--target)
            TARGET_DB="$2"
            shift 2
            ;;
        -d|--dry-run)
            DRY_RUN="true"
            shift
            ;;
        -h|--help)
            print_help
            exit 0
            ;;
        *)
            echo -e "${RED}❌ Unknown option: $1${NC}"
            echo
            print_help
            exit 1
            ;;
    esac
done

# Use environment variables as fallback
if [ -z "$SOURCE_DB" ] && [ -n "$OLD_DB_PATH" ]; then
    SOURCE_DB="$OLD_DB_PATH"
fi

if [ -z "$TARGET_DB" ] && [ -n "$DATABASE_URL" ]; then
    TARGET_DB="$DATABASE_URL"
fi

# Validate required parameters
if [ -z "$SOURCE_DB" ] || [ -z "$TARGET_DB" ]; then
    print_banner
    echo -e "${RED}❌ Error: Both source and target databases are required${NC}"
    echo
    if [ -z "$SOURCE_DB" ]; then
        echo -e "Missing source database. Use -s flag or set OLD_DB_PATH environment variable."
    fi
    if [ -z "$TARGET_DB" ]; then
        echo -e "Missing target database. Use -t flag or set DATABASE_URL environment variable."
    fi
    echo
    echo "Use -h or --help for more information."
    exit 1
fi

# Main execution
print_banner

# Check dependencies
check_dependencies

# Validate connection strings
validate_connection_string "source" "$SOURCE_DB"
validate_connection_string "target" "$TARGET_DB"

# Show configuration
echo -e "${BLUE}📋 Migration Configuration:${NC}"
echo -e "  Source Database: ${YELLOW}$SOURCE_DB${NC}"
echo -e "  Target Database: ${YELLOW}$TARGET_DB${NC}"
echo -e "  Dry Run:         ${YELLOW}$DRY_RUN${NC}"
echo

# Confirm before proceeding (unless it's a dry run)
if [ "$DRY_RUN" = "false" ]; then
    echo -e "${YELLOW}⚠️  This will modify your PostgreSQL database.${NC}"
    read -p "Continue? (y/N): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}❌ Migration cancelled by user${NC}"
        exit 0
    fi
    echo
fi

# Run the migration
run_migration "$SOURCE_DB" "$TARGET_DB" "$DRY_RUN"
