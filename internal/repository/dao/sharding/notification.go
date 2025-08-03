package sharding

import (
	"context"
	"errors"
	"fmt"
	"github.com/ecodeclub/ekit/list"
	"github.com/ecodeclub/ekit/mapx"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/syncx"
	"github.com/ego-component/egorm"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"notification-platform/internal/domain"
	"notification-platform/internal/errs"
	idgen "notification-platform/internal/pkg/idgenerator"
	"notification-platform/internal/pkg/sharding"
	"notification-platform/internal/repository/dao"
	"strings"
	"sync"
	"time"
)

var _ dao.NotificationDAO = (*NotificationShardingDAO)(nil)

type modifyIds struct {
	callbackTab string
	successIds  []uint64
	failedIds   []uint64
}

func (m *modifyIds) failToStr() string {
	return m.listToStr(m.failedIds)
}

func (m *modifyIds) successToStr() string {
	return m.listToStr(m.successIds)
}

type Query struct {
	SQL  string
	Args []any
}

func (m *modifyIds) listToStr(list []uint64) string {
	strSlice := make([]string, len(list))
	for i, num := range list {
		strSlice[i] = fmt.Sprintf("%d", num)
	}
	return strings.Join(strSlice, ",")
}

type NotificationShardingDAO struct {
	dbs                          *syncx.Map[string, *egorm.Component]
	notificationShardingStrategy sharding.ShardingStrategy
	callbackLogShardingStrategy  sharding.ShardingStrategy
	idGenerator                  *idgen.Generator
}

func (s *NotificationShardingDAO) Create(ctx context.Context, data dao.Notification) (dao.Notification, error) {
	return s.create(ctx, data, false)
}

func (s *NotificationShardingDAO) CreateWithCallbackLog(ctx context.Context, data dao.Notification) (dao.Notification, error) {
	return s.create(ctx, data, true)
}

func (s *NotificationShardingDAO) BatchCreate(ctx context.Context, dataList []dao.Notification) ([]dao.Notification, error) {
	return s.batchCreate(ctx, dataList, false)
}

func (s *NotificationShardingDAO) BatchCreateWithCallbackLog(ctx context.Context, datas []dao.Notification) ([]dao.Notification, error) {
	return s.batchCreate(ctx, datas, true)
}

func (s *NotificationShardingDAO) GetByID(ctx context.Context, id uint64) (dao.Notification, error) {
	dst := s.notificationShardingStrategy.ShardWithID(int64(id))
	gormdb, ok := s.dbs.Load(dst.DB)
	if !ok {
		return dao.Notification{}, fmt.Errorf("未知库名 %s", dst.DB)
	}
	var data dao.Notification
	err := gormdb.WithContext(ctx).Table(dst.Table).
		Where("id = ?", id).
		First(&data).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dao.Notification{}, fmt.Errorf("%w: id=%d", errs.ErrNotificationNotFound, id)
		}
		return dao.Notification{}, err
	}
	return data, nil
}

func (s *NotificationShardingDAO) BatchGetByIDs(ctx context.Context, ids []uint64) (map[uint64]dao.Notification, error) {
	idsMap := make(map[[2]string][]uint64, len(ids))
	for _, id := range ids {
		dst := s.notificationShardingStrategy.ShardWithID(int64(id))
		v, ok := idsMap[[2]string{
			dst.DB,
			dst.Table,
		}]
		if ok {
			v = append(v, id)
		} else {
			v = []uint64{id}
		}
		idsMap[[2]string{
			dst.DB,
			dst.Table,
		}] = v
	}
	notifiMap := make(map[uint64]dao.Notification, len(idsMap))
	mu := new(sync.RWMutex)
	var eg errgroup.Group
	for key := range idsMap {
		dbTab := key
		dbIds := idsMap[dbTab]
		eg.Go(func() error {
			var notifications []dao.Notification
			dbName := dbTab[0]
			tableName := dbTab[1]
			gormdb, ok := s.dbs.Load(dbName)
			if !ok {
				return fmt.Errorf("未知库名 %s", dbName)
			}
			err := gormdb.WithContext(ctx).
				Table(tableName).
				Where("id in (?)", dbIds).
				Find(&notifications).Error

			for idx := range notifications {
				notification := notifications[idx]
				mu.Lock()
				notifiMap[notification.ID] = notification
				mu.Unlock()
			}
			return err
		})
	}
	return notifiMap, eg.Wait()
}

