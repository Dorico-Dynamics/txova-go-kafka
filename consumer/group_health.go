package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/IBM/sarama"

	obshealth "github.com/Dorico-Dynamics/txova-go-observability/health"
)

// GroupHealthConfig configures the consumer group health checker.
type GroupHealthConfig struct {
	Name     string
	Brokers  []string
	GroupID  string
	Topics   []string
	Required bool
	MaxLag   int64
	Logger   *slog.Logger
}

type healthAdmin interface {
	DescribeConsumerGroups(groups []string) ([]*sarama.GroupDescription, error)
	ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error)
	Close() error
}

type healthClient interface {
	Partitions(topic string) ([]int32, error)
	GetOffset(topic string, partitionID int32, time int64) (int64, error)
	Close() error
}

// GroupHealthChecker checks consumer group membership, partition assignment, and lag.
type GroupHealthChecker struct {
	name             string
	groupID          string
	topics           []string
	required         bool
	maxLag           int64
	logger           *slog.Logger
	client           healthClient
	admin            healthAdmin
	decodeAssignment func(member *sarama.GroupMemberDescription) (*sarama.ConsumerGroupMemberAssignment, error)
}

var _ obshealth.Checker = (*GroupHealthChecker)(nil)

// NewGroupHealthChecker creates a health checker for a Kafka consumer group.
func NewGroupHealthChecker(cfg *GroupHealthConfig) (*GroupHealthChecker, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if len(cfg.Brokers) == 0 {
		return nil, ErrNoBrokers
	}
	if cfg.GroupID == "" {
		return nil, ErrNoGroupID
	}
	if len(cfg.Topics) == 0 {
		return nil, ErrNoTopics
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	client, err := sarama.NewClient(cfg.Brokers, DefaultConfig().toSaramaConfig())
	if err != nil {
		return nil, fmt.Errorf("creating kafka client: %w", err)
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("creating kafka admin client: %w", err)
	}

	name := cfg.Name
	if name == "" {
		name = cfg.GroupID
	}

	return &GroupHealthChecker{
		name:     name,
		groupID:  cfg.GroupID,
		topics:   append([]string(nil), cfg.Topics...),
		required: cfg.Required,
		maxLag:   cfg.MaxLag,
		logger:   logger,
		client:   client,
		admin:    admin,
		decodeAssignment: func(member *sarama.GroupMemberDescription) (*sarama.ConsumerGroupMemberAssignment, error) {
			return member.GetMemberAssignment()
		},
	}, nil
}

// Name returns the checker name.
func (c *GroupHealthChecker) Name() string {
	return c.name
}

// Required returns whether this health check is required for service readiness.
func (c *GroupHealthChecker) Required() bool {
	return c.required
}

// Close releases the Kafka resources used by the checker.
func (c *GroupHealthChecker) Close() error {
	if c == nil {
		return nil
	}

	var adminErr error
	if c.admin != nil {
		if err := c.admin.Close(); err != nil {
			adminErr = fmt.Errorf("closing kafka admin client: %w", err)
		}
	}

	var clientErr error
	if c.client != nil {
		if err := c.client.Close(); err != nil {
			clientErr = fmt.Errorf("closing kafka client: %w", err)
		}
	}

	return errors.Join(adminErr, clientErr)
}

// Check evaluates the health of the consumer group.
func (c *GroupHealthChecker) Check(ctx context.Context) obshealth.Result {
	start := time.Now()

	if err := ctx.Err(); err != nil {
		return obshealth.NewUnhealthyResult(time.Since(start), err)
	}
	if c.client == nil || c.admin == nil {
		return obshealth.NewUnhealthyResult(time.Since(start), fmt.Errorf("checker is not initialized"))
	}

	description, err := c.describeGroup()
	if err != nil {
		c.logger.Warn("consumer group describe failed", "group_id", c.groupID, "error", err.Error())
		return obshealth.NewUnhealthyResult(time.Since(start), err)
	}
	if err = ctx.Err(); err != nil {
		return obshealth.NewUnhealthyResult(time.Since(start), err)
	}

	assignedPartitions, err := c.countAssignedPartitions(description)
	if err != nil {
		c.logger.Warn("consumer group assignment decode failed", "group_id", c.groupID, "error", err.Error())
		return obshealth.NewUnhealthyResult(time.Since(start), err)
	}
	if err = ctx.Err(); err != nil {
		return obshealth.NewUnhealthyResult(time.Since(start), err)
	}

	lagSummary, err := c.calculateLag()
	if err != nil {
		c.logger.Warn("consumer group lag check failed", "group_id", c.groupID, "error", err.Error())
		return obshealth.NewUnhealthyResult(time.Since(start), err)
	}

	details := map[string]any{
		"group_id":            c.groupID,
		"group_state":         description.State,
		"members":             len(description.Members),
		"assigned_partitions": assignedPartitions,
		"topics":              c.sortedTopics(),
		"total_lag":           lagSummary.totalLag,
		"max_partition_lag":   lagSummary.maxPartitionLag,
		"lag_by_partition":    lagSummary.byPartition,
	}

	if len(description.Members) == 0 || assignedPartitions == 0 {
		return obshealth.NewUnhealthyResult(time.Since(start), fmt.Errorf("consumer group has no active partition assignments")).WithDetails(details)
	}

	if description.State != "Stable" {
		return obshealth.NewDegradedResult(time.Since(start), fmt.Sprintf("consumer group state is %s", description.State)).WithDetails(details)
	}

	if c.maxLag > 0 && lagSummary.totalLag > c.maxLag {
		return obshealth.NewDegradedResult(time.Since(start), fmt.Sprintf("consumer group lag %d exceeds threshold %d", lagSummary.totalLag, c.maxLag)).WithDetails(details)
	}

	return obshealth.NewHealthyResult(time.Since(start)).WithDetails(details)
}

func (c *GroupHealthChecker) describeGroup() (*sarama.GroupDescription, error) {
	groups, err := c.admin.DescribeConsumerGroups([]string{c.groupID})
	if err != nil {
		return nil, fmt.Errorf("describing consumer group %s: %w", c.groupID, err)
	}
	if len(groups) == 0 || groups[0] == nil {
		return nil, fmt.Errorf("consumer group %s not found", c.groupID)
	}

	group := groups[0]
	if group.Err != sarama.ErrNoError {
		return nil, fmt.Errorf("describing consumer group %s: %w", c.groupID, group.Err)
	}

	return group, nil
}

func (c *GroupHealthChecker) countAssignedPartitions(group *sarama.GroupDescription) (int, error) {
	total := 0

	for _, member := range group.Members {
		assignment, err := c.decodeAssignment(member)
		if err != nil {
			return 0, fmt.Errorf("decoding member assignment: %w", err)
		}
		for _, partitions := range assignment.Topics {
			total += len(partitions)
		}
	}

	return total, nil
}

type lagSummary struct {
	totalLag        int64
	maxPartitionLag int64
	byPartition     map[string]int64
}

func (c *GroupHealthChecker) calculateLag() (lagSummary, error) {
	topicPartitions := make(map[string][]int32, len(c.topics))
	for _, topic := range c.topics {
		partitions, err := c.client.Partitions(topic)
		if err != nil {
			return lagSummary{}, fmt.Errorf("listing partitions for topic %s: %w", topic, err)
		}
		topicPartitions[topic] = partitions
	}

	offsets, err := c.admin.ListConsumerGroupOffsets(c.groupID, topicPartitions)
	if err != nil {
		return lagSummary{}, fmt.Errorf("listing consumer group offsets: %w", err)
	}
	if offsets.Err != sarama.ErrNoError {
		return lagSummary{}, fmt.Errorf("listing consumer group offsets: %w", offsets.Err)
	}

	summary := lagSummary{byPartition: make(map[string]int64)}

	for topic, partitions := range topicPartitions {
		for _, partition := range partitions {
			latestOffset, err := c.client.GetOffset(topic, partition, sarama.OffsetNewest)
			if err != nil {
				return lagSummary{}, fmt.Errorf("getting latest offset for %s[%d]: %w", topic, partition, err)
			}

			block := offsets.GetBlock(topic, partition)
			if block == nil {
				return lagSummary{}, fmt.Errorf("missing committed offset for %s[%d]", topic, partition)
			}
			if block.Err != sarama.ErrNoError {
				return lagSummary{}, fmt.Errorf("getting committed offset for %s[%d]: %w", topic, partition, block.Err)
			}

			lag := latestOffset
			if block.Offset >= 0 {
				lag = latestOffset - block.Offset
			}
			if lag < 0 {
				lag = 0
			}

			key := fmt.Sprintf("%s[%d]", topic, partition)
			summary.byPartition[key] = lag
			summary.totalLag += lag
			if lag > summary.maxPartitionLag {
				summary.maxPartitionLag = lag
			}
		}
	}

	return summary, nil
}

func (c *GroupHealthChecker) sortedTopics() []string {
	topics := append([]string(nil), c.topics...)
	sort.Strings(topics)
	return topics
}
