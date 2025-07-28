package ai

import (
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/interfaces"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
)

// Conversion functions to transform service results to MCP-compatible format

// convertEntityToMCPFormat converts EntityWithRoom to MCP format
func convertEntityToMCPFormat(entityWithRoom *interfaces.EntityWithRoom) map[string]interface{} {
	if entityWithRoom == nil || entityWithRoom.Entity == nil {
		return nil
	}

	entity := entityWithRoom.Entity
	result := map[string]interface{}{
		"entity_id":     entity.GetID(),
		"state":         string(entity.GetState()),
		"friendly_name": entity.GetFriendlyName(),
		"entity_type":   string(entity.GetType()),
		"source":        string(entity.GetSource()),
		"attributes":    entity.GetAttributes(),
		"last_changed":  entity.GetLastUpdated().Format(time.RFC3339),
		"last_updated":  entity.GetLastUpdated().Format(time.RFC3339),
	}

	// Add capabilities if available
	if caps := entity.GetCapabilities(); len(caps) > 0 {
		capStrings := make([]string, len(caps))
		for i, cap := range caps {
			capStrings[i] = string(cap)
		}
		result["capabilities"] = capStrings
	}

	// Add room information if available
	if entityWithRoom.Room != nil {
		result["room"] = convertRoomToMCPFormat(entityWithRoom.Room)
		result["room_id"] = entityWithRoom.Room.ID
	}

	// Add area information if available
	if entityWithRoom.Area != nil {
		result["area"] = convertAreaToMCPFormat(entityWithRoom.Area)
		result["area_id"] = entityWithRoom.Area.ID
	}

	// Add metadata if available
	if metadata := entity.GetMetadata(); metadata != nil {
		result["metadata"] = map[string]interface{}{
			"source":           string(metadata.Source),
			"source_entity_id": metadata.SourceEntityID,
			"source_device_id": metadata.SourceDeviceID,
			"last_synced":      metadata.LastSynced.Format(time.RFC3339),
			"quality_score":    metadata.QualityScore,
			"is_virtual":       metadata.IsVirtual,
		}

		if len(metadata.SyncErrors) > 0 {
			result["metadata"].(map[string]interface{})["sync_errors"] = metadata.SyncErrors
		}
	}

	// Add context information
	result["context"] = map[string]interface{}{
		"id":        entity.GetID(),
		"parent_id": nil,
		"user_id":   nil,
	}

	return result
}

// convertEntitesToMCPFormat converts array of EntityWithRoom to MCP format
func convertEntitesToMCPFormat(entities []*interfaces.EntityWithRoom) []interface{} {
	if entities == nil {
		return []interface{}{}
	}

	result := make([]interface{}, len(entities))
	for i, entity := range entities {
		result[i] = convertEntityToMCPFormat(entity)
	}
	return result
}

// convertRoomToMCPFormat converts PMARoom to MCP format
func convertRoomToMCPFormat(room *types.PMARoom) map[string]interface{} {
	if room == nil {
		return nil
	}

	result := map[string]interface{}{
		"room_id":     room.ID,
		"room_name":   room.Name,
		"description": room.Description,
		"icon":        room.Icon,
		"created_at":  room.CreatedAt.Format(time.RFC3339),
		"updated_at":  room.UpdatedAt.Format(time.RFC3339),
	}

	if room.EntityIDs != nil {
		result["entities"] = room.EntityIDs
	} else {
		result["entities"] = []string{}
	}

	if room.ParentID != nil {
		result["parent_id"] = *room.ParentID
	}

	if room.Children != nil && len(room.Children) > 0 {
		result["children"] = room.Children
	}

	return result
}

// convertAreaToMCPFormat converts PMAArea to MCP format
func convertAreaToMCPFormat(area *types.PMAArea) map[string]interface{} {
	if area == nil {
		return nil
	}

	result := map[string]interface{}{
		"area_id":     area.ID,
		"area_name":   area.Name,
		"description": area.Description,
		"created_at":  area.CreatedAt.Format(time.RFC3339),
		"updated_at":  area.UpdatedAt.Format(time.RFC3339),
	}

	if area.Icon != "" {
		result["icon"] = area.Icon
	}

	if area.RoomIDs != nil && len(area.RoomIDs) > 0 {
		result["room_ids"] = area.RoomIDs
	}

	if area.EntityIDs != nil && len(area.EntityIDs) > 0 {
		result["entity_ids"] = area.EntityIDs
	}

	return result
}

