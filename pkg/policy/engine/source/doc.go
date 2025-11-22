// Package source provides policy sources for the policy engine.
//
// A policy source is responsible for loading and watching policies.
// This package provides file-based and in-memory implementations.
//
// # File Source
//
// The file source loads policies from YAML files on disk and watches
// for changes using fsnotify:
//
//	source := source.NewFileSource("policies/")
//	policies, err := source.LoadPolicies(ctx)
//
// # Hot-Reload
//
// File sources support hot-reload by watching for file system changes:
//
//	events, err := source.Watch(ctx)
//	for event := range events {
//	    if event.Error != nil {
//	        log.Error("watch error", "error", event.Error)
//	        continue
//	    }
//	    // Reload policies
//	    policies, err := source.LoadPolicies(ctx)
//	}
//
// # In-Memory Source
//
// The in-memory source is useful for testing:
//
//	source := source.NewMemorySource(policies...)
//	policies, err := source.LoadPolicies(ctx)
package source
