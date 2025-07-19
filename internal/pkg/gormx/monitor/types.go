package monitor

//go:generate mockgen -source=./monitor.go -package=monitormocks -destination=./mocks/monitor.mock.go DBMonitor
type DBMonitor interface {
	Health() bool
	// Report 上报数据库调用时的error,来收集调用时的报错。
	Report(err error)
}
