package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/IBM/sarama"

	obshealth "github.com/Dorico-Dynamics/txova-go-observability/health"
)

type fakeHealthAdmin struct {
	describeGroups []*sarama.GroupDescription
	describeErr    error
	offsets        *sarama.OffsetFetchResponse
	offsetsErr     error
	closeErr       error
}

func (f *fakeHealthAdmin) DescribeConsumerGroups(groups []string) ([]*sarama.GroupDescription, error) {
	return f.describeGroups, f.describeErr
}

func (f *fakeHealthAdmin) ListConsumerGroupOffsets(group string, topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	return f.offsets, f.offsetsErr
}

func (f *fakeHealthAdmin) Close() error {
	return f.closeErr
}

type fakeHealthClient struct {
	partitions map[string][]int32
	offsets    map[string]int64
	closeErr   error
}

func (f *fakeHealthClient) Partitions(topic string) ([]int32, error) {
	partitions, ok := f.partitions[topic]
	if !ok {
		return nil, errors.New("missing topic")
	}
	return partitions, nil
}

func (f *fakeHealthClient) GetOffset(topic string, partitionID int32, time int64) (int64, error) {
	value, ok := f.offsets[offsetKey(topic, partitionID)]
	if !ok {
		return 0, errors.New("missing offset")
	}
	return value, nil
}

func (f *fakeHealthClient) Close() error {
	return f.closeErr
}

func offsetKey(topic string, partition int32) string {
	return fmt.Sprintf("%s|%d", topic, partition)
}

func TestGroupHealthCheckerCheckHealthy(t *testing.T) {
	t.Parallel()

	checker := &GroupHealthChecker{
		name:     "ride-consumer",
		groupID:  "ride-service-group",
		topics:   []string{"ride-events"},
		required: true,
		maxLag:   20,
		logger:   slog.Default(),
		client: &fakeHealthClient{
			partitions: map[string][]int32{"ride-events": {0, 1}},
			offsets: map[string]int64{
				offsetKey("ride-events", 0): 100,
				offsetKey("ride-events", 1): 55,
			},
		},
		admin: &fakeHealthAdmin{
			describeGroups: []*sarama.GroupDescription{{
				State: "Stable",
				Members: map[string]*sarama.GroupMemberDescription{
					"member-1": {},
				},
			}},
			offsets: newOffsetFetchResponse(map[string]map[int32]int64{
				"ride-events": {
					0: 90,
					1: 50,
				},
			}),
		},
		decodeAssignment: func(member *sarama.GroupMemberDescription) (*sarama.ConsumerGroupMemberAssignment, error) {
			return &sarama.ConsumerGroupMemberAssignment{
				Topics: map[string][]int32{"ride-events": {0, 1}},
			}, nil
		},
	}

	result := checker.Check(context.Background())
	if result.Status != obshealth.StatusHealthy {
		t.Fatalf("status = %v, want healthy", result.Status)
	}
	if result.Details["total_lag"] != int64(15) {
		t.Fatalf("total_lag = %v, want 15", result.Details["total_lag"])
	}
}

func TestGroupHealthCheckerCheckDegradedOnLag(t *testing.T) {
	t.Parallel()

	checker := &GroupHealthChecker{
		name:    "payment-consumer",
		groupID: "payment-service-group",
		topics:  []string{"payment-events"},
		maxLag:  10,
		logger:  slog.Default(),
		client: &fakeHealthClient{
			partitions: map[string][]int32{"payment-events": {0}},
			offsets: map[string]int64{
				offsetKey("payment-events", 0): 100,
			},
		},
		admin: &fakeHealthAdmin{
			describeGroups: []*sarama.GroupDescription{{
				State: "Stable",
				Members: map[string]*sarama.GroupMemberDescription{
					"member-1": {},
				},
			}},
			offsets: newOffsetFetchResponse(map[string]map[int32]int64{
				"payment-events": {
					0: 50,
				},
			}),
		},
		decodeAssignment: func(member *sarama.GroupMemberDescription) (*sarama.ConsumerGroupMemberAssignment, error) {
			return &sarama.ConsumerGroupMemberAssignment{
				Topics: map[string][]int32{"payment-events": {0}},
			}, nil
		},
	}

	result := checker.Check(context.Background())
	if result.Status != obshealth.StatusDegraded {
		t.Fatalf("status = %v, want degraded", result.Status)
	}
}

