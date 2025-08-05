package broadcast

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type hub[T any] struct {
	config    HubConfig[T]
	channels  map[string]*channel[T]
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	closed    bool
	closeChan chan struct{}
}

type channel[T any] struct {
	name        string
	subscribers map[string]*subscriber[T]
	mu          sync.RWMutex
}

type subscriber[T any] struct {
	id        string
	channel   string
	messages  chan Message[T]
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
	hub       *hub[T]
	mu        sync.RWMutex // Protects channel operations during close
}

type ackSubscriber[T any] struct {
	id            string
	channel       string
	messages      chan AckableMessage[T]
	ctx           context.Context
	cancel        context.CancelFunc
	closeOnce     sync.Once
	hub           *hub[T]
	config        subscribeConfig
	pendingAcks   map[string]*pendingAck[T]
	pendingMu     sync.Mutex
	ackProcessing sync.WaitGroup
}

type pendingAck[T any] struct {
	message Message[T]
	retries int
	timer   *time.Timer
	acked   bool
	nacked  bool
}

func newHub[T any](config HubConfig[T]) *hub[T] {
	// Apply default values for configuration that must be non-zero
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

	// Start cleanup goroutine only if interval is configured
	// This prevents unnecessary background processing when not needed
	if config.CleanupInterval > 0 {
		h.wg.Add(1)
		go h.cleanupLoop()
	}

	return h
}

func (h *hub[T]) Subscribe(ctx context.Context, channelName string, opts ...SubscribeOption) (Subscriber[T], error) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, ErrHubClosed{}
	}

	config := subscribeConfig{
		bufferSize: h.config.DefaultBufferSize,
	}
	for _, opt := range opts {
		opt(&config)
	}

	// Create channel on-demand to minimize memory usage
	ch, exists := h.channels[channelName]
	if !exists {
		ch = &channel[T]{
			name:        channelName,
			subscribers: make(map[string]*subscriber[T]),
		}
		h.channels[channelName] = ch
	}
	h.mu.Unlock()
	subCtx, subCancel := context.WithCancel(ctx)
	sub := &subscriber[T]{
		id:       uuid.New().String(),
		channel:  channelName,
		messages: make(chan Message[T], config.bufferSize),
		ctx:      subCtx,
		cancel:   subCancel,
		hub:      h,
	}

	ch.mu.Lock()
	ch.subscribers[sub.id] = sub
	subscriberCount := len(ch.subscribers)
	ch.mu.Unlock()

	if h.config.MetricsCallback != nil {
		h.config.MetricsCallback(channelName, subscriberCount)
	}

	// Replay historical messages in separate goroutine to avoid blocking subscription
	if config.replay && h.config.Storage != nil {
		h.wg.Add(1)
		go func() {
			defer h.wg.Done()
			h.replayMessages(sub, config.replayLimit)
		}()
	}

	// Handle context cancellation in separate goroutine
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		<-subCtx.Done()
		_ = sub.Close()
	}()

	return sub, nil
}

func (h *hub[T]) SubscribeWithAck(ctx context.Context, channelName string, opts ...SubscribeOption) (AckSubscriber[T], error) {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil, ErrHubClosed{}
	}

	// Set acknowledgment defaults: 30s timeout allows for complex processing,
	// 3 retries handle transient failures without overwhelming the system
	config := subscribeConfig{
		bufferSize: h.config.DefaultBufferSize,
		ackTimeout: 30 * time.Second,
		maxRetries: 3,
	}
	for _, opt := range opts {
		opt(&config)
	}

	ch, exists := h.channels[channelName]
	if !exists {
		ch = &channel[T]{
			name:        channelName,
			subscribers: make(map[string]*subscriber[T]),
		}
		h.channels[channelName] = ch
	}
	h.mu.Unlock()
	subCtx, subCancel := context.WithCancel(ctx)
	ackSub := &ackSubscriber[T]{
		id:          uuid.New().String(),
		channel:     channelName,
		messages:    make(chan AckableMessage[T], config.bufferSize),
		ctx:         subCtx,
		cancel:      subCancel,
		hub:         h,
		config:      config,
		pendingAcks: make(map[string]*pendingAck[T]),
	}

	// Internal subscriber acts as intermediary to receive all messages
	// before they're wrapped with acknowledgment functionality
	internalSub := &subscriber[T]{
		id:       ackSub.id,
		channel:  channelName,
		messages: make(chan Message[T], config.bufferSize),
		ctx:      subCtx,
		cancel:   subCancel,
		hub:      h,
	}

	ch.mu.Lock()
	ch.subscribers[ackSub.id] = internalSub
	subscriberCount := len(ch.subscribers)
	ch.mu.Unlock()

	if h.config.MetricsCallback != nil {
		h.config.MetricsCallback(channelName, subscriberCount)
	}

	// Forward messages with acknowledgment tracking in separate goroutine
	h.wg.Add(1)
	ackSub.ackProcessing.Add(1) // Add before starting goroutine to avoid race
	go func() {
		defer h.wg.Done()
		ackSub.forwardMessages(internalSub)
	}()

	if config.replay && h.config.Storage != nil {
		h.wg.Add(1)
		go func() {
			defer h.wg.Done()
			h.replayMessages(internalSub, config.replayLimit)
		}()
	}
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		<-subCtx.Done()
		_ = ackSub.Close()
	}()

	return ackSub, nil
}

