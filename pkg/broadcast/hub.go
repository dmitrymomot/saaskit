package broadcast

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// hub implements the Hub interface
type hub[T any] struct {
	config    HubConfig
	channels  map[string]*channel[T]
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closed    bool
	closeChan chan struct{}
}

// channel manages subscribers for a specific channel
type channel[T any] struct {
	name        string
	subscribers map[string]*subscriber[T]
	mu          sync.RWMutex
}

// subscriber implements the Subscriber interface
type subscriber[T any] struct {
	id        string
	channel   string
	messages  chan Message[T]
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
	hub       *hub[T]
}

// newHub creates a new hub instance
func newHub[T any](config HubConfig) *hub[T] {
	if config.DefaultBufferSize <= 0 {
		config.DefaultBufferSize = 100
	}
	if config.SlowConsumerTimeout <= 0 {
		config.SlowConsumerTimeout = 5 * time.Second
	}
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = 30 * time.Second
	}
	if config.ReplayTimeout <= 0 {
		config.ReplayTimeout = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())
	h := &hub[T]{
		config:    config,
		channels:  make(map[string]*channel[T]),
		ctx:       ctx,
		cancel:    cancel,
		closeChan: make(chan struct{}),
	}

	if config.CleanupInterval > 0 {
		h.wg.Add(1)
		go h.cleanupLoop()
	}

	return h
}

// Subscribe creates a new subscription to a channel
func (h *hub[T]) Subscribe(ctx context.Context, channelName string, opts ...SubscribeOption) (Subscriber[T], error) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, ErrHubClosed{}
	}

	// Apply options
	config := subscribeConfig{
		bufferSize: h.config.DefaultBufferSize,
	}
	for _, opt := range opts {
		opt(&config)
	}

	// Get or create channel
	ch, exists := h.channels[channelName]
	if !exists {
		ch = &channel[T]{
			name:        channelName,
			subscribers: make(map[string]*subscriber[T]),
		}
		h.channels[channelName] = ch
	}
	h.mu.Unlock()

	// Create subscriber
	subCtx, subCancel := context.WithCancel(ctx)
	sub := &subscriber[T]{
		id:       uuid.New().String(),
		channel:  channelName,
		messages: make(chan Message[T], config.bufferSize),
		ctx:      subCtx,
		cancel:   subCancel,
		hub:      h,
	}

	// Register subscriber
	ch.mu.Lock()
	ch.subscribers[sub.id] = sub
	subscriberCount := len(ch.subscribers)
	ch.mu.Unlock()

	// Call metrics callback if configured
	if h.config.MetricsCallback != nil {
		h.config.MetricsCallback(channelName, subscriberCount)
	}

	// Handle replay if requested
	if config.replay && h.config.Storage != nil {
		h.wg.Add(1)
		go func() {
			defer h.wg.Done()
			h.replayMessages(sub, config.replayLimit)
		}()
	}

	// Handle context cancellation
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		<-subCtx.Done()
		_ = sub.Close() // Error already logged if any
	}()

	return sub, nil
}

// Publish sends a message to all subscribers of a channel
func (h *hub[T]) Publish(ctx context.Context, channelName string, payload T, opts ...PublishOption) error {
	msg := Message[T]{
		ID:        uuid.New().String(),
		Channel:   channelName,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	// Apply options
	config := publishConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	if config.metadata != nil {
		msg.Metadata = config.metadata
	}

	return h.PublishMessage(ctx, msg)
}

// PublishMessage sends a pre-built message
func (h *hub[T]) PublishMessage(ctx context.Context, message Message[T]) error {
	h.mu.RLock()
	if h.closed {
		h.mu.RUnlock()
		return ErrHubClosed{}
	}

	ch, exists := h.channels[message.Channel]
	if !exists {
		h.mu.RUnlock()
		return nil // No subscribers, not an error
	}
	h.mu.RUnlock()

	// Store message if storage is configured
	if h.config.Storage != nil {
		if err := h.config.Storage.Store(ctx, Message[any]{
			ID:        message.ID,
			Channel:   message.Channel,
			Payload:   message.Payload,
			Timestamp: message.Timestamp,
			Metadata:  message.Metadata,
		}); err != nil {
			return &ErrStorageFailure{Operation: "store", Err: err}
		}
	}

	// Broadcast to subscribers
	ch.mu.RLock()
	subscribers := make([]*subscriber[T], 0, len(ch.subscribers))
	for _, sub := range ch.subscribers {
		subscribers = append(subscribers, sub)
	}
	ch.mu.RUnlock()

	// Send to each subscriber with timeout
	timeout := h.config.SlowConsumerTimeout
	for _, sub := range subscribers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.closeChan:
			return ErrHubClosed{}
		default:
			h.sendToSubscriber(sub, message, timeout)
		}
	}

	return nil
}

