package sqlite

import (
	"database/sql"
	"testing"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	_ "github.com/mattn/go-sqlite3"
)

func setupPMATestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create entities table
	_, err = db.Exec(`
		CREATE TABLE entities (
			entity_id TEXT PRIMARY KEY,
			friendly_name TEXT,
			domain TEXT NOT NULL,
			state TEXT,
			attributes TEXT,
			last_updated TIMESTAMP NOT NULL,
			room_id INTEGER,
			pma_capabilities TEXT,
			available BOOLEAN DEFAULT TRUE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create entities table: %v", err)
	}

	// Create entity_metadata table
	_, err = db.Exec(`
		CREATE TABLE entity_metadata (
			entity_id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			source_entity_id TEXT NOT NULL,
			metadata TEXT,
			quality_score REAL DEFAULT 1.0,
			last_synced TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			is_virtual BOOLEAN DEFAULT FALSE,
			virtual_sources TEXT,
			FOREIGN KEY (entity_id) REFERENCES entities(entity_id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create entity_metadata table: %v", err)
	}

	return db
}

func TestCreateOrUpdatePMAEntity(t *testing.T) {
	db := setupPMATestDB(t)
	defer db.Close()

	repo := &EntityRepository{db: db}

	// Create test metadata
	metadata := &types.PMAMetadata{
		Source:         types.SourceHomeAssistant,
		SourceEntityID: "light.living_room",
		LastSynced:     time.Now(),
		QualityScore:   0.9,
	}

	// Create test entity
	entity := &types.PMABaseEntity{
		ID:           "light.living_room",
		Type:         types.EntityTypeLight,
		FriendlyName: "Living Room Light",
		State:        types.StateOn,
		Attributes:   map[string]interface{}{"brightness": 255},
		LastUpdated:  time.Now(),
		Capabilities: []types.PMACapability{types.CapabilityDimmable},
		Available:    true,
		Metadata:     metadata,
	}

	// Test create
	err := repo.CreateOrUpdatePMAEntity(entity)
	if err != nil {
		t.Fatalf("Failed to create PMA entity: %v", err)
	}

	// Test retrieve
	retrieved, err := repo.GetPMAEntity("light.living_room")
	if err != nil {
		t.Fatalf("Failed to get PMA entity: %v", err)
	}

	if retrieved.GetID() != "light.living_room" {
		t.Errorf("Expected entity ID 'light.living_room', got '%s'", retrieved.GetID())
	}

	if retrieved.GetFriendlyName() != "Living Room Light" {
		t.Errorf("Expected friendly name 'Living Room Light', got '%s'", retrieved.GetFriendlyName())
	}

	if retrieved.GetSource() != types.SourceHomeAssistant {
		t.Errorf("Expected source 'homeassistant', got '%s'", retrieved.GetSource())
	}

	// Test update
	entity.State = types.StateOff
	err = repo.CreateOrUpdatePMAEntity(entity)
	if err != nil {
		t.Fatalf("Failed to update PMA entity: %v", err)
	}

	// Verify update
	updated, err := repo.GetPMAEntity("light.living_room")
	if err != nil {
		t.Fatalf("Failed to get updated PMA entity: %v", err)
	}

	if updated.GetState() != types.StateOff {
		t.Errorf("Expected state 'off', got '%s'", updated.GetState())
	}
}

func TestGetPMAEntitiesBySource(t *testing.T) {
	db := setupPMATestDB(t)
	defer db.Close()

	repo := &EntityRepository{db: db}

	// Create test entities from different sources
	entities := []*types.PMABaseEntity{
		{
			ID:           "light.ha_light",
			Type:         types.EntityTypeLight,
			FriendlyName: "HA Light",
			State:        types.StateOn,
			Attributes:   map[string]interface{}{},
			LastUpdated:  time.Now(),
			Available:    true,
			Metadata: &types.PMAMetadata{
				Source:         types.SourceHomeAssistant,
				SourceEntityID: "light.ha_light",
				LastSynced:     time.Now(),
				QualityScore:   1.0,
			},
		},
		{
			ID:           "switch.shelly_switch",
			Type:         types.EntityTypeSwitch,
			FriendlyName: "Shelly Switch",
			State:        types.StateOff,
			Attributes:   map[string]interface{}{},
			LastUpdated:  time.Now(),
			Available:    true,
			Metadata: &types.PMAMetadata{
				Source:         types.SourceShelly,
				SourceEntityID: "switch.shelly_switch",
				LastSynced:     time.Now(),
				QualityScore:   1.0,
			},
		},
	}

	// Create entities
	for _, entity := range entities {
		err := repo.CreateOrUpdatePMAEntity(entity)
		if err != nil {
			t.Fatalf("Failed to create entity %s: %v", entity.ID, err)
		}
	}

	// Test retrieving by source
	haEntities, err := repo.GetPMAEntitiesBySource(types.SourceHomeAssistant)
	if err != nil {
		t.Fatalf("Failed to get HA entities: %v", err)
	}

	if len(haEntities) != 1 {
		t.Errorf("Expected 1 HA entity, got %d", len(haEntities))
	}

	if haEntities[0].GetID() != "light.ha_light" {
		t.Errorf("Expected HA entity ID 'light.ha_light', got '%s'", haEntities[0].GetID())
	}

	shellyEntities, err := repo.GetPMAEntitiesBySource(types.SourceShelly)
	if err != nil {
		t.Fatalf("Failed to get Shelly entities: %v", err)
	}

	if len(shellyEntities) != 1 {
		t.Errorf("Expected 1 Shelly entity, got %d", len(shellyEntities))
	}

	if shellyEntities[0].GetID() != "switch.shelly_switch" {
		t.Errorf("Expected Shelly entity ID 'switch.shelly_switch', got '%s'", shellyEntities[0].GetID())
	}
}

func TestDeletePMAEntity(t *testing.T) {
	db := setupPMATestDB(t)
	defer db.Close()

	repo := &EntityRepository{db: db}

	// Create test entity
	entity := &types.PMABaseEntity{
		ID:           "test.entity",
		Type:         types.EntityTypeSensor,
		FriendlyName: "Test Entity",
		State:        types.StateUnknown,
		Attributes:   map[string]interface{}{},
		LastUpdated:  time.Now(),
		Available:    true,
		Metadata: &types.PMAMetadata{
			Source:         types.SourcePMA,
			SourceEntityID: "test.entity",
			LastSynced:     time.Now(),
			QualityScore:   1.0,
		},
	}

	err := repo.CreateOrUpdatePMAEntity(entity)
	if err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Verify entity exists
	_, err = repo.GetPMAEntity("test.entity")
	if err != nil {
		t.Fatalf("Failed to get entity before deletion: %v", err)
	}

	// Delete entity
	err = repo.DeletePMAEntity("test.entity")
	if err != nil {
		t.Fatalf("Failed to delete entity: %v", err)
	}

	// Verify entity is deleted
	_, err = repo.GetPMAEntity("test.entity")
	if err == nil {
		t.Error("Expected error when getting deleted entity, but got nil")
	}
}
