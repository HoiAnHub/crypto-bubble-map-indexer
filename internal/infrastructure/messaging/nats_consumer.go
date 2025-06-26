package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"crypto-bubble-map-indexer/internal/domain/entity"
	"crypto-bubble-map-indexer/internal/infrastructure/config"
	"crypto-bubble-map-indexer/internal/infrastructure/logger"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// NATSConsumer handles NATS JetStream consumption
type NATSConsumer struct {
	conn      *nats.Conn
	js        nats.JetStreamContext
	sub       *nats.Subscription
	config    *config.NATSConfig
	logger    *logger.Logger
	msgChan   chan *entity.Transaction
	isRunning bool
}

// NewNATSConsumer creates a new NATS consumer
func NewNATSConsumer(cfg *config.NATSConfig, logger *logger.Logger) *NATSConsumer {
	return &NATSConsumer{
		config:  cfg,
		logger:  logger.WithComponent("nats-consumer"),
		msgChan: make(chan *entity.Transaction, cfg.MaxPendingMessages),
	}
}

// Connect connects to NATS server and sets up consumer
func (n *NATSConsumer) Connect(ctx context.Context) error {
	if !n.config.Enabled {
		n.logger.Info("NATS is disabled, skipping connection")
		return nil
	}

	n.logger.Info("Connecting to NATS server", zap.String("url", n.config.URL))

	// Connect to NATS
	opts := []nats.Option{
		nats.Name("bubble-map-indexer"),
		nats.Timeout(n.config.ConnectTimeout),
		nats.ReconnectWait(n.config.ReconnectDelay),
		nats.MaxReconnects(n.config.ReconnectAttempts),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			n.logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			n.logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			n.logger.Info("NATS connection closed")
		}),
	}

	conn, err := nats.Connect(n.config.URL, opts...)
	if err != nil {
		n.logger.Error("Failed to connect to NATS", zap.Error(err))
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	n.conn = conn

	// Try JetStream first, if not available fall back to core NATS
	js, err := conn.JetStream()
	if err != nil {
		n.logger.Warn("JetStream not available, using core NATS", zap.Error(err))
		return n.setupCoreNATSSubscription()
	}

	n.js = js
	return n.setupJetStreamSubscription()
}

// setupJetStreamSubscription sets up JetStream subscription
func (n *NATSConsumer) setupJetStreamSubscription() error {
	subject := fmt.Sprintf("%s.events", n.config.SubjectPrefix)

	n.logger.Info("Setting up JetStream subscription",
		zap.String("subject", subject))

	// Use existing consumer "example-consumer" for WorkQueue streams
	existingConsumer := "example-consumer"

	n.logger.Info("Using existing JetStream consumer",
		zap.String("consumer", existingConsumer),
		zap.String("stream", n.config.StreamName))

	// Use PullSubscribe with existing consumer
	sub, err := n.js.PullSubscribe(subject, existingConsumer, nats.Bind(n.config.StreamName, existingConsumer))
	if err != nil {
		n.logger.Warn("Failed to bind to existing consumer, falling back to core NATS", zap.Error(err))
		return n.setupCoreNATSSubscription()
	}

	n.sub = sub
	n.isRunning = true

	// Start message processing
	go n.processJetStreamMessages()

	n.logger.Info("Successfully connected to NATS JetStream with existing consumer",
		zap.String("subject", subject),
		zap.String("consumer", existingConsumer))

	return nil
}

// processJetStreamMessages processes messages from JetStream pull subscription
func (n *NATSConsumer) processJetStreamMessages() {
	n.logger.Info("Starting JetStream message processing")

	for n.isRunning {
		// Fetch messages in batches
		msgs, err := n.sub.Fetch(10, nats.MaxWait(5*1000000000)) // 5 seconds timeout
		if err != nil {
			if err == nats.ErrTimeout {
				n.logger.Debug("No messages available, continuing...")
				continue
			}
			n.logger.Error("Failed to fetch messages", zap.Error(err))
			continue
		}

		n.logger.Debug("Fetched messages from JetStream", zap.Int("count", len(msgs)))

		for _, msg := range msgs {
			n.handleMessage(msg)
		}
	}

	n.logger.Info("Stopped JetStream message processing")
}

// setupCoreNATSSubscription sets up core NATS subscription
func (n *NATSConsumer) setupCoreNATSSubscription() error {
	subject := fmt.Sprintf("%s.events", n.config.SubjectPrefix)
	queueGroup := n.config.ConsumerGroup

	n.logger.Info("Setting up core NATS subscription",
		zap.String("subject", subject),
		zap.String("queue_group", queueGroup))

	sub, err := n.conn.QueueSubscribe(subject, queueGroup, func(msg *nats.Msg) {
		n.handleMessage(msg)
	})

	if err != nil {
		n.logger.Error("Failed to subscribe to subject", zap.Error(err))
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	n.sub = sub
	n.isRunning = true

	n.logger.Info("Successfully connected to core NATS",
		zap.String("subject", subject),
		zap.String("queue_group", queueGroup))

	return nil
}

// handleMessage handles incoming NATS messages
func (n *NATSConsumer) handleMessage(msg *nats.Msg) {
	var tx entity.Transaction
	if err := json.Unmarshal(msg.Data, &tx); err != nil {
		n.logger.Error("Failed to unmarshal transaction", zap.Error(err))
		if msg.Reply != "" {
			msg.Respond([]byte("ERROR: Failed to unmarshal"))
		}
		return
	}

	n.logger.Debug("Processing transaction",
		zap.String("hash", tx.Hash),
		zap.String("from", tx.From),
		zap.String("to", tx.To),
		zap.String("value", tx.Value))

	// Send to message channel
	select {
	case n.msgChan <- &tx:
		n.logger.Debug("Sent transaction to processing channel", zap.String("hash", tx.Hash))
		// Acknowledge if it's a JetStream message
		if msg.Reply != "" {
			msg.Ack()
		}
	default:
		// Channel is full
		n.logger.Warn("Message channel is full, dropping message", zap.String("hash", tx.Hash))
		if msg.Reply != "" {
			msg.Nak()
		}
	}
}

// Disconnect disconnects from NATS server
func (n *NATSConsumer) Disconnect() error {
	n.isRunning = false

	if n.sub != nil {
		n.sub.Unsubscribe()
		n.sub = nil
	}
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}
	close(n.msgChan)
	n.logger.Info("Disconnected from NATS JetStream")
	return nil
}

// IsConnected checks if connected to NATS
func (n *NATSConsumer) IsConnected() bool {
	return n.isRunning && n.conn != nil && n.conn.IsConnected()
}

// GetMessageChannel returns the message channel
func (n *NATSConsumer) GetMessageChannel() <-chan *entity.Transaction {
	return n.msgChan
}
