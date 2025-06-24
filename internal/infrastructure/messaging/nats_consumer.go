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

// NATSConsumer handles NATS consumption
type NATSConsumer struct {
	conn      *nats.Conn
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

	// Subscribe directly to the subject using a queue group
	subject := fmt.Sprintf("%s.events", n.config.SubjectPrefix)
	queueGroup := n.config.ConsumerGroup

	n.logger.Info("Subscribing to subject with queue group",
		zap.String("subject", subject),
		zap.String("queue_group", queueGroup))

	sub, err := conn.QueueSubscribe(subject, queueGroup, func(msg *nats.Msg) {
		var tx entity.Transaction
		if err := json.Unmarshal(msg.Data, &tx); err != nil {
			n.logger.Error("Failed to unmarshal transaction", zap.Error(err))
			return
		}

		// Send to message channel
		select {
		case n.msgChan <- &tx:
			n.logger.Debug("Processed transaction", zap.String("hash", tx.Hash))
			msg.Ack()
		default:
			// Channel is full, skip this message
			n.logger.Warn("Message channel is full, skipping message", zap.String("hash", tx.Hash))
		}
	})

	if err != nil {
		n.logger.Error("Failed to subscribe to subject", zap.Error(err))
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	n.sub = sub
	n.isRunning = true
	n.logger.Info("Successfully connected to NATS and subscribed to subject",
		zap.String("subject", subject),
		zap.String("queue_group", queueGroup))

	return nil
}

// Disconnect disconnects from NATS server
func (n *NATSConsumer) Disconnect() error {
	if n.sub != nil {
		n.sub.Unsubscribe()
		n.sub = nil
	}
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}
	n.isRunning = false
	close(n.msgChan)
	n.logger.Info("Disconnected from NATS")
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

// We've moved the functionality directly into the Connect method
// and are using QueueSubscribe with a callback function instead
