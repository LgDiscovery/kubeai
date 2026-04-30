package queue

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"time"
)

const DeadLetterStreamSuffix = ":dead"

type DeadLetterQueue struct {
	client    *redis.ClusterClient
	streamKey string
	group     string
}

func NewDeadLetterQueue(client *redis.ClusterClient, baseStream string, group string) *DeadLetterQueue {
	tag := fmt.Sprintf("{%s}", baseStream)
	return &DeadLetterQueue{client, tag + DeadLetterStreamSuffix, group}
}

// Init 初始化消费者组（幂等）
func (q *DeadLetterQueue) Init(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.streamKey, q.group, "0").Err()
	if err != nil {
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			logx.Infof("consumer group[%s] already exists,skip create", q.group)
			return nil
		}
		logx.Errorf("create consumer group[%s] failed: %v", q.group, err)
		return err
	}
	logx.Infof("init dead letter queue success,stream: %s created,consumer group[%s] created", q.streamKey, q.group)
	return nil
}

// Push 推送失败任务到死信队列
// taskID: 任务唯一标识
// data: 任务原始数据
// reason: 进入死信的原因（如：重试3次失败、业务异常）
func (d *DeadLetterQueue) Push(ctx context.Context, taskID string,
	data []byte, reason string, taskType string) error {
	// 参数合法性校验
	if taskID == "" {
		return errors.New("taskID cannot be empty")
	}
	if taskType == "" {
		return errors.New("taskType cannot be empty")
	}
	if len(data) == 0 {
		return errors.New("task data cannot be empty")
	}
	if reason == "" {
		reason = "unknown error"
	}
	// 写入死信队列
	err := d.client.XAdd(ctx, &redis.XAddArgs{
		Stream: d.streamKey,
		Values: map[string]interface{}{
			"task_id":   taskID,
			"data":      data, // 直接存[]byte，避免编码问题
			"reason":    reason,
			"task_type": taskType,
			"time":      time.Now().UnixMilli(), // 毫秒级时间戳，更精准
		},
	}).Err()

	if err != nil {
		logx.Errorf("failed to push dead letter task[%s], reason: %s, err: %v", taskID, reason, err)
		return err
	}

	logx.Infof("dead letter task[%s] pushed successfully", taskID)
	return nil
}

// Pop 消费死信队列（阻塞式消费，支持优雅退出）
func (d *DeadLetterQueue) Pop(ctx context.Context, consumerName string, handler func(taskID string, data []byte, taskType string) error) {
	logx.Infof("dead letter consumer[%s] started, stream: %s", consumerName, d.streamKey)
	for {
		select {
		case <-ctx.Done():
			logx.Infof("dead letter consumer[%s] exited gracefully", consumerName)
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
			if err != nil || len(streams) == 0 || len(streams[0].Messages) == 0 {
				continue
			}
			for _, msg := range streams[0].Messages {
				taskID, ok := d.safeGetString(msg.Values, "task_id")
				if !ok {
					logx.Errorf("invalid dead letter message, id: %s, missing task_id", msg.ID)
					d.ackMessage(ctx, msg.ID)
					continue
				}
				data, ok := d.safeGetBytes(msg.Values, "data")
				if !ok {
					logx.Errorf("invalid dead letter message, id: %s, missing data", msg.ID)
					d.ackMessage(ctx, msg.ID)
					continue
				}
				taskType, ok := d.safeGetString(msg.Values, "task_type")
				if !ok {
					logx.Errorf("invalid dead letter message, id: %s, missing task_type", msg.ID)
					d.ackMessage(ctx, msg.ID)
					continue
				}
				reason, _ := d.safeGetString(msg.Values, "reason")
				logx.Infof("processing dead letter task[%s], reason: %s", taskID, reason)

				if err := handler(taskID, data, taskType); err == nil {
					// 处理成功：确认消息，永久删除
					d.ackMessage(ctx, msg.ID)
					logx.Infof("dead letter task[%s] processed successfully", taskID)
				} else {
					logx.Errorf("handle task %s failed: %v", taskID, err)
					// 处理失败不 ACK，消息会留在 pending 列表，后续可重试
				}
			}
		}
	}
}

// ackMessage 确认消息（封装XAck，统一日志）
func (d *DeadLetterQueue) ackMessage(ctx context.Context, msgID string) {
	if err := d.client.XAck(ctx, d.streamKey, d.group, msgID).Err(); err != nil {
		logx.Errorf("failed to ack message[%s], err: %v", msgID, err)
	}
}

// safeGetString 安全获取string类型字段，避免panic
func (d *DeadLetterQueue) safeGetString(values map[string]interface{}, key string) (string, bool) {
	val, ok := values[key]
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// safeGetBytes 安全获取[]byte类型字段，适配Redis存储格式
func (d *DeadLetterQueue) safeGetBytes(values map[string]interface{}, key string) ([]byte, bool) {
	val, ok := values[key]
	if !ok {
		return nil, false
	}
	switch v := val.(type) {
	case []byte:
		return v, true
	case string:
		return []byte(v), true
	default:
		return nil, false
	}
}
