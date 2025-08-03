package sharding

import (
	"context"
	"errors"
	"fmt"
	"github.com/ecodeclub/ekit/syncx"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"notification-platform/internal/domain"
	idgen "notification-platform/internal/pkg/idgenerator"
	"notification-platform/internal/pkg/sharding"
	"notification-platform/internal/repository/dao"
	"time"
)

var _ dao.TxNotificationDAO = (*TxNShardingDAO)(nil)

type TxNShardingDAO struct {
	dbs                 *syncx.Map[string, *egorm.Component]
	nShardingStrategy   sharding.ShardingStrategy
	txnShardingStrategy sharding.ShardingStrategy
	idGen               idgen.Generator
}

func (t *TxNShardingDAO) Create(ctx context.Context, notification dao.TxNotification) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TxNShardingDAO) FindCheckBack(_ context.Context, _, _ int) ([]dao.TxNotification, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TxNShardingDAO) UpdateCheckStatus(_ context.Context, _ []dao.TxNotification, _ domain.SendStatus) error {
	// TODO implement me
	panic("implement me")
}

func (t *TxNShardingDAO) First(_ context.Context, _ int64) (dao.TxNotification, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TxNShardingDAO) BatchGetTxNotification(_ context.Context, _ []int64) (map[int64]dao.TxNotification, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TxNShardingDAO) GetByBizIDKey(_ context.Context, _ int64, _ string) (dao.TxNotification, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TxNShardingDAO) UpdateNotificationID(_ context.Context, _ int64, _ string, _ uint64) error {
	// TODO implement me
	panic("implement me")
}

func (t *TxNShardingDAO) Prepare(ctx context.Context, txn dao.TxNotification, notification dao.Notification) (uint64, error) {
	nowTime := time.Now()
	now := nowTime.UnixMilli()
	txn.Ctime = now
	txn.Utime = now
	notification.Ctime = now
	notification.Utime = now
	// 获取db
	txndst := t.txnShardingStrategy.Shard(txn.BizID, txn.Key)
	notificationDst := t.nShardingStrategy.Shard(notification.BizID, notification.Key)
	gormDB, ok := t.dbs.Load(txndst.DB)
	if !ok {
		return 0, fmt.Errorf("未知库名 %s", notificationDst.DB)
	}

	err := gormDB.Transaction(func(tx *gorm.DB) error {
		for {
			notification.ID = uint64(t.idGen.GenerateID(notification.BizID, notification.Key))
			res := tx.WithContext(ctx).
				Table(notificationDst.Table).
				Create(&notification)
			if res.Error != nil {
				if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
					// 唯一键冲突直接返回
					if !dao.CheckErrIsIDDuplicate(notification.ID, res.Error) {
						return nil
					}
					// 主键冲突 重试再找个主键
					continue
				}
				return res.Error
			}
			if res.RowsAffected == 0 {
				return nil
			}
			txn.NotificationID = notification.ID
			return tx.WithContext(ctx).
				Table(txndst.Table).
				Clauses(clause.OnConflict{
					DoNothing: true,
				}).Create(&txn).Error
		}
	})
	return notification.ID, err
}

func (t *TxNShardingDAO) UpdateStatus(ctx context.Context, bizID int64, key string, status domain.TxNotificationStatus, notificationStatus domain.SendStatus) error {
	// 获取db
	now := time.Now().UnixMilli()
	txndst := t.txnShardingStrategy.Shard(bizID, key)
	notificationDst := t.nShardingStrategy.Shard(bizID, key)
	gormDB, ok := t.dbs.Load(txndst.DB)
	if !ok {
		return fmt.Errorf("未知库名 %s", txndst.DB)
	}

	return gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.WithContext(ctx).
			Table(txndst.Table).
			Model(&dao.TxNotification{}).
			Where("biz_id = ? AND `key` = ? AND status = 'PREPARE'", bizID, key).
			Updates(map[string]any{
				"status": status,
				"utime":  now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return dao.ErrUpdateStatusFailed
		}
		return tx.WithContext(ctx).
			Table(notificationDst.Table).
			Model(&dao.Notification{}).
			Where("biz_id = ? AND `key` = ? ", bizID, key).
			Updates(map[string]any{
				"status": notificationStatus,
				"utime":  now,
			}).Error
	})
}

// NewTxNShardingDAO creates a new TxNShardingDAO with the provided dependencies
func NewTxNShardingDAO(
	dbs *syncx.Map[string, *egorm.Component],
	nStrategy sharding.ShardingStrategy,
	txnStrategy sharding.ShardingStrategy,
) *TxNShardingDAO {
	return &TxNShardingDAO{
		dbs:                 dbs,
		nShardingStrategy:   nStrategy,
		txnShardingStrategy: txnStrategy,
	}
}
