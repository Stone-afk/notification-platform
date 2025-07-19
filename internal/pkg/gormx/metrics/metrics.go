package metrics

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
	"time"
)

const (
	// 影响行数直方图配置
	rowAffectedBucketStart = 1  // 起始桶值
	rowAffectedBucketWidth = 10 // 桶宽度
	rowAffectedBucketCount = 10 // 桶数量

	// 摘要指标配置 - 分位数和允许误差
	summaryMaxAgeMinutes  = 5     // 摘要窗口期(分钟)
	summaryP50ErrorMargin = 0.05  // P50分位数允许误差
	summaryP90ErrorMargin = 0.01  // P90分位数允许误差
	summaryP95ErrorMargin = 0.005 // P95分位数允许误差
	summaryP99ErrorMargin = 0.001 // P99分位数允许误差
)

// GormMetricsPlugin 是一个实现了gorm.Plugin接口的度量插件
// 它为所有数据库操作收集指标：响应时间、请求数和错误数
type GormMetricsPlugin struct {
	// Prometheus 指标
	requestCount *prometheus.CounterVec
	responseTime *prometheus.SummaryVec // 使用SummaryVec记录响应时间
	errorCount   *prometheus.CounterVec
	rowsAffected *prometheus.HistogramVec

	// 指标注册表
	registry prometheus.Registerer
}

// Name 返回插件名称
func (p *GormMetricsPlugin) Name() string {
	return "GormMetricsPlugin"
}

// NewGormMetricsPlugin 创建一个新的GORM度量插件
func NewGormMetricsPlugin() *GormMetricsPlugin {
	registry := prometheus.DefaultRegisterer

	requestCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gorm",
			Name:      "requests_total",
			Help:      "Total number of GORM database operations.",
		},
		[]string{"operation", "table"},
	)

	responseTime := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: "gorm",
			Name:      "response_time_seconds",
			Help:      "Response time of GORM database operations in seconds.",
			Objectives: map[float64]float64{
				0.5:  summaryP50ErrorMargin,
				0.9:  summaryP90ErrorMargin,
				0.95: summaryP95ErrorMargin,
				0.99: summaryP99ErrorMargin,
			},
			MaxAge: time.Minute * summaryMaxAgeMinutes,
		},
		[]string{"operation", "table", "status"},
	)

	errorCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gorm",
			Name:      "errors_total",
			Help:      "Total number of GORM database operation errors.",
		},
		[]string{"operation", "table", "error_type"},
	)

	rowsAffected := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "gorm",
			Name:      "rows_affected",
			Help:      "Number of rows affected by GORM database operations.",
			Buckets:   prometheus.LinearBuckets(rowAffectedBucketStart, rowAffectedBucketWidth, rowAffectedBucketCount),
		},
		[]string{"operation", "table"},
	)

	// 注册指标
	registry.MustRegister(requestCount, responseTime, errorCount, rowsAffected)

	return &GormMetricsPlugin{
		requestCount: requestCount,
		responseTime: responseTime,
		errorCount:   errorCount,
		rowsAffected: rowsAffected,
		registry:     registry,
	}
}

// Initialize 初始化插件，注册GORM回调
func (p *GormMetricsPlugin) Initialize(db *gorm.DB) error {
	// 查询操作
	if err := db.Callback().Query().Before("gorm:query").Register("metrics:before_query", p.beforeQuery); err != nil {
		return err
	}
	if err := db.Callback().Query().After("gorm:query").Register("metrics:after_query", p.afterQuery); err != nil {
		return err
	}

	// 创建操作
	if err := db.Callback().Create().Before("gorm:create").Register("metrics:before_create", p.beforeCreate); err != nil {
		return err
	}
	if err := db.Callback().Create().After("gorm:create").Register("metrics:after_create", p.afterCreate); err != nil {
		return err
	}

	// 更新操作
	if err := db.Callback().Update().Before("gorm:update").Register("metrics:before_update", p.beforeUpdate); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("metrics:after_update", p.afterUpdate); err != nil {
		return err
	}

	// 删除操作
	if err := db.Callback().Delete().Before("gorm:delete").Register("metrics:before_delete", p.beforeDelete); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("metrics:after_delete", p.afterDelete); err != nil {
		return err
	}

	// 原始SQL操作
	if err := db.Callback().Raw().Before("gorm:raw").Register("metrics:before_raw", p.beforeRaw); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("metrics:after_raw", p.afterRaw); err != nil {
		return err
	}

	return nil
}

