package dao

import "github.com/ego-component/egorm"

func InitTables(db *egorm.Component) error {
	return db.AutoMigrate(
		&BusinessConfig{},
		&Notification{},
		&TxNotification{},
		&CallbackLog{},
		&Provider{},
		&ChannelTemplate{},
		&ChannelTemplateVersion{},
		&ChannelTemplateProvider{},
		&Quota{},
	)
}
