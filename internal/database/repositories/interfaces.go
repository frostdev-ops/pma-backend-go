package repositories

import (
	"context"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
)

// UserRepository defines user data access methods
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
}

// ConfigRepository defines system config data access methods
type ConfigRepository interface {
	Get(ctx context.Context, key string) (*models.SystemConfig, error)
	Set(ctx context.Context, config *models.SystemConfig) error
	GetAll(ctx context.Context) ([]*models.SystemConfig, error)
	Delete(ctx context.Context, key string) error
}

// EntityRepository defines entity data access methods
type EntityRepository interface {
	Create(ctx context.Context, entity *models.Entity) error
	GetByID(ctx context.Context, entityID string) (*models.Entity, error)
	GetAll(ctx context.Context) ([]*models.Entity, error)
	GetByRoom(ctx context.Context, roomID int) ([]*models.Entity, error)
	Update(ctx context.Context, entity *models.Entity) error
	Delete(ctx context.Context, entityID string) error
}

// RoomRepository defines room data access methods
type RoomRepository interface {
	Create(ctx context.Context, room *models.Room) error
	GetByID(ctx context.Context, id int) (*models.Room, error)
	GetByName(ctx context.Context, name string) (*models.Room, error)
	GetAll(ctx context.Context) ([]*models.Room, error)
	Update(ctx context.Context, room *models.Room) error
	Delete(ctx context.Context, id int) error
}
