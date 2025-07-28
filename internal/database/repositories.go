package database

import (
	"database/sql"

	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/frostdev-ops/pma-backend-go/internal/database/sqlite"
	"github.com/jmoiron/sqlx"
)

// Repositories holds all repository instances
type Repositories struct {
	User         repositories.UserRepository
	Config       repositories.ConfigRepository
	Entity       repositories.EntityRepository
	Room         repositories.RoomRepository
	Auth         repositories.AuthRepository
	Kiosk        repositories.KioskRepository
	Network      repositories.NetworkRepository
	UPS          repositories.UPSRepository
	Camera       repositories.CameraRepository
	Display      repositories.DisplayRepository
	Bluetooth    repositories.BluetoothRepository
	Energy       repositories.EnergyRepository
	Conversation repositories.ConversationRepository
	MCP          repositories.MCPRepository
	Area         repositories.AreaRepository
	Controller   repositories.ControllerRepository
	Screensaver  repositories.ScreensaverRepository
}

// NewRepositories creates all repository instances
func NewRepositories(db *sql.DB) *Repositories {
	// Create sqlx wrapper for repositories that need it
	sqlxDB := sqlx.NewDb(db, "sqlite")

	return &Repositories{
		User:         sqlite.NewUserRepository(db),
		Config:       sqlite.NewConfigRepository(db),
		Entity:       sqlite.NewEntityRepository(db),
		Room:         sqlite.NewRoomRepository(db),
		Auth:         sqlite.NewAuthRepository(db),
		Kiosk:        sqlite.NewKioskRepository(db),
		Network:      sqlite.NewNetworkRepository(db),
		UPS:          sqlite.NewUPSRepository(db),
		Camera:       sqlite.NewCameraRepository(db),
		Display:      sqlite.NewDisplaySettingsRepository(db),
		Bluetooth:    sqlite.NewBluetoothRepository(db),
		Energy:       sqlite.NewEnergyRepository(db),
		Conversation: sqlite.NewConversationRepository(db),
		MCP:          sqlite.NewMCPRepository(db),
		Area:         sqlite.NewAreaRepository(db),
		Controller:   sqlite.NewControllerRepository(db),
		Screensaver:  sqlite.NewScreensaverRepository(sqlxDB),
	}
}
