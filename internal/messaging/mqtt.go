package messaging

import (
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type subscription struct {
	topic   string
	qos     byte
	handler mqtt.MessageHandler
}

type Client struct {
	client        mqtt.Client
	clientID      string
	mu            sync.RWMutex
	subscriptions []subscription
}

func NewClient(brokerAddr, clientID string) *Client {
	c := &Client{clientID: clientID}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerAddr)
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(3 * time.Second)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.OnConnect = c.onConnect
	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		fmt.Printf("Conexão MQTT perdida (client=%s): %v\n", clientID, err)
	}

	c.client = mqtt.NewClient(opts)
	return c
}

func (c *Client) Connect() {
	token := c.client.Connect()
	if !token.WaitTimeout(3 * time.Second) {
		fmt.Printf("MQTT indisponível no momento (client=%s): timeout de conexão\n", c.clientID)
		return
	}

	if token.Error() != nil {
		fmt.Printf("MQTT indisponível no momento (client=%s): %v\n", c.clientID, token.Error())
	}
}

func (c *Client) Subscribe(topic string, qos byte, handler mqtt.MessageHandler) {
	c.mu.Lock()
	c.subscriptions = append(c.subscriptions, subscription{
		topic:   topic,
		qos:     qos,
		handler: handler,
	})
	c.mu.Unlock()

	if c.client.IsConnected() {
		c.subscribeNow(topic, qos, handler)
	}
}

func (c *Client) Publish(topic string, qos byte, retained bool, payload []byte) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("cliente MQTT desconectado")
	}

	token := c.client.Publish(topic, qos, retained, payload)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}

func (c *Client) onConnect(_ mqtt.Client) {
	fmt.Printf("Conectado ao broker MQTT (client=%s)\n", c.clientID)

	c.mu.RLock()
	subs := make([]subscription, len(c.subscriptions))
	copy(subs, c.subscriptions)
	c.mu.RUnlock()

	for _, sub := range subs {
		c.subscribeNow(sub.topic, sub.qos, sub.handler)
	}
}

func (c *Client) subscribeNow(topic string, qos byte, handler mqtt.MessageHandler) {
	token := c.client.Subscribe(topic, qos, handler)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Falha ao assinar tópico MQTT (%s): %v\n", topic, token.Error())
	}
}