// convertSystemStatusToMCPFormat converts SystemStatus to MCP format
func convertSystemStatusToMCPFormat(status *interfaces.SystemStatus) map[string]interface{} {
	if status == nil {
		return nil
	}

	result := map[string]interface{}{
		"status":    status.Status,
		"timestamp": status.Timestamp.Format(time.RFC3339),
		"device_id": status.DeviceID,
		"uptime":    status.Uptime.Seconds(),
	}

	// Add CPU information
	if status.CPU != nil {
		result["cpu"] = map[string]interface{}{
			"usage":        status.CPU.Usage,
			"load_average": status.CPU.LoadAverage,
			"cores":        status.CPU.Cores,
			"model":        status.CPU.Model,
		}
		if status.CPU.Frequency > 0 {
			result["cpu"].(map[string]interface{})["frequency"] = status.CPU.Frequency
		}
	}

	// Add Memory information
	if status.Memory != nil {
		result["memory"] = map[string]interface{}{
			"total":        status.Memory.Total,
			"available":    status.Memory.Available,
			"used":         status.Memory.Used,
			"used_percent": status.Memory.UsedPercent,
		}
		if status.Memory.Buffers > 0 {
			result["memory"].(map[string]interface{})["buffers"] = status.Memory.Buffers
		}
		if status.Memory.Cached > 0 {
			result["memory"].(map[string]interface{})["cached"] = status.Memory.Cached
		}
	}

	// Add Disk information
	if status.Disk != nil {
		result["disk"] = map[string]interface{}{
			"total":        status.Disk.Total,
			"free":         status.Disk.Free,
			"used":         status.Disk.Used,
			"used_percent": status.Disk.UsedPercent,
			"filesystem":   status.Disk.Filesystem,
			"mount_point":  status.Disk.MountPoint,
		}
	}

	// Add Services information
	if status.Services != nil {
		result["services"] = status.Services
	}

	// Add System Load information
	if status.SystemLoad != nil && len(status.SystemLoad) > 0 {
		result["system_load"] = status.SystemLoad
	}

	// Add Network information
	if status.NetworkInfo != nil {
		networkInfo := map[string]interface{}{
			"interfaces": make([]interface{}, len(status.NetworkInfo.Interfaces)),
		}

		for i, iface := range status.NetworkInfo.Interfaces {
			interfaceInfo := map[string]interface{}{
				"name":      iface.Name,
				"is_up":     iface.IsUp,
				"addresses": iface.Addresses,
				"mtu":       iface.MTU,
			}
			if iface.BytesSent > 0 {
				interfaceInfo["bytes_sent"] = iface.BytesSent
			}
			if iface.BytesRecv > 0 {
				interfaceInfo["bytes_recv"] = iface.BytesRecv
			}
			networkInfo["interfaces"].([]interface{})[i] = interfaceInfo
		}

		if status.NetworkInfo.PublicIP != "" {
			networkInfo["public_ip"] = status.NetworkInfo.PublicIP
		}

		result["network"] = networkInfo
	}

	// Add Temperature information
	if status.Temperature != nil {
		tempInfo := map[string]interface{}{
			"unit": status.Temperature.Unit,
		}
		if status.Temperature.CPUTemp > 0 {
			tempInfo["cpu_temp"] = status.Temperature.CPUTemp
		}
		if status.Temperature.GPUTemp > 0 {
			tempInfo["gpu_temp"] = status.Temperature.GPUTemp
		}
		if status.Temperature.SystemTemp > 0 {
			tempInfo["system_temp"] = status.Temperature.SystemTemp
		}
		result["temperature"] = tempInfo
	}

	return result
}

// convertDeviceInfoToMCPFormat converts DeviceInfo to MCP format
func convertDeviceInfoToMCPFormat(deviceInfo *interfaces.DeviceInfo) map[string]interface{} {
	if deviceInfo == nil {
		return nil
	}

	return map[string]interface{}{
		"device_id":    deviceInfo.DeviceID,
		"hostname":     deviceInfo.Hostname,
		"platform":     deviceInfo.Platform,
		"architecture": deviceInfo.Architecture,
		"kernel_info":  deviceInfo.KernelInfo,
		"cpu_model":    deviceInfo.CPUModel,
		"cpu_cores":    deviceInfo.CPUCores,
		"total_memory": deviceInfo.TotalMemory,
		"boot_time":    deviceInfo.BootTime.Format(time.RFC3339),
		"timezone":     deviceInfo.Timezone,
	}
}

