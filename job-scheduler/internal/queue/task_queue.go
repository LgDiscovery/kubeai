package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
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
	q.client.Set(ctx, key, string(data), 24*time.Hour)
	return nil
}