// 辅助函数：从GORM DB中获取表名和操作类型
func getTableAndOperation(db *gorm.DB) (tableName, operation string) {
	const unknowStr = "unknow"
	tableName = unknowStr
	operation = unknowStr

	// 获取表名
	if db.Statement.Schema != nil {
		tableName = db.Statement.Schema.Table
	} else if db.Statement.Table != "" {
		tableName = db.Statement.Table
	}
	// 确定操作类型
	if db.Statement.SQL.String() != "" {
		sqlUpper := db.Statement.SQL.String()
		switch {
		case len(sqlUpper) >= 6 && sqlUpper[:6] == "SELECT":
			operation = "select"
		case len(sqlUpper) >= 6 && sqlUpper[:6] == "INSERT":
			operation = "insert"
		case len(sqlUpper) >= 6 && sqlUpper[:6] == "UPDATE":
			operation = "update"
		case len(sqlUpper) >= 6 && sqlUpper[:6] == "DELETE":
			operation = "delete"
		}
	}

	return tableName, operation
}

// 开始计时并增加请求计数
func (p *GormMetricsPlugin) startTimer(db *gorm.DB, operation string) {
	table, op := getTableAndOperation(db)
	if op != "unknown" {
		operation = op
	}

	// 增加请求计数
	p.requestCount.WithLabelValues(operation, table).Inc()

	// 存储开始时间
	startTime := time.Now()
	db.Set("metrics:start_time", startTime)
	db.Set("metrics:operation", operation)
	db.Set("metrics:table", table)
}

// 结束计时并记录指标
func (p *GormMetricsPlugin) stopTimer(db *gorm.DB) {
	// 获取开始时间和操作信息
	startTimeValue, exists := db.Get("metrics:start_time")
	if !exists {
		return
	}
	startTime, ok := startTimeValue.(time.Time)
	if !ok {
		return
	}

	// 计算持续时间
	duration := time.Since(startTime).Seconds()
	// 获取表名和操作类型
	operationValue, _ := db.Get("metrics:operation")
	tableValue, _ := db.Get("metrics:table")
	operation, _ := operationValue.(string)
	table, _ := tableValue.(string)

	// 设置状态标签
	status := "success"
	if db.Error != nil {
		status = "error"
		errorType := "unknown_error"
		// 检查是否为记录未找到错误
		if errors.Is(db.Error, gorm.ErrRecordNotFound) {
			errorType = "record_not_found"
		}
		// 增加错误计数
		p.errorCount.WithLabelValues(operation, table, errorType).Inc()
	}
	// 记录响应时间
	p.responseTime.WithLabelValues(operation, table, status).Observe(duration)
	// 记录影响的行数
	if db.Statement.RowsAffected > 0 {
		p.rowsAffected.WithLabelValues(operation, table).Observe(float64(db.Statement.RowsAffected))
	}
}

// 查询操作的回调
func (p *GormMetricsPlugin) beforeQuery(db *gorm.DB) {
	p.startTimer(db, "select")
}

func (p *GormMetricsPlugin) afterQuery(db *gorm.DB) {
	p.stopTimer(db)
}

// 创建操作的回调
func (p *GormMetricsPlugin) beforeCreate(db *gorm.DB) {
	p.startTimer(db, "insert")
}

func (p *GormMetricsPlugin) afterCreate(db *gorm.DB) {
	p.stopTimer(db)
}

// 更新操作的回调
func (p *GormMetricsPlugin) beforeUpdate(db *gorm.DB) {
	p.startTimer(db, "update")
}

func (p *GormMetricsPlugin) afterUpdate(db *gorm.DB) {
	p.stopTimer(db)
}

// 删除操作的回调
func (p *GormMetricsPlugin) beforeDelete(db *gorm.DB) {
	p.startTimer(db, "delete")
}

func (p *GormMetricsPlugin) afterDelete(db *gorm.DB) {
	p.stopTimer(db)
}

// 原始SQL操作的回调
func (p *GormMetricsPlugin) beforeRaw(db *gorm.DB) {
	p.startTimer(db, "raw")
}

func (p *GormMetricsPlugin) afterRaw(db *gorm.DB) {
	p.stopTimer(db)
}

// GetMetrics 返回所有配置的Prometheus指标，可用于测试
func (p *GormMetricsPlugin) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"requestCount": p.requestCount,
		"responseTime": p.responseTime,
		"errorCount":   p.errorCount,
		"rowsAffected": p.rowsAffected,
	}
}
