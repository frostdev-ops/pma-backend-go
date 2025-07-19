package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type EntityMetadata struct {
	Migrated       bool   `json:"migrated"`
	OriginalDomain string `json:"original_domain"`
	MigrationDate  string `json:"migration_date"`
}

func main() {
	// Get database path from command line args or use default
	dbPath := "./pma.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	log.Printf("Migrating entities to PMA format in database: %s", dbPath)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Verify the entity_metadata table exists
	var tableExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='entity_metadata'").Scan(&tableExists)
	if err != nil {
		log.Fatal("Failed to check for entity_metadata table:", err)
	}
	if tableExists == 0 {
		log.Fatal("entity_metadata table does not exist. Please run migration 017 first.")
	}

	// Get all existing entities
	rows, err := db.Query("SELECT entity_id, domain, attributes FROM entities WHERE entity_id NOT IN (SELECT entity_id FROM entity_metadata)")
	if err != nil {
		log.Fatal("Failed to query entities:", err)
	}
	defer rows.Close()

	// Prepare insert statement for metadata
	stmt, err := db.Prepare(`
		INSERT OR IGNORE INTO entity_metadata (entity_id, source, source_entity_id, metadata, quality_score, last_synced)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		log.Fatal("Failed to prepare statement:", err)
	}
	defer stmt.Close()

	// Migrate each entity
	count := 0
	for rows.Next() {
		var entityID, domain string
		var attributes sql.NullString

		if err := rows.Scan(&entityID, &domain, &attributes); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Determine source based on entity ID pattern
		source := determineSource(entityID)

		// Create metadata
		metadata := EntityMetadata{
			Migrated:       true,
			OriginalDomain: domain,
			MigrationDate:  "2024-01-01", // Current migration
		}

		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			log.Printf("Error marshaling metadata for %s: %v", entityID, err)
			continue
		}

		// Insert metadata
		_, err = stmt.Exec(entityID, source, entityID, string(metadataJSON), 1.0)
		if err != nil {
			log.Printf("Error inserting metadata for %s: %v", entityID, err)
			continue
		}

		count++
		if count%100 == 0 {
			log.Printf("Migrated %d entities...", count)
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error during row iteration: %v", err)
	}

	// Update entities table with default values for new columns
	log.Println("Updating entities table with default PMA values...")
	
	// Set available = true for all entities by default
	result, err := db.Exec("UPDATE entities SET available = true WHERE available IS NULL")
	if err != nil {
		log.Printf("Warning: Failed to update available column: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		log.Printf("Updated available column for %d entities", rowsAffected)
	}

	// Set empty capabilities array for entities without capabilities
	result, err = db.Exec("UPDATE entities SET pma_capabilities = '[]' WHERE pma_capabilities IS NULL")
	if err != nil {
		log.Printf("Warning: Failed to update pma_capabilities column: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		log.Printf("Updated pma_capabilities column for %d entities", rowsAffected)
	}

	log.Printf("Successfully migrated metadata for %d entities to PMA format", count)
}

// determineSource determines the source adapter based on entity ID patterns
func determineSource(entityID string) string {
	entityID = strings.ToLower(entityID)
	
	// Check for specific prefixes or patterns
	switch {
	case strings.HasPrefix(entityID, "ring"):
		return "ring"
	case strings.HasPrefix(entityID, "shel"):
		return "shelly"
	case strings.HasPrefix(entityID, "ups_"), strings.Contains(entityID, "ups"):
		return "ups"
	case strings.HasPrefix(entityID, "net_"), strings.Contains(entityID, "network"):
		return "network"
	case strings.Contains(entityID, "bluetooth"):
		return "bluetooth"
	case strings.Contains(entityID, "camera"):
		return "camera"
	default:
		// Default to homeassistant for standard patterns
		return "homeassistant"
	}
} 