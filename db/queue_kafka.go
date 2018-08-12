package db

/*
import (
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

func DefaultConfig() *sarama.Config {
	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message.
	cfg.Producer.Retry.Max = 10                   // Retry up to 10 times to produce the message.
	return cfg
}

// PartitionConsumer creates a new kafka partition consumer.
//
// Invoker is responsible for closing the resource. Failing to do so will
// result in resource leakage.
func PartitionConsumer(client sarama.Client, topic string, partition int32, offset int64) (sarama.PartitionConsumer, error) {
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("kafka.PartitionConsumer consumer: %s", err)
	}
	pc, err := consumer.ConsumePartition(topic, partition, offset)
	if err != nil {
		return nil, fmt.Errorf("kafka.PartitionConsumer pc: %s", err)
	}
	return pc, nil
}

func BrokerConfig() *sarama.Config {
	cfg := DefaultConfig()
	cfg.Net.MaxOpenRequests = 5
	cfg.Net.DialTimeout = 10 * time.Second
	cfg.Net.ReadTimeout = 10 * time.Second
	cfg.Net.WriteTimeout = 10 * time.Second
	return cfg
}

// FindLeader queries one or more live brokers to find out leader information
// and replica information for a given topic and partition.
func FindLeader(seedBrokers []string, topic string, partitionId int32) (partition *sarama.PartitionMetadata, leaderAddr string, err error) {
	brokerConfig := BrokerConfig()
	for _, seedBroker := range seedBrokers {
		broker := sarama.NewBroker(seedBroker)
		if err = broker.Open(brokerConfig); err != nil {
			log.Infof("FindLeader: failed to open connection to seed broker `%s': %s", seedBroker, err)
			continue
		}
		defer func(seedBroker string) {
			if err := broker.Close(); err != nil {
				log.Infof("FindLeader: non-fatal error closing connection to seed broker `%s': %s", seedBroker, err)
			}
		}(seedBroker)
		request := &sarama.MetadataRequest{Topics: []string{topic}}
		var response *sarama.MetadataResponse
		if response, err = broker.GetMetadata(request); err != nil {
			log.Infof("FindLeader: failed to get metadata from seed broker `%s': %s", seedBroker, err)
			continue
		}
		for _, topicMetaData := range response.Topics {
			for _, part := range topicMetaData.Partitions {
				if part.ID == partitionId {
					partition = part
					// Lookup broker leader address.
					for _, brokerInfo := range response.Brokers {
						if brokerInfo.ID() == partition.Leader {
							leaderAddr = brokerInfo.Addr()
						}
					}
					break
				}
			}
		}
	}
	if err != nil {
		return
	}
	if leaderAddr == "" {
		err = fmt.Errorf("failed to lookup leader address for partition.Leader=%v", partition.Leader)
		return
	}
	return
}
*/
