package sharding

import (
	"context"
	"errors"
	"fmt"
	"github.com/ecodeclub/ekit/syncx"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"notification-platform/internal/domain"
	shardingStr "notification-platform/internal/pkg/sharding"
	"notification-platform/internal/repository/dao"
	"strings"
	"time"
)

var _ dao.TxNotificationDAO = (*TxnTaskDAO)(nil)

type TxnTaskDAO struct {
	txnStr shardingStr.ShardingStrategy
	dbs    *syncx.Map[string, *egorm.Component]
}

func (t *TxnTaskDAO) FindCheckBack(ctx context.Context, offset, limit int) ([]dao.TxNotification, error) {
	db, txnTab, _, err := t.getDBTabFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	gormDB, ok := t.dbs.Load(db)
	if !ok {
		return nil, fmt.Errorf("未知库名 %s", db)
	}
	var txns []dao.TxNotification
	currentTime := time.Now().UnixMilli()
	err = gormDB.WithContext(ctx).
		Table(txnTab).
		Where("status = ? AND next_check_time <= ? AND next_check_time > 0", domain.TxNotificationStatusPrepare, currentTime).
		Offset(offset).
		Limit(limit).
		Order("next_check_time").
		Find(&txns).Error
	return txns, err
}

func (t *TxnTaskDAO) UpdateCheckStatus(ctx context.Context, txNotifications []dao.TxNotification, status domain.SendStatus) error {
	db, txnTab, ntab, err := t.getDBTabFromCtx(ctx)
	if err != nil {
		return err
	}
	gormDB, ok := t.dbs.Load(db)
	if !ok {
		return fmt.Errorf("未知库名 %s", db)
	}
	sqls := make([]string, 0, len(txNotifications))
	now := time.Now().UnixMilli()
	notificationIDs := make([]uint64, 0, len(txNotifications))
	for _, txNotification := range txNotifications {
		updateSQL := fmt.Sprintf("UPDATE `%s` set `status` = '%s',`utime` = %d ,`next_check_time` = %d,`check_count` = %d WHERE `key` = '%s' AND `biz_id` = %d AND `status` = 'PREPARE'", txnTab, txNotification.Status, now, txNotification.NextCheckTime, txNotification.CheckCount, txNotification.Key, txNotification.BizID)
		sqls = append(sqls, updateSQL)
		notificationIDs = append(notificationIDs, txNotification.NotificationID)
	}
	// 拼接所有SQL并执行
	// UPDATE xxx; UPDATE xxx;UPDATE xxx;
	if len(sqls) > 0 {
		return gormDB.Transaction(func(tx *gorm.DB) error {
			combinedSQL := strings.Join(sqls, "; ")
			err := tx.WithContext(ctx).Exec(combinedSQL).Error
			if err != nil {
				return err
			}
			if status != domain.SendStatusPrepare {
				return tx.
					Table(ntab).
					WithContext(ctx).Model(&dao.Notification{}).
					Where("id in ?", notificationIDs).
					Update("status", status).Error
			}
			return nil
		})
	}
	return nil
}

func (t *TxnTaskDAO) getDBTabFromCtx(ctx context.Context) (db, txnTab, ntab string, err error) {
	dst, ok := shardingStr.DstFromCtx(ctx)
	if !ok {
		return "", "", "", errors.New("ctx未找到表名")
	}
	txTab := t.txnStr.ExtractSuffixAndFormatFromTable(dst.Table)
	return dst.DB, txTab, dst.Table, nil
}

func (t *TxnTaskDAO) Create(ctx context.Context, notification dao.TxNotification) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TxnTaskDAO) First(_ context.Context, _ int64) (dao.TxNotification, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TxnTaskDAO) BatchGetTxNotification(_ context.Context, _ []int64) (map[int64]dao.TxNotification, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TxnTaskDAO) GetByBizIDKey(_ context.Context, _ int64, _ string) (dao.TxNotification, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TxnTaskDAO) UpdateNotificationID(_ context.Context, _ int64, _ string, _ uint64) error {
	// TODO implement me
	panic("implement me")
}

func (t *TxnTaskDAO) Prepare(_ context.Context, _ dao.TxNotification, _ dao.Notification) (uint64, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TxnTaskDAO) UpdateStatus(_ context.Context, _ int64, _ string, _ domain.TxNotificationStatus, _ domain.SendStatus) error {
	// TODO implement me
	panic("implement me")
}

func NewTxnTaskDAO(dbs *syncx.Map[string, *egorm.Component], txnStr shardingStr.ShardingStrategy) *TxnTaskDAO {
	return &TxnTaskDAO{
		dbs:    dbs,
		txnStr: txnStr,
	}
}
