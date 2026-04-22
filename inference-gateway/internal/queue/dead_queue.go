package queue

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
)

const DeadLetterStreamSuffix = "_dead"

type DeadLetterQueue struct {
	client    *redis.Client
	streamKey string
	group     string
}

func NewDeadLetterQueue(client *redis.Client, baseStream string, group string) *DeadLetterQueue {
	return &DeadLetterQueue{client, baseStream + DeadLetterStreamSuffix, group}
}

// Init 初始化消费者组（幂等）
func (q *DeadLetterQueue) Init(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.streamKey, q.group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

// Push 将失败任务放入死信队列
func (d *DeadLetterQueue) Push(ctx context.Context, taskID string, data []byte, reason string) error {
	return d.client.XAdd(ctx, &redis.XAddArgs{
		Stream: d.streamKey,
		Values: map[string]interface{}{
			"task_id": taskID,
			"data":    string(data),
			"reason":  reason,
			"time":    time.Now().Unix(),
		},
	}).Err()
}

// Pop 消费死信队列任务，调用 handler 处理
func (d *DeadLetterQueue) Pop(ctx context.Context, consumerName string, handler func(taskID string, data []byte) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 读取新消息（> 表示未分配给该消费者的消息）
			streams, err := d.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    d.group,
				Consumer: consumerName,
				Streams:  []string{d.streamKey, ">"},
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
					d.client.XAck(ctx, d.streamKey, d.group, msg.ID)
				} else {
					logx.Errorf("handle task %s failed: %v", taskID, err)
					// 处理失败不 ACK，消息会留在 pending 列表，后续可重试
				}
			}
		}
	}
}
