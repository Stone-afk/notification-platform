//go:build wireinject

package provider

import (
	"github.com/google/wire"
	"notification-platform/internal/repository"
	"notification-platform/internal/repository/dao"
	providersvc "notification-platform/internal/service/provider/manage"
	testioc "notification-platform/internal/test/ioc"
)

func Init() providersvc.Service {
	wire.Build(
		testioc.BaseSet,
		repository.NewProviderRepository,
		providersvc.NewProviderService,
		dao.NewProviderDAO,
	)
	return nil
}
