package unified

import (
	"testing"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestUnifiedEntityService_NewService(t *testing.T) {
	logger := logrus.New()
	typeRegistry := types.NewPMATypeRegistry(logger)

	service := NewUnifiedEntityService(typeRegistry, logger)

	assert.NotNil(t, service)
	assert.NotNil(t, service.registryManager)
	assert.NotNil(t, service.typeRegistry)
	assert.Equal(t, typeRegistry, service.typeRegistry)
	assert.NotNil(t, service.entityCache)
}

func TestUnifiedEntityService_GetRegistryManager(t *testing.T) {
	logger := logrus.New()
	typeRegistry := types.NewPMATypeRegistry(logger)
	service := NewUnifiedEntityService(typeRegistry, logger)

	registryManager := service.GetRegistryManager()
	assert.NotNil(t, registryManager)
	assert.Equal(t, service.registryManager, registryManager)
}

func TestUnifiedEntityService_RegistryComponents(t *testing.T) {
	logger := logrus.New()
	typeRegistry := types.NewPMATypeRegistry(logger)
	service := NewUnifiedEntityService(typeRegistry, logger)

	// Verify all registry components are available
	rm := service.GetRegistryManager()
	assert.NotNil(t, rm.GetAdapterRegistry())
	assert.NotNil(t, rm.GetEntityRegistry())
	assert.NotNil(t, rm.GetConflictResolver())
	assert.NotNil(t, rm.GetPriorityManager())
}