func (s *NotificationShardingDAO) GetByKey(ctx context.Context, bizID int64, key string) (dao.Notification, error) {
	dst := s.notificationShardingStrategy.Shard(bizID, key)
	gormdb, ok := s.dbs.Load(dst.DB)
	if !ok {
		return dao.Notification{}, fmt.Errorf("未知库名 %s", dst.DB)
	}
	var data dao.Notification
	err := gormdb.WithContext(ctx).Table(dst.Table).
		Where("`key` = ? AND `biz_id` = ?", key, bizID).
		First(&data).Error
	if err != nil {
		return dao.Notification{}, fmt.Errorf("查询通知列表失败:bizID: %d, key %s %w", bizID, key, err)
	}
	return data, nil
}

func (s *NotificationShardingDAO) GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]dao.Notification, error) {
	notiMap := make(map[[2]string][]string, len(keys))
	for idx := range keys {
		key := keys[idx]
		dst := s.notificationShardingStrategy.Shard(bizID, key)
		v, ok := notiMap[[2]string{
			dst.DB,
			dst.Table,
		}]
		if !ok {
			v = []string{
				key,
			}
		} else {
			v = append(v, key)
		}
		notiMap[[2]string{
			dst.DB,
			dst.Table,
		}] = v
	}

	var eg errgroup.Group

	notificationList := list.NewArrayList[dao.Notification](len(keys))
	curList := list.ConcurrentList[dao.Notification]{
		List: notificationList,
	}
	for tabDBKey, ks := range notiMap {
		eg.Go(func() error {
			dbName := tabDBKey[0]
			tabName := tabDBKey[1]
			gromDB, ok := s.dbs.Load(dbName)
			if !ok {
				return fmt.Errorf("未知库名 %s", dbName)
			}
			var data []dao.Notification
			err := gromDB.WithContext(ctx).
				Table(tabName).Where("`key` in ? AND `biz_id` = ?", ks, bizID).Find(&data).Error
			if err != nil {
				return err
			}

			// Lock before modifying the shared list
			return curList.Append(data...)
		})
	}
	err := eg.Wait()
	return curList.AsSlice(), err
}

func (s *NotificationShardingDAO) CASStatus(ctx context.Context, notification dao.Notification) error {
	updates := map[string]any{
		"status":  notification.Status,
		"version": gorm.Expr("version + 1"),
		"utime":   time.Now().Unix(),
	}
	dst := s.notificationShardingStrategy.ShardWithID(int64(notification.ID))
	// dst := s.notificationShardingStrategy.Shard(notification.BizID, notification.Key)
	gormDB, ok := s.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("未知库名 %s", dst.DB)
	}
	result := gormDB.WithContext(ctx).
		Table(dst.Table).
		Where("id = ? AND version = ?", notification.ID, notification.Version).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected < 1 {
		return fmt.Errorf("并发竞争失败 %w, id %d", errs.ErrNotificationVersionMismatch, notification.ID)
	}
	return nil
}

func (s *NotificationShardingDAO) UpdateStatus(ctx context.Context, notification dao.Notification) error {
	dst := s.notificationShardingStrategy.ShardWithID(int64(notification.ID))
	gormDB, ok := s.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("未知库名 %s", dst.DB)
	}
	return gormDB.WithContext(ctx).
		Table(dst.Table).
		Model(&dao.Notification{}).
		Where("id = ?", notification.ID).
		Updates(map[string]any{
			"status":  notification.Status,
			"version": gorm.Expr("version + 1"),
			"utime":   time.Now().Unix(),
		}).Error
}

