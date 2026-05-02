package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// PriorityZSetSuffix 优先级 ZSet 后缀
	PriorityZSetSuffix = ":priority"
	// TaskCacheSuffix 任务缓存KV后缀，用于快速查询任务数据
	TaskCacheSuffix = ":task:"
)

type TaskQueue struct {
	client          *redis.ClusterClient
	streamKey       string
	group           string
	priorityZSetKey string // Redis ZSet key（优先级排序，score核心）
	cacheKey        string // 任务缓存KV key，用于快速查询任务数据
}

func NewTaskQueue(client *redis.ClusterClient, streamKey, group string) *TaskQueue {
	// HashTag 包裹，强制 Redis Cluster 同槽，事务100%安全
	base := fmt.Sprintf("{%s}", streamKey)
	return &TaskQueue{
		client:          client,
		streamKey:       base,
		group:           group,
		priorityZSetKey: base + PriorityZSetSuffix,
		cacheKey:        base + TaskCacheSuffix,
	}
}

// Init 初始化消费者组（幂等）
func (q *TaskQueue) Init(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.streamKey, q.group, "0").Err()
	if err != nil {
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			logx.Infof("consumer group[%s] already exists,skip create", q.group)
			return nil
		}
		logx.Errorf("create consumer group[%s] failed: %v", q.group, err)
		return err
	}
	logx.Infof("init task queue success,stream: %s created,consumer group[%s] created", q.streamKey, q.group)
	return nil
}

// Push 推送任务（支持score优先级，原子操作，失败回滚）
// priority: 数字越大，优先级越高
func (q *TaskQueue) Push(ctx context.Context, taskID string, data []byte, priority int) error {
	if taskID == "" || len(data) == 0 {
		return errors.New("taskID and data cannot be empty")
	}

	// 1. 组装消息
	values := map[string]interface{}{
		"task_id":  taskID,
		"data":     data,
		"priority": priority,
		"time":     time.Now().UnixMilli(),
	}

	// 2. Redis事务:保证Stream +ZSet + KV 原子性
	tx := q.client.TxPipeline()
	tx.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey,
		Values: values,
	})
	tx.ZAdd(ctx, q.priorityZSetKey, redis.Z{
		Score:  float64(priority),
		Member: taskID,
	})
	// 同时存一份 kv(用于快速查询)
	cacheKey := q.cacheKey + taskID
	tx.Set(ctx, cacheKey, data, 24*time.Hour)

	// 执行事务
	_, err := tx.Exec(ctx)
	if err != nil {
		logx.Errorf("push task %s failed: %v", taskID, err)
		return err
	}
	logx.Infof("push task[%s] success, priority: %d", taskID, priority)
	return nil
}

// Consume 消费任务，调用 handler 处理 （按score从高到低消费）
func (q *TaskQueue) Consume(ctx context.Context, consumerName string, handler func(taskID string, data []byte) error) {
	logx.Infof("consumer[%s] start, priority mode enabled", consumerName)
	for {
		select {
		case <-ctx.Done():
			logx.Infof("consumer[%s] exit gracefully", consumerName)
			return
		default:
			tasks, err := q.client.ZRevRangeWithScores(ctx, q.priorityZSetKey, 0, 10).Result()
			if err != nil || len(tasks) == 0 {
				// 无任务，短休眠避免空轮询
				time.Sleep(100 * time.Millisecond)
				continue
			}
			taskID := tasks[0].Member.(string)
			//从Stream读取该任务（消费者组读取，保证可靠消费）
			streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    q.group,
				Consumer: consumerName,
				Streams:  []string{q.streamKey, ">"},
				Count:    10,
				Block:    1 * time.Second,
			}).Result()
			if err != nil || len(streams) == 0 || len(streams[0].Messages) == 0 {
				continue
			}
			msg := streams[0].Messages[0]
			msgTaskID, ok := msg.Values["task_id"].(string)
			if !ok || msgTaskID != taskID {
				continue
			}
			data, ok := msg.Values["data"].([]byte)
			if !ok {
				logx.Errorf("task[%s] data type error", taskID)
				q.ackAndRemove(ctx, msg.ID, taskID)
				continue
			}
			// 执行业务处理
			if err := handler(taskID, data); err == nil {
				// 处理成功：确认消息 + 移除优先级队列
				q.ackAndRemove(ctx, msg.ID, taskID)
				logx.Infof("process task[%s] success", taskID)
			} else {
				// 处理失败：不ACK，保留在Pending队列，等待重试/死信认领
				logx.Errorf("process task[%s] failed: %v", taskID, err)
			}
		}
	}
}

// ClaimPending 认领超时未处理的消息（用于死信恢复）
func (q *TaskQueue) ClaimPending(ctx context.Context, consumerName string, minIdle time.Duration, handler func(taskID string, data []byte) error) {
	logx.Infof("pending claim consumer[%s] start", consumerName)
	for {
		select {
		case <-ctx.Done():
			logx.Infof("pending claim consumer[%s] exit", consumerName)
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
					claimedMsgs, err := q.client.XClaim(ctx, &redis.XClaimArgs{
						Stream:   q.streamKey,
						Group:    q.group,
						Consumer: consumerName,
						MinIdle:  minIdle,
						Messages: []string{p.ID},
					}).Result()
					if err != nil || len(claimedMsgs) == 0 {
						continue
					}
					// 处理认领的消息
					for _, msg := range claimedMsgs {
						taskID, ok := msg.Values["task_id"].(string)
						if !ok {
							q.client.XAck(ctx, q.streamKey, q.group, msg.ID)
							continue
						}
						data, ok := msg.Values["data"].([]byte)
						if !ok {
							q.ackAndRemove(ctx, msg.ID, taskID)
							continue
						}

						if err := handler(taskID, data); err == nil {
							q.ackAndRemove(ctx, msg.ID, taskID)
							logx.Infof("claim and process task[%s] success", taskID)
						}
					}
				}
			}
		}
	}
}

// ackAndRemove 原子操作：ACK消息 + 移除优先级队列 + 删除缓存
func (q *TaskQueue) ackAndRemove(ctx context.Context, msgID string, taskID string) {
	tx := q.client.TxPipeline()
	tx.XAck(ctx, q.streamKey, q.group, msgID)
	tx.ZRem(ctx, q.priorityZSetKey, taskID)
	// 同时删除缓存
	cacheKey := q.cacheKey + taskID
	tx.Del(ctx, cacheKey)
	// 执行事务
	_, err := tx.Exec(ctx)
	if err != nil {
		logx.Errorf("ack and remove task %s failed: %v", taskID, err)
		return
	}
}

// GetTask 查询任务数据（从缓存）
func (q *TaskQueue) GetTask(ctx context.Context, taskID string) ([]byte, error) {
	cacheKey := q.cacheKey + taskID
	return q.client.Get(ctx, cacheKey).Bytes()
}

// DeleteTask 手动删除任务
func (q *TaskQueue) DeleteTask(ctx context.Context, msgID, taskID string) {
	q.ackAndRemove(ctx, msgID, taskID)
}
