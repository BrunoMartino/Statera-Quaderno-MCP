package observability

// DebugState is the read-only debug projection from the store plugin
// (GET /wp-json/statera-mcp/v1/debug). The toggle itself lives in the host
// (WORDPRESS_DEBUG on the container); this MCP only reports what took effect.
type DebugState struct {
	WPDebug        bool   `json:"wp_debug"`
	WPDebugLog     bool   `json:"wp_debug_log"`
	WPDebugDisplay bool   `json:"wp_debug_display"`
	LogPath        string `json:"log_path,omitempty"`
	LogSizeBytes   int64  `json:"log_size_bytes,omitempty"`
	LogWritable    bool   `json:"log_writable"`
	PluginVersion  string `json:"plugin_version,omitempty"`
	Notice         string `json:"notice,omitempty"`
}

// LogEntry is one normalized line across every source. Cron and Action Scheduler
// entries carry state (Schedule, NextRun, Status, Overdue); file sources do not.
type LogEntry struct {
	Source    string `json:"source"`
	Channel   string `json:"channel,omitempty"`
	Level     string `json:"level,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Message   string `json:"message"`
	Context   string `json:"context,omitempty"`
	Schedule  string `json:"schedule,omitempty"`
	NextRun   string `json:"next_run,omitempty"`
	Status    string `json:"status,omitempty"`
	Overdue   *bool  `json:"overdue,omitempty"`
}

// LogCollection is the collect_logs result.
type LogCollection struct {
	Source      string     `json:"source"`
	Entries     []LogEntry `json:"entries"`
	Truncated   bool       `json:"truncated"`
	SourcesRead []string   `json:"sources_read,omitempty"`
	Unavailable []string   `json:"unavailable,omitempty"`
}

// LogQuery is the allowlisted collect_logs filter.
type LogQuery struct {
	Source string
	Level  string
	Since  string
	Limit  int
	Search string
}