func TestGroupHealthCheckerCheckUnhealthyWithoutAssignments(t *testing.T) {
	t.Parallel()

	checker := &GroupHealthChecker{
		name:    "notification-consumer",
		groupID: "notification-service-group",
		topics:  []string{"notification-events"},
		logger:  slog.Default(),
		client: &fakeHealthClient{
			partitions: map[string][]int32{"notification-events": {0}},
			offsets: map[string]int64{
				offsetKey("notification-events", 0): 10,
			},
		},
		admin: &fakeHealthAdmin{
			describeGroups: []*sarama.GroupDescription{{
				State:   "Stable",
				Members: map[string]*sarama.GroupMemberDescription{},
			}},
			offsets: newOffsetFetchResponse(map[string]map[int32]int64{
				"notification-events": {
					0: 10,
				},
			}),
		},
		decodeAssignment: func(member *sarama.GroupMemberDescription) (*sarama.ConsumerGroupMemberAssignment, error) {
			return &sarama.ConsumerGroupMemberAssignment{Topics: map[string][]int32{}}, nil
		},
	}

	result := checker.Check(context.Background())
	if result.Status != obshealth.StatusUnhealthy {
		t.Fatalf("status = %v, want unhealthy", result.Status)
	}
}

func TestGroupHealthCheckerCheckUnhealthyOnDescribeError(t *testing.T) {
	t.Parallel()

	checker := &GroupHealthChecker{
		name:    "ride-consumer",
		groupID: "ride-service-group",
		topics:  []string{"ride-events"},
		logger:  slog.Default(),
		client: &fakeHealthClient{
			partitions: map[string][]int32{"ride-events": {0}},
			offsets: map[string]int64{
				offsetKey("ride-events", 0): 10,
			},
		},
		admin: &fakeHealthAdmin{
			describeErr: errors.New("broker unavailable"),
		},
		decodeAssignment: func(member *sarama.GroupMemberDescription) (*sarama.ConsumerGroupMemberAssignment, error) {
			return &sarama.ConsumerGroupMemberAssignment{}, nil
		},
	}

	result := checker.Check(context.Background())
	if result.Status != obshealth.StatusUnhealthy {
		t.Fatalf("status = %v, want unhealthy", result.Status)
	}
}

func TestGroupHealthCheckerCheckDegradedOnNonStableState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state string
	}{
		{name: "preparing rebalance", state: "PreparingRebalance"},
		{name: "completing rebalance", state: "CompletingRebalance"},
		{name: "empty state", state: "Empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			checker := &GroupHealthChecker{
				name:    "test-consumer",
				groupID: "test-group",
				topics:  []string{"test-events"},
				logger:  slog.Default(),
				client: &fakeHealthClient{
					partitions: map[string][]int32{"test-events": {0}},
					offsets: map[string]int64{
						offsetKey("test-events", 0): 10,
					},
				},
				admin: &fakeHealthAdmin{
					describeGroups: []*sarama.GroupDescription{{
						State: tt.state,
						Members: map[string]*sarama.GroupMemberDescription{
							"member-1": {},
						},
					}},
					offsets: newOffsetFetchResponse(map[string]map[int32]int64{
						"test-events": {0: 10},
					}),
				},
				decodeAssignment: func(_ *sarama.GroupMemberDescription) (*sarama.ConsumerGroupMemberAssignment, error) {
					return &sarama.ConsumerGroupMemberAssignment{
						Topics: map[string][]int32{"test-events": {0}},
					}, nil
				},
			}

			result := checker.Check(context.Background())
			if result.Status != obshealth.StatusDegraded {
				t.Fatalf("status = %v, want degraded for state %q", result.Status, tt.state)
			}
		})
	}
}