func (s *NotificationShardingDAO) BatchUpdateStatusSucceededOrFailed(ctx context.Context, successNotifications, failedNotifications []dao.Notification) error {
	if len(successNotifications) == 0 && len(failedNotifications) == 0 {
		return nil
	}

	dbMap := make(map[string]map[string]*modifyIds)

	for idx := range successNotifications {
		notification := successNotifications[idx]
		notificationDst := s.notificationShardingStrategy.ShardWithID(int64(notification.ID))
		callbackDst := s.callbackLogShardingStrategy.ShardWithID(int64(notification.ID))

		tableMap, exists := dbMap[notificationDst.DB]
		if !exists {
			tableMap = make(map[string]*modifyIds)
			dbMap[notificationDst.DB] = tableMap
		}

		modifyID, exists := tableMap[notificationDst.Table]
		if !exists {
			modifyID = &modifyIds{
				callbackTab: callbackDst.Table,
				successIds:  []uint64{notification.ID},
				failedIds:   []uint64{},
			}
			tableMap[notificationDst.Table] = modifyID
		} else {
			modifyID.successIds = append(modifyID.successIds, notification.ID)
			if modifyID.callbackTab == "" {
				modifyID.callbackTab = callbackDst.Table
			}
		}
	}

	for idx := range failedNotifications {
		notification := failedNotifications[idx]
		notificationDst := s.notificationShardingStrategy.ShardWithID(int64(notification.ID))
		tableMap, exists := dbMap[notificationDst.DB]
		if !exists {
			tableMap = make(map[string]*modifyIds)
			dbMap[notificationDst.DB] = tableMap
		}
		modifyID, exists := tableMap[notificationDst.Table]
		if !exists {
			modifyID = &modifyIds{
				successIds: []uint64{},
				failedIds:  []uint64{notification.ID},
			}
			tableMap[notificationDst.Table] = modifyID
		} else {
			modifyID.failedIds = append(modifyID.failedIds, notification.ID)
		}
	}

	// Process each database in parallel
	var eg errgroup.Group
	for dbName, tableMap := range dbMap {
		db := dbName
		tables := tableMap
		eg.Go(func() error {
			gormDB, ok := s.dbs.Load(db)
			if !ok {
				return fmt.Errorf("未知库名 %s", db)
			}
			return gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				return s.batchMark(tx, tables)
			})
		})
	}

	return eg.Wait()
}

// FindReadyNotifications 这个是循环任务用的不在这个dao中实现
func (s *NotificationShardingDAO) FindReadyNotifications(_ context.Context, _, _ int) ([]dao.Notification, error) {
	// TODO implement me
	panic("implement me")
}

func (s *NotificationShardingDAO) MarkSuccess(ctx context.Context, entity dao.Notification) error {
	now := time.Now().UnixMilli()
	dst := s.notificationShardingStrategy.ShardWithID(int64(entity.ID))
	callbackLogDst := s.callbackLogShardingStrategy.ShardWithID(int64(entity.ID))
	gormDB, ok := s.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("未知库名 %s", dst.DB)
	}
	return gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.
			Table(dst.Table).
			Model(&dao.Notification{}).
			Where("id = ?", entity.ID).
			Updates(map[string]any{
				"status":  entity.Status,
				"utime":   now,
				"version": gorm.Expr("version + 1"),
			}).Error
		if err != nil {
			return err
		}
		// 要把 callback log 标记为可以发送了
		return tx.Model(&dao.CallbackLog{}).
			Table(callbackLogDst.Table).
			Where("notification_id = ?", entity.ID).
			Updates(map[string]any{
				// 标记为可以发送回调了
				"status": domain.CallbackLogStatusPending,
				"utime":  now,
			}).Error
	})
}

func (s *NotificationShardingDAO) MarkFailed(ctx context.Context, entity dao.Notification) error {
	now := time.Now().UnixMilli()
	dst := s.notificationShardingStrategy.ShardWithID(int64(entity.ID))
	gormDB, ok := s.dbs.Load(dst.DB)
	if !ok {
		return fmt.Errorf("未知库名 %s", dst.DB)
	}
	return gormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.
			Model(&dao.Notification{}).
			Table(dst.Table).
			Where("id = ?", entity.ID).
			Updates(map[string]any{
				"status":  entity.Status,
				"utime":   now,
				"version": gorm.Expr("version + 1"),
			}).Error
	})
}

func (s *NotificationShardingDAO) MarkTimeoutSendingAsFailed(_ context.Context, _ int) (int64, error) {
	// TODO implement me
	panic("implement me")
}

