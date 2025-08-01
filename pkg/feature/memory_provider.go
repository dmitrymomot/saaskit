package feature

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

// MemoryProvider is an in-memory implementation of the Provider interface.
// All operations create deep copies to prevent external modification of stored flags.
type MemoryProvider struct {
	flags map[string]*Flag
	mu    sync.RWMutex
}

func NewMemoryProvider(initialFlags ...*Flag) (*MemoryProvider, error) {
	provider := &MemoryProvider{
		flags: make(map[string]*Flag),
	}

	for _, flag := range initialFlags {
		if flag == nil {
			continue
		}
		if flag.Name == "" {
			return nil, errors.Join(ErrInvalidFlag, errors.New("flag name cannot be empty"))
		}
		flagCopy := *flag

		if flagCopy.CreatedAt.IsZero() {
			flagCopy.CreatedAt = time.Now()
		}
		if flagCopy.UpdatedAt.IsZero() {
			flagCopy.UpdatedAt = flagCopy.CreatedAt
		}

		if flag.Tags != nil {
			flagCopy.Tags = slices.Clone(flag.Tags)
		}

		provider.flags[flag.Name] = &flagCopy
	}

	return provider, nil
}

func (m *MemoryProvider) IsEnabled(ctx context.Context, flagName string) (bool, error) {
	m.mu.RLock()
	flag, exists := m.flags[flagName]
	m.mu.RUnlock()

	if !exists {
		return false, ErrFlagNotFound
	}

	// Global disabled state overrides all strategies
	if !flag.Enabled {
		return false, nil
	}

	if flag.Strategy == nil {
		return flag.Enabled, nil
	}
	return flag.Strategy.Evaluate(ctx)
}

func (m *MemoryProvider) GetFlag(ctx context.Context, flagName string) (*Flag, error) {
	m.mu.RLock()
	flag, exists := m.flags[flagName]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrFlagNotFound
	}

	flagCopy := *flag
	if flag.Tags != nil {
		flagCopy.Tags = slices.Clone(flag.Tags)
	}
	return &flagCopy, nil
}

func (m *MemoryProvider) ListFlags(ctx context.Context, tags ...string) ([]*Flag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Flag

	if len(tags) == 0 {
		result = make([]*Flag, 0, len(m.flags))
		for _, flag := range m.flags {
			flagCopy := *flag
			if flag.Tags != nil {
				flagCopy.Tags = make([]string, len(flag.Tags))
				copy(flagCopy.Tags, flag.Tags)
			}
			result = append(result, &flagCopy)
		}
		return result, nil
	}

	// Filter by tags - flag matches if it has any of the requested tags
	result = make([]*Flag, 0, len(m.flags))
	for _, flag := range m.flags {
		for _, tagToMatch := range tags {
			if slices.Contains(flag.Tags, tagToMatch) {
				flagCopy := *flag
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

func (m *MemoryProvider) CreateFlag(ctx context.Context, flag *Flag) error {
	if flag == nil {
		return errors.Join(ErrInvalidFlag, errors.New("flag cannot be nil"))
	}
	if flag.Name == "" {
		return errors.Join(ErrInvalidFlag, errors.New("flag name cannot be empty"))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.flags[flag.Name]; exists {
		return errors.Join(ErrInvalidFlag, errors.New("flag already exists"))
	}

	now := time.Now()
	flag.CreatedAt = now
	flag.UpdatedAt = now

	flagCopy := *flag
	m.flags[flag.Name] = &flagCopy

	return nil
}

func (m *MemoryProvider) UpdateFlag(ctx context.Context, flag *Flag) error {
	if flag == nil {
		return errors.Join(ErrInvalidFlag, errors.New("flag cannot be nil"))
	}
	if flag.Name == "" {
		return errors.Join(ErrInvalidFlag, errors.New("flag name cannot be empty"))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.flags[flag.Name]
	if !exists {
		return ErrFlagNotFound
	}

	flag.CreatedAt = existing.CreatedAt
	flag.UpdatedAt = time.Now()

	flagCopy := *flag
	m.flags[flag.Name] = &flagCopy

	return nil
}

func (m *MemoryProvider) DeleteFlag(ctx context.Context, flagName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.flags[flagName]; !exists {
		return ErrFlagNotFound
	}

	delete(m.flags, flagName)

	return nil
}

func (m *MemoryProvider) Close() error {
	return nil
}