func TestGroupHealthCheckerCloseErrors(t *testing.T) {
	t.Parallel()

	t.Run("admin close error only", func(t *testing.T) {
		t.Parallel()
		checker := &GroupHealthChecker{
			admin:  &fakeHealthAdmin{closeErr: errors.New("admin close failed")},
			client: &fakeHealthClient{},
		}
		err := checker.Close()
		if err == nil || !errors.Is(err, checker.admin.(*fakeHealthAdmin).closeErr) {
			t.Fatalf("expected admin close error, got %v", err)
		}
	})

	t.Run("client close error only", func(t *testing.T) {
		t.Parallel()
		checker := &GroupHealthChecker{
			admin:  &fakeHealthAdmin{},
			client: &fakeHealthClient{closeErr: errors.New("client close failed")},
		}
		err := checker.Close()
		if err == nil || !errors.Is(err, checker.client.(*fakeHealthClient).closeErr) {
			t.Fatalf("expected client close error, got %v", err)
		}
	})

	t.Run("both close errors joined", func(t *testing.T) {
		t.Parallel()
		adminErr := errors.New("admin close failed")
		clientErr := errors.New("client close failed")
		checker := &GroupHealthChecker{
			admin:  &fakeHealthAdmin{closeErr: adminErr},
			client: &fakeHealthClient{closeErr: clientErr},
		}
		err := checker.Close()
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, adminErr) {
			t.Fatalf("expected joined error to contain admin error, got %v", err)
		}
		if !errors.Is(err, clientErr) {
			t.Fatalf("expected joined error to contain client error, got %v", err)
		}
	})

	t.Run("nil checker returns nil", func(t *testing.T) {
		t.Parallel()
		var checker *GroupHealthChecker
		if err := checker.Close(); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}

func TestNewGroupHealthCheckerValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *GroupHealthConfig
		wantErr string
	}{
		{name: "nil config", cfg: nil, wantErr: "config is required"},
		{name: "no brokers", cfg: &GroupHealthConfig{GroupID: "g", Topics: []string{"t"}}, wantErr: "brokers"},
		{name: "no group ID", cfg: &GroupHealthConfig{Brokers: []string{"b"}, Topics: []string{"t"}}, wantErr: "group ID"},
		{name: "no topics", cfg: &GroupHealthConfig{Brokers: []string{"b"}, GroupID: "g"}, wantErr: "topics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewGroupHealthChecker(tt.cfg)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestGroupHealthCheckerCheckContextCancelled(t *testing.T) {
	t.Parallel()

	t.Run("cancelled before check", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		checker := &GroupHealthChecker{
			name:    "test-consumer",
			groupID: "test-group",
			topics:  []string{"test-events"},
			logger:  slog.Default(),
			client:  &fakeHealthClient{},
			admin:   &fakeHealthAdmin{},
		}

		result := checker.Check(ctx)
		if result.Status != obshealth.StatusUnhealthy {
			t.Fatalf("status = %v, want unhealthy", result.Status)
		}
	})

	t.Run("cancelled after describe", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())

		checker := &GroupHealthChecker{
			name:    "test-consumer",
			groupID: "test-group",
			topics:  []string{"test-events"},
			logger:  slog.Default(),
			client:  &fakeHealthClient{},
			admin: &fakeHealthAdmin{
				describeGroups: []*sarama.GroupDescription{{
					State: "Stable",
					Members: map[string]*sarama.GroupMemberDescription{
						"member-1": {},
					},
				}},
			},
			decodeAssignment: func(_ *sarama.GroupMemberDescription) (*sarama.ConsumerGroupMemberAssignment, error) {
				cancel()
				return &sarama.ConsumerGroupMemberAssignment{
					Topics: map[string][]int32{"test-events": {0}},
				}, nil
			},
		}

		result := checker.Check(ctx)
		if result.Status != obshealth.StatusUnhealthy {
			t.Fatalf("status = %v, want unhealthy", result.Status)
		}
	})
}

func newOffsetFetchResponse(offsets map[string]map[int32]int64) *sarama.OffsetFetchResponse {
	response := &sarama.OffsetFetchResponse{
		Blocks: make(map[string]map[int32]*sarama.OffsetFetchResponseBlock, len(offsets)),
	}

	for topic, partitions := range offsets {
		response.Blocks[topic] = make(map[int32]*sarama.OffsetFetchResponseBlock, len(partitions))
		for partition, offset := range partitions {
			response.Blocks[topic][partition] = &sarama.OffsetFetchResponseBlock{
				Offset: offset,
				Err:    sarama.ErrNoError,
			}
		}
	}

	return response
}