// convertEnergyDataToMCPFormat converts EnergyData to MCP format
func convertEnergyDataToMCPFormat(energyData *interfaces.EnergyData) map[string]interface{} {
	if energyData == nil {
		return nil
	}

	result := map[string]interface{}{
		"timestamp": energyData.Timestamp.Format(time.RFC3339),
	}

	// For overall energy data
	if energyData.EntityID == "" {
		result["total_power_consumption"] = energyData.TotalPowerConsumption
		result["total_energy_usage"] = energyData.TotalEnergyUsage
		result["total_cost"] = energyData.TotalCost
		result["ups_power_consumption"] = energyData.UPSPowerConsumption

		if energyData.DeviceBreakdown != nil {
			breakdown := make(map[string]interface{})
			for deviceID, deviceEnergy := range energyData.DeviceBreakdown {
				breakdown[deviceID] = map[string]interface{}{
					"device_name":       deviceEnergy.DeviceName,
					"power_consumption": deviceEnergy.PowerConsumption,
					"energy_usage":      deviceEnergy.EnergyUsage,
					"cost":              deviceEnergy.Cost,
					"state":             deviceEnergy.State,
					"is_on":             deviceEnergy.IsOn,
					"percentage":        deviceEnergy.Percentage,
				}
			}
			result["device_breakdown"] = breakdown
		}
	} else {
		// For device-specific energy data
		result["entity_id"] = energyData.EntityID
		result["device_name"] = energyData.DeviceName
		result["power_consumption"] = energyData.PowerConsumption
		result["energy_usage"] = energyData.EnergyUsage
		result["cost"] = energyData.Cost
		result["state"] = energyData.State
		result["is_on"] = energyData.IsOn

		if energyData.Current > 0 {
			result["current"] = energyData.Current
		}
		if energyData.Voltage > 0 {
			result["voltage"] = energyData.Voltage
		}
		if energyData.Frequency > 0 {
			result["frequency"] = energyData.Frequency
		}

		result["has_sensors"] = energyData.HasSensors
		if energyData.SensorsFound != nil {
			result["sensors_found"] = energyData.SensorsFound
		}

		if energyData.Percentage > 0 {
			result["percentage"] = energyData.Percentage
		}
	}

	return result
}

// convertEnergySettingsToMCPFormat converts EnergySettings to MCP format
func convertEnergySettingsToMCPFormat(settings *interfaces.EnergySettings) map[string]interface{} {
	if settings == nil {
		return nil
	}

	return map[string]interface{}{
		"energy_rate":       settings.EnergyRate,
		"currency":          settings.Currency,
		"tracking_enabled":  settings.TrackingEnabled,
		"update_interval":   settings.UpdateInterval,
		"historical_period": settings.HistoricalPeriod,
	}
}

// convertAutomationResultToMCPFormat converts AutomationResult to MCP format
func convertAutomationResultToMCPFormat(result *interfaces.AutomationResult) map[string]interface{} {
	if result == nil {
		return nil
	}

	return map[string]interface{}{
		"success":       result.Success,
		"automation_id": result.AutomationID,
		"name":          result.Name,
		"message":       result.Message,
		"created_at":    result.CreatedAt.Format(time.RFC3339),
		"note":          result.Note,
	}
}

// convertSceneResultToMCPFormat converts SceneResult to MCP format
func convertSceneResultToMCPFormat(result *interfaces.SceneResult) map[string]interface{} {
	if result == nil {
		return nil
	}

	return map[string]interface{}{
		"success":     result.Success,
		"scene_id":    result.SceneID,
		"message":     result.Message,
		"executed_at": result.ExecutedAt.Format(time.RFC3339),
		"note":        result.Note,
	}
}

// convertControlResultToMCPFormat converts PMAControlResult to MCP format
func convertControlResultToMCPFormat(result *types.PMAControlResult) map[string]interface{} {
	if result == nil {
		return nil
	}

	mcpResult := map[string]interface{}{
		"success":      result.Success,
		"entity_id":    result.EntityID,
		"action":       result.Action,
		"processed_at": result.ProcessedAt.Format(time.RFC3339),
		"duration":     result.Duration.Milliseconds(),
	}

	if result.NewState != "" {
		mcpResult["new_state"] = string(result.NewState)
	}

	if result.Attributes != nil {
		mcpResult["attributes"] = result.Attributes
	}

	if result.Error != nil {
		mcpResult["error"] = map[string]interface{}{
			"code":      result.Error.Code,
			"message":   result.Error.Message,
			"source":    result.Error.Source,
			"entity_id": result.Error.EntityID,
			"timestamp": result.Error.Timestamp.Format(time.RFC3339),
			"retryable": result.Error.Retryable,
		}

		if result.Error.Details != nil {
			mcpResult["error"].(map[string]interface{})["details"] = result.Error.Details
		}
	}

	return mcpResult
}
