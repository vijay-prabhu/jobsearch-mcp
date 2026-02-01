package mcp

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolDefinitions contains all available MCP tools
var ToolDefinitions = []Tool{
	{
		Name:        "list_conversations",
		Description: "List job search conversations with optional filters. Returns conversations sorted by last activity.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"waiting_on_me", "waiting_on_them", "stale", "active", "closed", "all"},
					"description": "Filter by conversation status. Use 'all' or omit for no filter.",
				},
				"company": map[string]interface{}{
					"type":        "string",
					"description": "Filter by company name (case-insensitive partial match)",
				},
				"since_days": map[string]interface{}{
					"type":        "integer",
					"description": "Only show conversations with activity in the last N days",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 20)",
				},
				"include_archived": map[string]interface{}{
					"type":        "boolean",
					"description": "Include archived conversations (default: false)",
				},
			},
		},
	},
	{
		Name:        "get_conversation",
		Description: "Get detailed information about a specific conversation including email timeline.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"description": "Company name (case-insensitive) or conversation ID",
				},
			},
			"required": []string{"identifier"},
		},
	},
	{
		Name:        "get_pending_actions",
		Description: "Get conversations that need your attention - either waiting for your response or stale.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"include_stale": map[string]interface{}{
					"type":        "boolean",
					"description": "Include stale conversations that may need follow-up (default: true)",
				},
			},
		},
	},
	{
		Name:        "search_conversations",
		Description: "Search across all conversations by company name, recruiter, position, or email subject.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query text",
				},
			},
			"required": []string{"query"},
		},
	},
	{
		Name:        "get_stats",
		Description: "Get aggregate statistics about your job search including conversation counts and response rates.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"since_days": map[string]interface{}{
					"type":        "integer",
					"description": "Calculate stats for the last N days only",
				},
				"detailed": map[string]interface{}{
					"type":        "boolean",
					"description": "Include detailed breakdown with company stats and activity chart (default: false)",
				},
			},
		},
	},
	{
		Name:        "merge_conversations",
		Description: "Merge two conversations into one. All emails from the source conversation are moved to the target.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]interface{}{
					"type":        "string",
					"description": "Target conversation (company name or ID) - emails will be merged into this",
				},
				"source": map[string]interface{}{
					"type":        "string",
					"description": "Source conversation (company name or ID) - will be deleted after merge",
				},
			},
			"required": []string{"target", "source"},
		},
	},
	{
		Name:        "archive_conversation",
		Description: "Archive or unarchive a conversation. Archived conversations are hidden from default list output.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"identifier": map[string]interface{}{
					"type":        "string",
					"description": "Company name or conversation ID",
				},
				"unarchive": map[string]interface{}{
					"type":        "boolean",
					"description": "Set to true to unarchive instead of archive (default: false)",
				},
			},
			"required": []string{"identifier"},
		},
	},
}
