package feature

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

// MemoryProvider is an in-memory implementation of the Provider interface.
// It's useful for testing and simple applications.
type MemoryProvider struct {
	flags map[string]*Flag
	mu    sync.RWMutex
}

// NewMemoryProvider creates a new in-memory feature flag provider.
func NewMemoryProvider(initialFlags ...*Flag) (*MemoryProvider, error) {
	provider := &MemoryProvider{
		flags: make(map[string]*Flag),
	}

	// Add initial flags if provided
	for _, flag := range initialFlags {
		if flag == nil {
			continue
		}
		if flag.Name == "" {
			return nil, errors.Join(ErrInvalidFlag, errors.New("flag name cannot be empty"))
		}
		// Create a deep copy of the flag
		flagCopy := *flag

		// Set timestamps if not already set
		if flagCopy.CreatedAt.IsZero() {
			flagCopy.CreatedAt = time.Now()
		}
		if flagCopy.UpdatedAt.IsZero() {
			flagCopy.UpdatedAt = flagCopy.CreatedAt
		}

		// Make a deep copy of the Tags slice
		if flag.Tags != nil {
			flagCopy.Tags = slices.Clone(flag.Tags)
		}

		// Store the copy
		provider.flags[flag.Name] = &flagCopy
	}

	return provider, nil
}

// IsEnabled checks if a flag is enabled for the given context.
func (m *MemoryProvider) IsEnabled(ctx context.Context, flagName string) (bool, error) {
	m.mu.RLock()
	flag, exists := m.flags[flagName]
	m.mu.RUnlock()

	if !exists {
		return false, ErrFlagNotFound
	}

	// If the flag is globally disabled, return false immediately
	if !flag.Enabled {
		return false, nil
	}

	// If no strategy is set, the flag is simply enabled/disabled globally
	if flag.Strategy == nil {
		return flag.Enabled, nil
	}

	// Evaluate the flag's strategy
	return flag.Strategy.Evaluate(ctx)
}

// GetFlag retrieves a flag by name.
func (m *MemoryProvider) GetFlag(ctx context.Context, flagName string) (*Flag, error) {
	m.mu.RLock()
	flag, exists := m.flags[flagName]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrFlagNotFound
	}

	// Return a copy to prevent external modification
	flagCopy := *flag
	// Create a copy of the Tags slice to prevent modification of the original
	if flag.Tags != nil {
		flagCopy.Tags = slices.Clone(flag.Tags)
	}
	return &flagCopy, nil
}

// ListFlags returns all flags, optionally filtered by tags.
func (m *MemoryProvider) ListFlags(ctx context.Context, tags ...string) ([]*Flag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Flag

	// If no tags specified, return all flags
	if len(tags) == 0 {
		result = make([]*Flag, 0, len(m.flags))
		for _, flag := range m.flags {
			// Return copies to prevent external modification
			flagCopy := *flag
			// Create a copy of the Tags slice to prevent modification of the original
			if flag.Tags != nil {
				flagCopy.Tags = make([]string, len(flag.Tags))
				copy(flagCopy.Tags, flag.Tags)
			}
			result = append(result, &flagCopy)
		}
		return result, nil
	}

	// Filter flags by tags
	result = make([]*Flag, 0)
	for _, flag := range m.flags {
		// Check if flag has any of the specified tags
		for _, tagToMatch := range tags {
			if slices.Contains(flag.Tags, tagToMatch) {
				// Flag matches at least one tag, add it and move to next flag
				flagCopy := *flag
				// Create a copy of the Tags slice to prevent modification of the original
				if flag.Tags != nil {
					flagCopy.Tags = slices.Clone(flag.Tags)
				}
				result = append(result, &flagCopy)
				goto nextFlag
			}
		}
	nextFlag:
	}

	return result, nil
}

// CreateFlag creates a new flag.
func (m *MemoryProvider) CreateFlag(ctx context.Context, flag *Flag) error {
	if flag == nil {
		return errors.Join(ErrInvalidFlag, errors.New("flag cannot be nil"))
	}
	if flag.Name == "" {
		return errors.Join(ErrInvalidFlag, errors.New("flag name cannot be empty"))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if flag already exists
	if _, exists := m.flags[flag.Name]; exists {
		return errors.Join(ErrInvalidFlag, errors.New("flag already exists"))
	}

	// Set timestamps
	now := time.Now()
	flag.CreatedAt = now
	flag.UpdatedAt = now

	// Store a copy to prevent external modification
	flagCopy := *flag
	m.flags[flag.Name] = &flagCopy

	return nil
}

// UpdateFlag updates an existing flag.
func (m *MemoryProvider) UpdateFlag(ctx context.Context, flag *Flag) error {
	if flag == nil {
		return errors.Join(ErrInvalidFlag, errors.New("flag cannot be nil"))
	}
	if flag.Name == "" {
		return errors.Join(ErrInvalidFlag, errors.New("flag name cannot be empty"))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if flag exists
	existing, exists := m.flags[flag.Name]
	if !exists {
		return ErrFlagNotFound
	}

	// Preserve original creation time
	flag.CreatedAt = existing.CreatedAt
	flag.UpdatedAt = time.Now()

	// Store a copy to prevent external modification
	flagCopy := *flag
	m.flags[flag.Name] = &flagCopy

	return nil
}

// DeleteFlag removes a flag.
func (m *MemoryProvider) DeleteFlag(ctx context.Context, flagName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if flag exists
	if _, exists := m.flags[flagName]; !exists {
		return ErrFlagNotFound
	}

	// Remove the flag
	delete(m.flags, flagName)

	return nil
}

// Close releases any resources. For the memory provider, this is a no-op.
func (m *MemoryProvider) Close() error {
	// No resources to release for memory provider
	return nil
}
