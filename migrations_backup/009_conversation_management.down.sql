-- Drop conversation management schema
-- This migration removes all conversation and MCP related tables

-- Drop indexes first
DROP INDEX IF EXISTS idx_conversation_analytics_date;
DROP INDEX IF EXISTS idx_conversation_analytics_conversation_id;

DROP INDEX IF EXISTS idx_mcp_tool_executions_success;
DROP INDEX IF EXISTS idx_mcp_tool_executions_created_at;
DROP INDEX IF EXISTS idx_mcp_tool_executions_tool_id;
DROP INDEX IF EXISTS idx_mcp_tool_executions_conversation_id;

DROP INDEX IF EXISTS idx_mcp_tools_enabled;
DROP INDEX IF EXISTS idx_mcp_tools_category;
DROP INDEX IF EXISTS idx_mcp_tools_name;

DROP INDEX IF EXISTS idx_conversation_messages_role;
DROP INDEX IF EXISTS idx_conversation_messages_created_at;
DROP INDEX IF EXISTS idx_conversation_messages_conversation_id;

DROP INDEX IF EXISTS idx_conversations_archived;
DROP INDEX IF EXISTS idx_conversations_last_message_at;
DROP INDEX IF EXISTS idx_conversations_created_at;
DROP INDEX IF EXISTS idx_conversations_user_id;

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS conversation_analytics;
DROP TABLE IF EXISTS mcp_tool_executions;
DROP TABLE IF EXISTS mcp_tools;
DROP TABLE IF EXISTS conversation_messages;
DROP TABLE IF EXISTS conversations; 