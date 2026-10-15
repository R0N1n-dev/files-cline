package fs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// AppConfig is the persisted user configuration for Vela Files.
type AppConfig struct {
	Theme        string   `json:"theme"`    // "light" | "dark" | "system"
	ViewMode     string   `json:"viewMode"` // "grid" | "list"
	IconSize     int      `json:"iconSize"` // grid icon size in px
	SortBy       string   `json:"sortBy"`   // name | size | modified | type
	SortDesc     bool     `json:"sortDesc"`
	ShowHidden   bool     `json:"showHidden"`
	SplitView    bool     `json:"splitView"`
	ConflictMode string   `json:"conflictMode"` // rename | overwrite | skip | ask
	LastPath     string   `json:"lastPath"`
	Bookmarks    []string `json:"bookmarks"`
	SidebarWidth int      `json:"sidebarWidth"`
}

// ConfigStore persists AppConfig to $XDG_CONFIG_HOME/vela-files/config.json.
type ConfigStore struct {
	mu   sync.Mutex
	path string
	cfg  AppConfig
}

// DefaultConfig returns the sensible Linux desktop defaults.
func DefaultConfig() AppConfig {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "/workspace"
	}
	return AppConfig{
		Theme:        "system",
		ViewMode:     "grid",
		IconSize:     112,
		SortBy:       "name",
		SortDesc:     false,
		ShowHidden:   false,
		SplitView:    false,
		ConflictMode: ConflictRename,
		LastPath:     home,
		Bookmarks:    []string{},
		SidebarWidth: 232,
	}
}

// NewConfigStore loads (or creates) the config file at the default location.
func NewConfigStore() *ConfigStore {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, ".config")
		} else {
			dir = os.TempDir()
		}
	}
	store := &ConfigStore{
		path: filepath.Join(dir, "vela-files", "config.json"),
		cfg:  DefaultConfig(),
	}
	store.load()
	return store
}

func (c *ConfigStore) load() {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	c.mu.Lock()
	c.cfg = mergeConfig(DefaultConfig(), cfg)
	c.mu.Unlock()
}

func mergeConfig(base, override AppConfig) AppConfig {
	if override.Theme != "" {
		base.Theme = override.Theme
	}
	if override.ViewMode != "" {
		base.ViewMode = override.ViewMode
	}
	if override.IconSize >= 48 && override.IconSize <= 320 {
		base.IconSize = override.IconSize
	}
	if override.SortBy != "" {
		base.SortBy = override.SortBy
	}
	if override.ConflictMode != "" {
		base.ConflictMode = override.ConflictMode
	}
	if override.LastPath != "" {
		base.LastPath = override.LastPath
	}
	if override.SidebarWidth >= 160 && override.SidebarWidth <= 520 {
		base.SidebarWidth = override.SidebarWidth
	}
	base.SortDesc = override.SortDesc
	base.ShowHidden = override.ShowHidden
	base.SplitView = override.SplitView
	if override.Bookmarks != nil {
		base.Bookmarks = override.Bookmarks
	}
	return base
}

// Get returns a copy of the current configuration.
func (c *ConfigStore) Get() AppConfig {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := c.cfg
	cp.Bookmarks = append([]string{}, c.cfg.Bookmarks...)
	return cp
}

// Save merges and persists a configuration update.
func (c *ConfigStore) Save(update AppConfig) (AppConfig, error) {
	c.mu.Lock()
	c.cfg = mergeConfig(c.cfg, update)
	snapshot := c.cfg
	path := c.path
	c.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return snapshot, err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return snapshot, err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return snapshot, err
	}
	return snapshot, nil
}

// AddBookmark adds a path to bookmarks (deduplicated).
func (c *ConfigStore) AddBookmark(path string) ([]string, error) {
	abs, err := filepath.Abs(ExpandPath(path))
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	for _, b := range c.cfg.Bookmarks {
		if b == abs {
			c.mu.Unlock()
			return c.cfg.Bookmarks, nil
		}
	}
	c.cfg.Bookmarks = append(c.cfg.Bookmarks, abs)
	cfg := c.cfg
	c.mu.Unlock()
	saved, err := c.Save(cfg)
	return saved.Bookmarks, err
}

// RemoveBookmark removes a bookmark path.
func (c *ConfigStore) RemoveBookmark(path string) ([]string, error) {
	c.mu.Lock()
	kept := make([]string, 0, len(c.cfg.Bookmarks))
	for _, b := range c.cfg.Bookmarks {
		if b != path {
			kept = append(kept, b)
		}
	}
	c.cfg.Bookmarks = kept
	cfg := c.cfg
	c.mu.Unlock()
	saved, err := c.Save(cfg)
	return saved.Bookmarks, err
}
