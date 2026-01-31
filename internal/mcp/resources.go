package mcp

// Resource defines an MCP resource
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ResourceDefinitions lists all available resources
var ResourceDefinitions = []Resource{
	{
		URI:         "jobsearch://summary",
		Name:        "Job Search Summary",
		Description: "Current job search status overview with counts by status",
		MimeType:    "text/plain",
	},
	{
		URI:         "jobsearch://pending",
		Name:        "Pending Actions",
		Description: "Conversations requiring your attention (waiting_on_me and stale)",
		MimeType:    "text/plain",
	},
	{
		URI:         "jobsearch://recent",
		Name:        "Recent Activity",
		Description: "Last 10 conversations with recent activity",
		MimeType:    "text/plain",
	},
	{
		URI:         "jobsearch://companies",
		Name:        "Companies List",
		Description: "All companies you're in conversation with",
		MimeType:    "text/plain",
	},
}

// resourcesListResult is the response for resources/list
type resourcesListResult struct {
	Resources []Resource `json:"resources"`
}

// readResourceParams is the params for resources/read
type readResourceParams struct {
	URI string `json:"uri"`
}

// readResourceResult is the response for resources/read
type readResourceResult struct {
	Contents []resourceContent `json:"contents"`
}

type resourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}
