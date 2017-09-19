package nsqproducer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Scalingo/go-internal-tools/logger"
	"github.com/nsqio/go-nsq"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/errgo.v1"
)

type Producer interface {
	Publish(ctx context.Context, topic string, message NsqMessageSerialize) error
	DeferredPublish(ctx context.Context, topic string, delay int64, message NsqMessageSerialize) error
}

type NsqProducer struct {
	producer   *nsq.Producer
	config     *nsq.Config
	skipLogSet map[string]bool
}

type ProducerOpts struct {
	Host       string
	Port       string
	NsqConfig  *nsq.Config
	SkipLogSet map[string]bool
}

type WithLoggableFields interface {
	LoggableFields() logrus.Fields
}

type NsqMessageSerialize struct {
	At      int64       `json:"at"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`

	// Automatically set by context if existing, generated otherwise
	RequestID string `json:"request_id"`
}

func New(opts ProducerOpts) (*NsqProducer, error) {
	client, err := nsq.NewProducer(opts.Host+":"+opts.Port, opts.NsqConfig)
	if err != nil {
		return nil, fmt.Errorf("init-nsq: cannot initialize nsq producer: %v:%v", opts.Host, opts.Port)
	}
	if opts.SkipLogSet == nil {
		opts.SkipLogSet = map[string]bool{}
	}
	return &NsqProducer{producer: client, config: opts.NsqConfig, skipLogSet: opts.SkipLogSet}, nil
}

func (p *NsqProducer) Stop() {
	p.producer.Stop()
}

func (p *NsqProducer) Publish(ctx context.Context, topic string, message NsqMessageSerialize) error {
	message.RequestID = p.requestID(ctx)

	body, err := json.Marshal(message)
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}

	err = p.producer.Publish(topic, body)
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}

	p.log(ctx, message, logrus.Fields{})

	return nil
}

func (p *NsqProducer) DeferredPublish(ctx context.Context, topic string, delay int64, message NsqMessageSerialize) error {
	message.RequestID = p.requestID(ctx)

	body, err := json.Marshal(message)
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}

	err = p.producer.DeferredPublish(topic, time.Duration(delay)*time.Second, body)
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}

	p.log(ctx, message, logrus.Fields{"message_delay": delay})

	return nil
}

func (p *NsqProducer) requestID(ctx context.Context) string {
	reqid, ok := ctx.Value("request_id").(string)
	if !ok {
		return uuid.NewV4().String()
	}
	return reqid
}

func (p *NsqProducer) logger(ctx context.Context) logrus.FieldLogger {
	return logger.Get(ctx)
}

func (p *NsqProducer) log(ctx context.Context, message NsqMessageSerialize, fields logrus.Fields) {
	if p.skipLogSet[message.Type] {
		return
	}

	logger := p.logger(ctx).WithFields(fields)

	if logger.Level == logrus.DebugLevel {
		logger.WithFields(logrus.Fields{"message_type": message.Type, "message_payload": message.Payload}).Debug("publish message")
	} else {
		// We don't want the complete payload to be dump in the logs With this
		// interface we can, for each type of payload, add fields in the logs.
		if payload, ok := message.Payload.(WithLoggableFields); ok {
			logger = logger.WithFields(payload.LoggableFields())
		}
		logger.WithFields(logrus.Fields{"message_type": message.Type}).Info("publish message")
	}
}