// sendToSubscriber sends a message to a single subscriber with timeout
func (h *hub[T]) sendToSubscriber(sub *subscriber[T], msg Message[T], timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case sub.messages <- msg:
		// Success
	case <-timer.C:
		// Slow consumer
		// Try to close slow consumer once
		closed := false
		sub.closeOnce.Do(func() {
			closed = true
		})
		if closed {
			// First time detecting slow consumer
			go func() {
				_ = sub.Close() // Error already logged if any
			}()
		}
	case <-sub.ctx.Done():
		// Subscriber closed
	}
}

// Channels returns a list of active channels
func (h *hub[T]) Channels() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	channels := make([]string, 0, len(h.channels))
	for name := range h.channels {
		channels = append(channels, name)
	}
	return channels
}

// SubscriberCount returns the number of subscribers for a channel
func (h *hub[T]) SubscriberCount(channelName string) int {
	h.mu.RLock()
	ch, exists := h.channels[channelName]
	h.mu.RUnlock()

	if !exists {
		return 0
	}

	ch.mu.RLock()
	count := len(ch.subscribers)
	ch.mu.RUnlock()

	return count
}

// Close gracefully shuts down the hub
func (h *hub[T]) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	close(h.closeChan)
	h.mu.Unlock()

	// Cancel context to stop background goroutines
	h.cancel()

	// Close all subscribers
	h.mu.RLock()
	channels := make([]*channel[T], 0, len(h.channels))
	for _, ch := range h.channels {
		channels = append(channels, ch)
	}
	h.mu.RUnlock()

	for _, ch := range channels {
		ch.mu.RLock()
		subscribers := make([]*subscriber[T], 0, len(ch.subscribers))
		for _, sub := range ch.subscribers {
			subscribers = append(subscribers, sub)
		}
		ch.mu.RUnlock()

		for _, sub := range subscribers {
			_ = sub.Close() // Best effort cleanup during shutdown
		}
	}

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(h.config.ShutdownTimeout):
		return ErrShutdownTimeout{}
	}
}

// cleanupLoop periodically cleans up empty channels
func (h *hub[T]) cleanupLoop() {
	defer h.wg.Done()
	ticker := time.NewTicker(h.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.cleanupEmptyChannels()
		case <-h.ctx.Done():
			return
		}
	}
}

// cleanupEmptyChannels removes channels with no subscribers
func (h *hub[T]) cleanupEmptyChannels() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for name, ch := range h.channels {
		ch.mu.RLock()
		empty := len(ch.subscribers) == 0
		ch.mu.RUnlock()

		if empty {
			delete(h.channels, name)
		}
	}
}

// replayMessages replays recent messages to a new subscriber
func (h *hub[T]) replayMessages(sub *subscriber[T], limit int) {
	if h.config.Storage == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.config.ReplayTimeout)
	defer cancel()

	messages, err := h.config.Storage.Load(ctx, sub.channel, LoadOptions{
		Limit: limit,
	})
	if err != nil {
		return
	}

	for _, msg := range messages {
		// Type assertion with safety check
		payload, ok := any(msg.Payload).(T)
		if !ok {
			// Skip messages that can't be cast to expected type
			continue
		}

		select {
		case sub.messages <- Message[T]{
			ID:        msg.ID,
			Channel:   msg.Channel,
			Payload:   payload,
			Timestamp: msg.Timestamp,
			Metadata:  msg.Metadata,
		}:
		case <-sub.ctx.Done():
			return
		}
	}
}

// Subscriber implementation

// Messages returns a channel to receive messages
func (s *subscriber[T]) Messages() <-chan Message[T] {
	return s.messages
}

// Channel returns the subscribed channel name
func (s *subscriber[T]) Channel() string {
	return s.channel
}

// ID returns the unique subscriber ID
func (s *subscriber[T]) ID() string {
	return s.id
}

// Close unsubscribes and cleans up resources
func (s *subscriber[T]) Close() error {
	s.closeOnce.Do(func() {
		// Cancel context
		s.cancel()

		// Remove from hub
		s.hub.mu.RLock()
		ch, exists := s.hub.channels[s.channel]
		s.hub.mu.RUnlock()

		if exists {
			ch.mu.Lock()
			delete(ch.subscribers, s.id)
			subscriberCount := len(ch.subscribers)
			ch.mu.Unlock()

			// Call metrics callback if configured
			if s.hub.config.MetricsCallback != nil {
				s.hub.config.MetricsCallback(s.channel, subscriberCount)
			}
		}

		// Close message channel
		close(s.messages)
	})

	return nil
}