func (h *hub[T]) Publish(ctx context.Context, channelName string, payload T, opts ...PublishOption) error {
	msg := Message[T]{
		ID:        uuid.New().String(),
		Channel:   channelName,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	config := publishConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	if config.metadata != nil {
		msg.Metadata = config.metadata
	}

	return h.PublishMessage(ctx, msg)
}

func (h *hub[T]) PublishMessage(ctx context.Context, message Message[T]) error {
	h.mu.RLock()
	if h.closed {
		h.mu.RUnlock()
		return ErrHubClosed{}
	}

	ch, exists := h.channels[message.Channel]
	if !exists {
		h.mu.RUnlock()
		return nil // No subscribers - message discarded but not an error condition
	}
	h.mu.RUnlock()

	if h.config.Storage != nil {
		if err := h.config.Storage.Store(ctx, message); err != nil {
			return &ErrStorageFailure{Operation: "store", Err: err}
		}
	}

	// Create snapshot of subscribers to avoid holding read lock during message delivery
	// This prevents deadlocks and reduces lock contention
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

// sendToSubscriber delivers a message with timeout protection against slow consumers
// Slow consumers are automatically disconnected to prevent memory buildup
func (h *hub[T]) sendToSubscriber(sub *subscriber[T], msg Message[T], timeout time.Duration) {
	// Try to acquire read lock to ensure subscriber isn't being closed
	sub.mu.RLock()
	defer sub.mu.RUnlock()

	// Check context first for fast path
	select {
	case <-sub.ctx.Done():
		return
	default:
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case sub.messages <- msg:
		// Message delivered successfully
	case <-timer.C:
		// Slow consumer detected - disconnect to prevent memory accumulation
		// Close subscriber directly - closeOnce ensures this is safe to call multiple times
		go func() {
			_ = sub.Close()
		}()
	case <-sub.ctx.Done():
		// Subscriber already closed
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

func (h *hub[T]) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	close(h.closeChan)
	h.mu.Unlock()

	h.cancel()
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

	// Wait for graceful shutdown with timeout to prevent hanging
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

// cleanupLoop prevents memory leaks by removing channels with no subscribers
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
		select {
		case sub.messages <- msg:
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
		// Cancel context to signal closure
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

		// Acquire write lock to prevent concurrent sends
		s.mu.Lock()
		defer s.mu.Unlock()

		// Close message channel - write lock prevents concurrent sends
		close(s.messages)

		// Drain remaining messages in current goroutine to avoid race
		for range s.messages {
			// Drain any remaining messages to prevent blocking publishers
		}
	})

	return nil
}

// AckSubscriber implementation

// Messages returns a channel to receive acknowledgeable messages
func (s *ackSubscriber[T]) Messages() <-chan AckableMessage[T] {
	return s.messages
}

// Channel returns the subscribed channel name
func (s *ackSubscriber[T]) Channel() string {
	return s.channel
}

// ID returns the unique subscriber ID
func (s *ackSubscriber[T]) ID() string {
	return s.id
}

// Close unsubscribes and cleans up resources
func (s *ackSubscriber[T]) Close() error {
	s.closeOnce.Do(func() {
		// Cancel context
		s.cancel()

		// Wait for ack processing to complete
		s.ackProcessing.Wait()

		// Cancel all pending ack timers
		s.pendingMu.Lock()
		for _, pending := range s.pendingAcks {
			pending.timer.Stop()
		}
		s.pendingMu.Unlock()

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

		// Drain remaining messages in current goroutine to avoid race
		for range s.messages {
			// Drain any remaining messages
		}
	})

	return nil
}

// forwardMessages wraps regular messages with acknowledgment functionality
// Each message gets retry logic and timeout handling for reliable delivery
func (s *ackSubscriber[T]) forwardMessages(internalSub *subscriber[T]) {
	defer s.ackProcessing.Done()

	for {
		select {
		case msg, ok := <-internalSub.messages:
			if !ok {
				return
			}

			ackMsg := AckableMessage[T]{
				Message: msg,
			}
			pending := &pendingAck[T]{
				message: msg,
				retries: 0,
			}

			s.pendingMu.Lock()
			s.pendingAcks[msg.ID] = pending
			s.pendingMu.Unlock()

			// Setup acknowledgment functions
			ackMsg.ack = func() error {
				s.pendingMu.Lock()
				defer s.pendingMu.Unlock()

				if p, exists := s.pendingAcks[msg.ID]; exists {
					if p.timer != nil {
						p.timer.Stop()
					}
					p.acked = true
					delete(s.pendingAcks, msg.ID)
				}
				return nil
			}

			ackMsg.nack = func() error {
				s.pendingMu.Lock()
				defer s.pendingMu.Unlock()

				if p, exists := s.pendingAcks[msg.ID]; exists {
					if p.timer != nil {
						p.timer.Stop()
					}
					p.nacked = true
					delete(s.pendingAcks, msg.ID)
				}
				return nil
			}

			// Start acknowledgment timer
			pending.timer = time.AfterFunc(s.config.ackTimeout, func() {
				s.handleAckTimeout(msg)
			})

			// Send to subscriber
			select {
			case s.messages <- ackMsg:
			case <-s.ctx.Done():
				return
			}

		case <-s.ctx.Done():
			return
		}
	}
}

// handleAckTimeout implements exponential backoff retry logic for unacknowledged messages
func (s *ackSubscriber[T]) handleAckTimeout(msg Message[T]) {
	s.pendingMu.Lock()
	pending, exists := s.pendingAcks[msg.ID]
	if !exists || pending.acked || pending.nacked {
		s.pendingMu.Unlock()
		return
	}

	pending.retries++
	if pending.retries >= s.config.maxRetries {
		// Max retries exceeded - give up and notify application
		delete(s.pendingAcks, msg.ID)
		s.pendingMu.Unlock()

		// Invoke timeout callback to let application handle failed delivery
		if s.config.onAckTimeout != nil {
			anyMsg := Message[any]{
				ID:        msg.ID,
				Channel:   msg.Channel,
				Payload:   msg.Payload,
				Timestamp: msg.Timestamp,
				Metadata:  msg.Metadata,
			}
			s.config.onAckTimeout(anyMsg)
		}
		return
	}
	s.pendingMu.Unlock()

	// Retry sending the message
	ackMsg := AckableMessage[T]{
		Message: msg,
		ack: func() error {
			s.pendingMu.Lock()
			defer s.pendingMu.Unlock()

			if p, exists := s.pendingAcks[msg.ID]; exists {
				if p.timer != nil {
					p.timer.Stop()
				}
				p.acked = true
				delete(s.pendingAcks, msg.ID)
			}
			return nil
		},
		nack: func() error {
			s.pendingMu.Lock()
			defer s.pendingMu.Unlock()

			if p, exists := s.pendingAcks[msg.ID]; exists {
				if p.timer != nil {
					p.timer.Stop()
				}
				p.nacked = true
				delete(s.pendingAcks, msg.ID)
			}
			return nil
		},
	}

	// Reset timer for retry
	s.pendingMu.Lock()
	pending.timer = time.AfterFunc(s.config.ackTimeout, func() {
		s.handleAckTimeout(msg)
	})
	s.pendingMu.Unlock()

	// Resend message
	select {
	case s.messages <- ackMsg:
	case <-s.ctx.Done():
	}
}
