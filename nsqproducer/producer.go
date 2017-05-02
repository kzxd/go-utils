package nsqproducer

import (
	"encoding/json"
	"time"

	"log"

	"github.com/nsqio/go-nsq"
	"gopkg.in/errgo.v1"
)

type Producer interface {
	Publish(topic string, message NsqMessageSerialize) error
	DeferredPublish(topic string, delay int64, message NsqMessageSerialize) error
}

type NsqProducer struct {
	producer *nsq.Producer
	config   *nsq.Config
}

type ProducerOpts struct {
	Host      string
	Port      string
	NsqConfig *nsq.Config
}

type NsqMessageSerialize struct {
	At      int64       `json:"at"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func New(opts ProducerOpts) *NsqProducer {
	client, err := nsq.NewProducer(opts.Host+":"+opts.Port, opts.NsqConfig)
	if err != nil {
		log.Fatalf("init-nsq: cannot initialize nsq producer: %v:%v", opts.Host, opts.Port)
	}
	return &NsqProducer{producer: client, config: opts.NsqConfig}
}

func (p *NsqProducer) Stop() {
	p.producer.Stop()
}

func (p *NsqProducer) Publish(topic string, message NsqMessageSerialize) error {
	body, err := json.Marshal(message)
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}

	err = p.producer.Publish(topic, body)
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}
	return nil
}

func (p *NsqProducer) DeferredPublish(topic string, delay int64, message NsqMessageSerialize) error {
	body, err := json.Marshal(message)
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}

	err = p.producer.DeferredPublish(topic, time.Duration(delay)*time.Second, body)
	if err != nil {
		return errgo.Mask(err, errgo.Any)
	}

	return nil

}
