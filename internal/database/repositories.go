package database

import (
	"database/sql"

	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/frostdev-ops/pma-backend-go/internal/database/sqlite"
)

// Repositories holds all repository instances
type Repositories struct {
	User   repositories.UserRepository
	Config repositories.ConfigRepository
	Entity repositories.EntityRepository
	Room   repositories.RoomRepository
}

// NewRepositories creates all repository instances
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		User:   sqlite.NewUserRepository(db),
		Config: sqlite.NewConfigRepository(db),
		Entity: sqlite.NewEntityRepository(db),
		Room:   sqlite.NewRoomRepository(db),
	}
}
