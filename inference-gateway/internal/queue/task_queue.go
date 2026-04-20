package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type TaskQueue struct {
	client    *redis.Client
	streamKey string
	group     string
}

func NewTaskQueue(client *redis.Client, streamKey, group string) *TaskQueue {
	return &TaskQueue{
		client:    client,
		streamKey: streamKey,
		group:     group,
	}
}

// Init 初始化消费者组（幂等）
func (q *TaskQueue) Init(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.streamKey, q.group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

// Push 发布任务到队列，支持优先级（通过 score 实现）
func (q *TaskQueue) Push(ctx context.Context, taskID string, data []byte, priority int) error {
	// 1. 推送到 Stream (用于消费)
	err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey,
		Values: map[string]interface{}{
			"task_id":  taskID,
			"data":     data,
			"priority": priority,
			"time":     time.Now().Unix(),
		},
	}).Err()
	if err != nil {
		return err
	}

	// 2. 同时存一份 kv(用于快速查询)
	key := fmt.Sprintf("%s:%s", q.streamKey, taskID)
	q.client.Set(ctx, key, string(data), 0)
	return nil
}

// Consume 消费任务，调用 handler 处理
func (q *TaskQueue) Consume(ctx context.Context, consumerName string, handler func(taskID string, data []byte) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 读取新消息（> 表示未分配给该消费者的消息）
			streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    q.group,
				Consumer: consumerName,
				Streams:  []string{q.streamKey, ">"},
				Count:    1,
				Block:    5 * time.Second,
			}).Result()
			if err != nil || len(streams) == 0 {
				continue
			}
			for _, msg := range streams[0].Messages {
				taskID := msg.Values["task_id"].(string)
				data := []byte(msg.Values["data"].(string))
				if err := handler(taskID, data); err == nil {
					// 处理成功，确认消息
					q.client.XAck(ctx, q.streamKey, q.group, msg.ID)
				} else {
					logx.Errorf("handle task %s failed: %v", taskID, err)
					// 处理失败不 ACK，消息会留在 pending 列表，后续可重试
				}
			}
		}
	}
}

// ClaimPending 认领超时未处理的消息（用于死信恢复）
func (q *TaskQueue) ClaimPending(ctx context.Context, consumerName string, minIdle time.Duration, handler func(taskID string, data []byte) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 获取 pending 消息
			pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{
				Stream: q.streamKey,
				Group:  q.group,
				Start:  "-",
				End:    "+",
				Count:  10,
			}).Result()
			if err != nil || len(pending) == 0 {
				time.Sleep(5 * time.Second)
				continue
			}
			for _, p := range pending {
				if p.Idle >= minIdle {
					// 认领消息
					claimed, err := q.client.XClaim(ctx, &redis.XClaimArgs{
						Stream:   q.streamKey,
						Group:    q.group,
						Consumer: consumerName,
						MinIdle:  minIdle,
						Messages: []string{p.ID},
					}).Result()
					if err != nil {
						continue
					}
					for _, msg := range claimed {
						taskID := msg.Values["task_id"].(string)
						data := []byte(msg.Values["data"].(string))
						if err := handler(taskID, data); err == nil {
							q.client.XAck(ctx, q.streamKey, q.group, msg.ID)
						}
					}
				}
			}
		}
	}
}
