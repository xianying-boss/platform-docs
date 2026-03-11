// Package registry maintains the in-memory catalogue of available tools.
package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sandbox/platform/pkg/types"
)

// Registry is a thread-safe store of tool manifests.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]types.ToolManifest
}

// New creates an empty Registry and loads the 12 built-in tools.
func New() *Registry {
	r := &Registry{tools: make(map[string]types.ToolManifest)}
	for _, m := range builtinTools() {
		r.tools[m.Name] = m
	}
	return r
}

// LoadDir scans a directory for manifest.json files and registers all tools.
func (r *Registry) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name(), "manifest.json")
		if err := r.loadManifest(path); err != nil {
			return fmt.Errorf("load manifest %s: %w", path, err)
		}
	}
	return nil
}

// Get returns the manifest for a tool, or an error if not found.
func (r *Registry) Get(name string) (types.ToolManifest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.tools[name]
	if !ok {
		return types.ToolManifest{}, fmt.Errorf("tool %q not registered", name)
	}
	return m, nil
}

// All returns a snapshot of all registered manifests.
func (r *Registry) All() []types.ToolManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]types.ToolManifest, 0, len(r.tools))
	for _, m := range r.tools {
		out = append(out, m)
	}
	return out
}

func (r *Registry) loadManifest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var m types.ToolManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	r.mu.Lock()
	r.tools[m.Name] = m
	r.mu.Unlock()
	return nil
}

// builtinTools returns manifests for the 12 starter tools.
func builtinTools() []types.ToolManifest {
	return []types.ToolManifest{
		{Name: "html_parse",        Tier: types.TierWASM,    Entrypoint: "html_parse.wasm",       TimeoutSecs: 10},
		{Name: "json_parse",        Tier: types.TierWASM,    Entrypoint: "json_parse.wasm",        TimeoutSecs: 10},
		{Name: "markdown_convert",  Tier: types.TierWASM,    Entrypoint: "markdown_convert.wasm",  TimeoutSecs: 10},
		{Name: "docx_generate",     Tier: types.TierWASM,    Entrypoint: "docx_generate.wasm",     TimeoutSecs: 30},
		{Name: "python_run",        Tier: types.TierMicroVM, Entrypoint: "main.py",                TimeoutSecs: 60},
		{Name: "bash_run",          Tier: types.TierMicroVM, Entrypoint: "run.sh",                 TimeoutSecs: 60},
		{Name: "git_clone",         Tier: types.TierMicroVM, Entrypoint: "clone.sh",               TimeoutSecs: 120},
		{Name: "file_ops",          Tier: types.TierMicroVM, Entrypoint: "file_ops.sh",            TimeoutSecs: 30},
		{Name: "browser_open",      Tier: types.TierGUI,     Entrypoint: "browser.py",             TimeoutSecs: 120},
		{Name: "web_scrape",        Tier: types.TierGUI,     Entrypoint: "scrape.py",              TimeoutSecs: 120},
		{Name: "excel_edit",        Tier: types.TierGUI,     Entrypoint: "excel.py",               TimeoutSecs: 60},
		{Name: "office_automation", Tier: types.TierGUI,     Entrypoint: "office.py",              TimeoutSecs: 300},
	}
}
