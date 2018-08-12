package db

/*import (
	"testing"

	"gigawatt-server/config"

	"github.com/Shopify/sarama"
)

func setupTestConnection(t *testing.T) (conn sarama.Client, closer func()) {
	var err error
	if conn, err = sarama.NewClient(config.KafkaAddresses, DefaultConfig()); err != nil {
		t.Fatalf("Failed to connect to kafka addresses=%v: %s", config.KafkaAddresses, err)
	}
	closer = func() {
		if err := conn.Close(); err != nil {
			t.Logf("non-fatal error closing connection: %s\n", err)
		}
	}
	return
}
func testProducer(client sarama.Client, t *testing.T) (producer sarama.SyncProducer, closer func()) {
	var err error
	if producer, err = sarama.NewSyncProducer(config.KafkaAddresses, DefaultConfig()); err != nil {
		t.Fatalf("Failed to create sync producer: %s", err)
	}
	closer = func() {
		if err := producer.Close(); err != nil {
			t.Logf("non-fatal error closing producer: %s\n", err)
		}
	}
	return
}

func Test_Connect(t *testing.T) {
	if len(config.KafkaAddresses) == 0 {
		t.Skip("config.KafkaAddresses was empty")
	}
	_, closer := setupTestConnection(t)
	defer closer()
}

func Test_FindLeader(t *testing.T) {
	if len(config.KafkaAddresses) == 0 {
		t.Skip("config.KafkaAddresses was empty")
	}
	client, clientCloser := setupTestConnection(t)
	defer clientCloser()
	producer, producerCloser := testProducer(client, t)
	defer producerCloser()
	// Use producer to send one or more messages.
	topic := "test_topic"
	m := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder("the key"),
		Value: sarama.StringEncoder("testing"),
	}
	partitionId, offset, err := producer.SendMessage(m)
	if err != nil {
		t.Fatalf("Error using producer to send message: %s", err)
	}
	t.Logf("sending message yielded partitionId=%v offset=%v", partitionId, offset)
	// Find the leader.
	pinfo, leaderAddr, err := FindLeader(config.KafkaAddresses, topic, partitionId)
	if err != nil {
		t.Fatalf("Failed to get partition metadata: %s", err)
	}
	t.Logf("pinfo=%+v leaderAddr=%s", pinfo, leaderAddr)
}*/