func (s *NotificationShardingDAO) batchMark(tx *gorm.DB, ids map[string]*modifyIds) error {
	now := time.Now().Unix()
	sqls := make([]string, 0, len(ids))
	for notificationTab := range ids {
		modifyID := ids[notificationTab]
		if len(modifyID.successIds) > 0 {
			notificationSQL := fmt.Sprintf("UPDATE %s SET `version` = `version` + 1,`utime` = %d,`status` = '%s' WHERE id IN (%s) ",
				notificationTab, now, domain.SendStatusSucceeded.String(), modifyID.successToStr(),
			)
			callbackSQL := fmt.Sprintf("UPDATE %s SET `status` = '%s',utime = %d  WHERE notification_id IN (%s)", modifyID.callbackTab, domain.CallbackLogStatusPending.String(), now, modifyID.successToStr())
			sqls = append(sqls, notificationSQL, callbackSQL)
		}
		if len(modifyID.failedIds) > 0 {
			notificationSQL := fmt.Sprintf("UPDATE %s SET `version` = `version` + 1,`utime` = %d,`status` = '%s' WHERE id IN (%s) ",
				notificationTab, now, domain.SendStatusFailed.String(), modifyID.failToStr(),
			)
			sqls = append(sqls, notificationSQL)
		}
	}

	if len(sqls) > 0 {
		combinedSQL := strings.Join(sqls, "; ")
		err := tx.Exec(combinedSQL).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *NotificationShardingDAO) batchCreate(ctx context.Context, datas []dao.Notification, createCallbackLog bool) ([]dao.Notification, error) {
	if len(datas) == 0 {
		return []dao.Notification{}, nil
	}

	now := time.Now().UnixMilli()
	for i := range datas {
		datas[i].Ctime, datas[i].Utime = now, now
		datas[i].Version = 1
	}
	notificationList := slice.Map(datas, func(_ int, src dao.Notification) *dao.Notification {
		return &src
	})
	return s.tryBatchInsert(ctx, notificationList, createCallbackLog)
}

// tryBatchInsert attempts to insert all notifications in batch mode
func (s *NotificationShardingDAO) tryBatchInsert(ctx context.Context, datas []*dao.Notification, createCallbackLog bool) ([]dao.Notification, error) {
	// db => notifications
	notiMap := s.getNotificationMap(datas)
	var eg errgroup.Group
	dbNames := notiMap.Keys()
	for _, dbName := range dbNames {
		// 不可能不存在
		notis, _ := notiMap.Get(dbName)
		gormDB, ok := s.dbs.Load(dbName)
		if !ok {
			return nil, fmt.Errorf("库名%s没找到", dbName)
		}
		eg.Go(func() error {
			for {
				sqls, args, ids := s.genSQLs(gormDB, notis, createCallbackLog)
				if len(sqls) > 0 {
					combinedSQL := strings.Join(sqls, "; ")
					err := gormDB.WithContext(ctx).Exec(combinedSQL, args...).Error
					if err != nil {
						if errors.Is(err, gorm.ErrDuplicatedKey) && checkNotificationIds(ids, err) {
							continue
						}
						return err
					}
				}
				return nil
			}
		})
	}
	err := eg.Wait()
	return slice.Map(datas, func(_ int, src *dao.Notification) dao.Notification {
		if src != nil {
			return *src
		}
		return dao.Notification{}
	}), err
}

func checkNotificationIds(ids []uint64, err error) bool {
	for _, id := range ids {
		if !dao.CheckErrIsIDDuplicate(id, err) {
			return true
		}
	}
	return false
}

func (s *NotificationShardingDAO) genSQLs(db *egorm.Component, notis []*dao.Notification, callbackLog bool) (sqls []string, args []any, ids []uint64) {
	now := time.Now().UnixMilli()
	sessionDB := db.Session(&gorm.Session{DryRun: true})
	ids = make([]uint64, 0, len(notis))
	// 可能需要 callback log，所以 * 2
	const sqlRate = 2
	sqls = make([]string, 0, len(notis)*sqlRate)
	// notification 的字段数量 + callback log 的字段数量
	const paramsRate = 21
	args = make([]any, 0, len(notis)*paramsRate)
	// 生成 SQL
	// notis 里面放的是指针，所以可以直接操作
	for _, noti := range notis {
		id := s.idGenerator.GenerateID(noti.BizID, noti.Key)
		noti.ID = uint64(id)
		ids = append(ids, noti.ID)
		dst := s.notificationShardingStrategy.Shard(noti.BizID, noti.Key)
		stmt := sessionDB.Table(dst.Table).Create(noti).Statement
		sqls = append(sqls, stmt.SQL.String())
		args = append(args, stmt.Vars...)
		if callbackLog {
			dst = s.callbackLogShardingStrategy.Shard(noti.BizID, noti.Key)
			stmt = sessionDB.Table(dst.Table).Create(&dao.CallbackLog{
				NotificationID: noti.ID,
				Status:         domain.CallbackLogStatusInit.String(),
				NextRetryTime:  now,
				Ctime:          now,
				Utime:          now,
			}).Statement
			sqls = append(sqls, stmt.SQL.String())
			args = append(args, stmt.Vars...)
		}
	}
	return sqls, args, ids
}

func (s *NotificationShardingDAO) getNotificationMap(datas []*dao.Notification) *mapx.MultiMap[string, *dao.Notification] {
	// 最多就是 32 个 DB
	const maxDB = 32
	notiMap := mapx.NewMultiBuiltinMap[string, *dao.Notification](maxDB)
	for idx := range datas {
		data := datas[idx]
		dst := s.notificationShardingStrategy.Shard(data.BizID, data.Key)
		_ = notiMap.Put(dst.DB, data)
	}
	return notiMap
}

func (s *NotificationShardingDAO) create(ctx context.Context, data dao.Notification, createCallbackLog bool) (dao.Notification, error) {
	now := time.Now().UnixMilli()

	data.Ctime, data.Utime = now, now
	data.Version = 1
	// 获取分库分表规则
	notificationDst := s.notificationShardingStrategy.Shard(data.BizID, data.Key)
	callBackLogDst := s.callbackLogShardingStrategy.Shard(data.BizID, data.Key)
	notiDB, ok := s.dbs.Load(notificationDst.DB)
	if !ok {
		return dao.Notification{}, fmt.Errorf("未知库名 %s", notificationDst.DB)
	}
	// notification 和 callbacklog 是在同一个库的但是 表不同
	err := notiDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for {
			data.ID = uint64(s.idGenerator.GenerateID(data.BizID, data.Key))
			if err := tx.Table(notificationDst.Table).Create(&data).Error; err != nil {
				if s.isUniqueConstraintError(err) {
					return fmt.Errorf("%w", errs.ErrNotificationDuplicate)
				}
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					// 唯一键冲突直接返回
					if !dao.CheckErrIsIDDuplicate(data.ID, err) {
						return nil
					}
					// 主键冲突 重试再找个主键
					continue
				}
				return err
			}
			if createCallbackLog {
				if err := tx.Table(callBackLogDst.Table).Create(&dao.CallbackLog{
					NotificationID: data.ID,
					Status:         domain.CallbackLogStatusInit.String(),
					NextRetryTime:  now,
					Ctime:          now,
					Utime:          now,
				}).Error; err != nil {
					return fmt.Errorf("%w", errs.ErrCreateCallbackLogFailed)
				}
			}
			return nil
		}
	})

	return data, err
}

// isUniqueConstraintError 检查是否是唯一索引冲突错误
func (s *NotificationShardingDAO) isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	me := new(mysql.MySQLError)
	if ok := errors.As(err, &me); ok {
		const uniqueIndexErrNo uint16 = 1062
		return me.Number == uniqueIndexErrNo
	}
	return false
}

func NewNotificationShardingDAO(dbs *syncx.Map[string, *egorm.Component],
	notificationShardingSvc sharding.ShardingStrategy,
	callbackLogShardingSvc sharding.ShardingStrategy,
	idGenerator *idgen.Generator,
) *NotificationShardingDAO {
	return &NotificationShardingDAO{
		dbs:                          dbs,
		notificationShardingStrategy: notificationShardingSvc,
		callbackLogShardingStrategy:  callbackLogShardingSvc,
		idGenerator:                  idGenerator,
	}
}
