-- Conversation Management Migration
-- Persistent conversation management, multi-turn chat history, and MCP (Model Context Protocol) support

-- Conversations table - stores conversation sessions
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    system_prompt TEXT,
    provider TEXT NOT NULL DEFAULT 'auto',
    model TEXT,
    temperature REAL DEFAULT 0.7,
    max_tokens INTEGER DEFAULT 2000,
    context_data TEXT, -- JSON blob for conversation context
    metadata TEXT, -- JSON blob for additional metadata
    message_count INTEGER DEFAULT 0,
    last_message_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    archived BOOLEAN DEFAULT FALSE,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Conversation messages table - stores individual messages
CREATE TABLE IF NOT EXISTS conversation_messages (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content TEXT NOT NULL,
    tool_calls TEXT, -- JSON array of tool calls (for MCP support)
    tool_call_id TEXT, -- Reference to tool call this message responds to
    tokens_used INTEGER DEFAULT 0,
    model_used TEXT,
    provider_used TEXT,
    response_time_ms INTEGER,
    metadata TEXT, -- JSON blob for additional metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- MCP tools table - stores available tools/functions for Model Context Protocol
CREATE TABLE IF NOT EXISTS mcp_tools (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT UNIQUE NOT NULL,
    description TEXT NOT NULL,
    schema TEXT NOT NULL, -- JSON schema for tool parameters
    handler TEXT NOT NULL, -- Handler function/method name
    category TEXT DEFAULT 'general',
    enabled BOOLEAN DEFAULT TRUE,
    usage_count INTEGER DEFAULT 0,
    last_used DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- MCP tool executions table - logs tool execution history
CREATE TABLE IF NOT EXISTS mcp_tool_executions (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    conversation_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    tool_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    parameters TEXT NOT NULL, -- JSON parameters passed to tool
    result TEXT, -- Tool execution result
    error TEXT, -- Error message if execution failed
    execution_time_ms INTEGER,
    success BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES conversation_messages(id) ON DELETE CASCADE,
    FOREIGN KEY (tool_id) REFERENCES mcp_tools(id) ON DELETE CASCADE
);

-- Conversation analytics table - stores conversation metrics and analytics
CREATE TABLE IF NOT EXISTS conversation_analytics (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    conversation_id TEXT NOT NULL,
    total_messages INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    total_cost REAL DEFAULT 0.0,
    avg_response_time_ms REAL DEFAULT 0.0,
    tools_used INTEGER DEFAULT 0,
    providers_used TEXT, -- JSON array of providers used
    models_used TEXT, -- JSON array of models used
    sentiment_score REAL, -- Overall conversation sentiment
    complexity_score REAL, -- Conversation complexity metric
    satisfaction_rating INTEGER, -- User satisfaction (1-5 if provided)
    date DATE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    UNIQUE(conversation_id, date)
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at);
CREATE INDEX IF NOT EXISTS idx_conversations_last_message_at ON conversations(last_message_at);
CREATE INDEX IF NOT EXISTS idx_conversations_archived ON conversations(archived);

CREATE INDEX IF NOT EXISTS idx_conversation_messages_conversation_id ON conversation_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conversation_messages_created_at ON conversation_messages(created_at);
CREATE INDEX IF NOT EXISTS idx_conversation_messages_role ON conversation_messages(role);

CREATE INDEX IF NOT EXISTS idx_mcp_tools_name ON mcp_tools(name);
CREATE INDEX IF NOT EXISTS idx_mcp_tools_category ON mcp_tools(category);
CREATE INDEX IF NOT EXISTS idx_mcp_tools_enabled ON mcp_tools(enabled);

CREATE INDEX IF NOT EXISTS idx_mcp_tool_executions_conversation_id ON mcp_tool_executions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_executions_tool_id ON mcp_tool_executions(tool_id);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_executions_created_at ON mcp_tool_executions(created_at);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_executions_success ON mcp_tool_executions(success);

CREATE INDEX IF NOT EXISTS idx_conversation_analytics_conversation_id ON conversation_analytics(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conversation_analytics_date ON conversation_analytics(date);

-- Insert default MCP tools for PMA functionality
INSERT OR IGNORE INTO mcp_tools (name, description, schema, handler, category) VALUES
('get_entity_state', 'Get the current state of a Home Assistant entity', 
 '{"type":"object","properties":{"entity_id":{"type":"string","description":"Entity ID to query"}},"required":["entity_id"]}',
 'GetEntityState', 'home_assistant'),

('set_entity_state', 'Set the state of a Home Assistant entity', 
 '{"type":"object","properties":{"entity_id":{"type":"string","description":"Entity ID to control"},"state":{"type":"string","description":"New state value"},"attributes":{"type":"object","description":"Optional attributes"}},"required":["entity_id","state"]}',
 'SetEntityState', 'home_assistant'),

('get_room_entities', 'Get all entities in a specific room', 
 '{"type":"object","properties":{"room_id":{"type":"string","description":"Room ID to query"}},"required":["room_id"]}',
 'GetRoomEntities', 'home_assistant'),

('create_automation', 'Create a new automation rule', 
 '{"type":"object","properties":{"name":{"type":"string","description":"Automation name"},"triggers":{"type":"array","description":"Trigger conditions"},"actions":{"type":"array","description":"Actions to perform"}},"required":["name","triggers","actions"]}',
 'CreateAutomation', 'automation'),

('get_system_status', 'Get current system status and health information', 
 '{"type":"object","properties":{},"required":[]}',
 'GetSystemStatus', 'system'),

('get_energy_data', 'Get current energy consumption data', 
 '{"type":"object","properties":{"device_id":{"type":"string","description":"Optional specific device ID"}},"required":[]}',
 'GetEnergyData', 'energy'),

('analyze_patterns', 'Analyze usage patterns for entities or system', 
 '{"type":"object","properties":{"entity_ids":{"type":"array","items":{"type":"string"},"description":"Entity IDs to analyze"},"time_range":{"type":"string","description":"Time range for analysis"},"analysis_type":{"type":"string","description":"Type of analysis"}},"required":["entity_ids"]}',
 'AnalyzePatterns', 'analytics'),

('execute_scene', 'Execute a Home Assistant scene', 
 '{"type":"object","properties":{"scene_id":{"type":"string","description":"Scene ID to execute"}},"required":["scene_id"]}',
 'ExecuteScene', 'home_assistant'); 