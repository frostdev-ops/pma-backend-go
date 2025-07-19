# PMA Backend Integration Tests

This directory contains comprehensive integration tests for the PMA (Personal Management Automation) backend system.

## Test Structure

### Integration Tests (`integration/`)

The integration tests verify that all components work together correctly:

- **`pma_flow_test.go`** - Main integration test suite covering:
  - Complete entity flow (connect → sync → retrieve → control)
  - Multi-source conflict resolution
  - Registry operations and statistics
  - Action validation and error handling
  - Adapter health monitoring
  - Entity capabilities verification
  - Room and area assignment
  - Concurrent operations

- **`mock_adapter.go`** - Mock PMA adapter implementation for testing:
  - Implements full `PMAAdapter` interface
  - Provides controllable test entities (lights, sensors)
  - Simulates connection states and health metrics
  - Supports action execution and state changes
  - Includes conflict resolution test scenarios

- **`api_test.go`** - HTTP API integration tests:
  - Entity retrieval endpoints
  - Action execution via REST API
  - Authentication and authorization
  - Error handling and response formats
  - Adapter status endpoints

- **`performance_test.go`** - Performance and benchmarking tests:
  - Entity sync performance with large datasets
  - Conflict resolution performance
  - Concurrent operation handling
  - Memory usage and leak detection
  - Scaling tests for different entity counts

### End-to-End Tests (`e2e/`)

- **`test_pma_flow.sh`** - Complete end-to-end test script:
  - Server startup and shutdown
  - Health endpoint verification
  - Full API workflow testing
  - Authentication testing
  - WebSocket connection verification
  - Real HTTP request/response validation

## Running Tests

### Prerequisites

Ensure you have the following dependencies installed:

```bash
# Go dependencies (already in go.mod)
go mod tidy

# System dependencies for e2e tests
sudo apt-get install curl jq  # Ubuntu/Debian
# or
brew install curl jq         # macOS
```

### Integration Tests

Run all integration tests:

```bash
# From project root
go test ./tests/integration/... -v
```

Run specific test suites:

```bash
# Main PMA flow tests
go test ./tests/integration/ -run TestPMAIntegrationSuite -v

# API tests
go test ./tests/integration/ -run TestAPIIntegrationSuite -v

# Performance tests
go test ./tests/integration/ -run TestPerformance -v
```

Run benchmarks:

```bash
# All benchmarks
go test ./tests/integration/ -bench=. -benchmem

# Specific benchmarks
go test ./tests/integration/ -bench=BenchmarkEntitySync -benchmem
go test ./tests/integration/ -bench=BenchmarkConflictResolution -benchmem
```

### End-to-End Tests

Run the complete e2e test suite:

```bash
# From project root
./tests/e2e/test_pma_flow.sh
```

The script will:
1. Start the PMA backend server
2. Wait for it to be ready
3. Run a series of HTTP API tests
4. Clean up and stop the server
5. Report results

## Test Coverage

### What's Tested

1. **Core PMA Type System**
   - Entity creation and management
   - Type registry functionality
   - Adapter registration and lifecycle
   - Conflict resolution between sources

2. **Unified Entity Service**
   - Entity synchronization from adapters
   - Multi-source entity management
   - Action routing and execution
   - Room and area relationships

3. **API Layer**
   - HTTP endpoint functionality
   - JSON response formats
   - Authentication and authorization
   - Error handling and status codes

4. **Performance Characteristics**
   - Large dataset handling (up to 10,000 entities)
   - Concurrent operation safety
   - Memory usage patterns
   - Response time scaling

5. **Error Scenarios**
   - Invalid entity IDs
   - Unsupported actions
   - Adapter disconnection
   - Malformed requests

### Test Scenarios

#### Entity Flow Testing
```
Adapter → Connect → Sync Entities → Register in PMA Registry → 
Retrieve via API → Execute Actions → Verify State Changes
```

#### Conflict Resolution Testing
```
Multiple Adapters → Same Entity IDs → Priority-based Resolution → 
Verify Correct Source Selected → Action Routing to Right Adapter
```

#### Performance Testing
```
Large Entity Sets → Sync Performance → Retrieval Performance → 
Concurrent Access → Memory Usage → Scaling Characteristics
```

## Mock Components

### MockPMAAdapter

The mock adapter provides:

- **Test Entities**: Pre-configured lights, sensors, and other devices
- **Controllable Behavior**: Simulate connection states, health, metrics
- **Action Simulation**: Realistic state changes for turn_on/turn_off actions
- **Error Injection**: Test error conditions and edge cases
- **Performance Testing**: Support for large entity counts

### Test Data

Standard test entities created by the mock adapter:

- `ha_light.test_light` - Dimmable light entity
- `ha_sensor.test_sensor` - Temperature sensor
- `ha_light.conflict_test` - Used for conflict resolution testing

## Continuous Integration

These tests are designed to be run in CI/CD pipelines:

```yaml
# Example GitHub Actions step
- name: Run Integration Tests
  run: |
    go test ./tests/integration/... -v -race
    
- name: Run E2E Tests
  run: |
    ./tests/e2e/test_pma_flow.sh
```

## Debugging Failed Tests

### Integration Test Failures

1. **Check Logs**: Tests use logrus with debug level
2. **Examine Mock State**: Mock adapter maintains action logs
3. **Registry Inspection**: Use registry statistics for debugging
4. **Concurrent Issues**: Tests include race detection

### E2E Test Failures

1. **Server Logs**: Check `/tmp/pma-backend-test.log`
2. **Response Files**: Examine `/tmp/*_response.json` files
3. **Network Issues**: Verify port 3001 availability
4. **Dependencies**: Ensure curl and jq are installed

### Common Issues

- **Port Conflicts**: E2E tests use port 3001 by default
- **Database Locks**: SQLite database contention in concurrent tests
- **Timeout Issues**: Slow systems may need longer wait times
- **Missing Dependencies**: Ensure all Go modules are available

## Extending Tests

### Adding New Test Scenarios

1. **Integration Tests**: Add new test methods to existing suites
2. **Mock Entities**: Extend `MockPMAAdapter` with new entity types
3. **API Tests**: Add new endpoint tests to `api_test.go`
4. **Performance**: Add new benchmarks to `performance_test.go`

### Test Patterns

```go
// Standard integration test pattern
func (suite *PMAIntegrationTestSuite) TestNewFeature() {
    ctx := context.Background()
    
    // Setup
    err := suite.testAdapter.Connect(ctx)
    suite.Assert().NoError(err)
    
    // Test operation
    result, err := suite.unifiedService.SomeOperation(ctx, params)
    
    // Verify results
    suite.Assert().NoError(err)
    suite.Assert().Equal(expected, result)
}
```

## Metrics and Reporting

The tests collect various metrics:

- **Performance**: Benchmark results for throughput and latency
- **Coverage**: Go test coverage reports
- **Memory**: Memory usage patterns and leak detection
- **Concurrency**: Race condition detection
- **Health**: Adapter health and metrics validation

Run with coverage:

```bash
go test ./tests/integration/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
``` 