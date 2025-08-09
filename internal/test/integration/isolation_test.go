//go:build e2e

package integration

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	pkgis "notification-platform/internal/pkg/gormx/isolation"
	"notification-platform/internal/test/integration/testdata/isolation"
	"notification-platform/internal/test/ioc"
)

type User1 struct {
	ID   int64  `gorm:"primaryKey"`
	Name string `gorm:"column:name"`
}

type IsolationSuite struct {
	suite.Suite
	coreDB      *isolation.DB
	nonCoresDB  *isolation.DB
	isolationDB *pkgis.DB
	gormDB      *gorm.DB
}

func (s *IsolationSuite) SetupSuite() {
	dsn := "root:root@tcp(localhost:13316)/notification?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=True&loc=Local&timeout=1s&readTimeout=3s&writeTimeout=3s&multiStatements=true&interpolateParams=true"
	db, err := sql.Open("mysql", dsn)
	require.NoError(s.T(), err)
	s.coreDB = &isolation.DB{
		Name: "core",
		DB:   db,
	}

	db1, err := sql.Open("mysql", dsn)
	require.NoError(s.T(), err)
	s.nonCoresDB = &isolation.DB{
		Name: "nonCore",
		DB:   db1,
	}
	ioc.WaitForDBSetup(dsn)

	s.isolationDB = pkgis.NewIsolationDB(s.coreDB, s.nonCoresDB)
	s.gormDB = ioc.InitDBWithCustomConnPool(s.isolationDB)

	err = s.gormDB.AutoMigrate(&User1{})
	if err != nil {
		s.T().Fatal(err)
	}
}

func (s *IsolationSuite) TearDownSuite() {
	s.gormDB.Exec("Truncate  table `user1`")
}

func (s *IsolationSuite) TestIsolation() {
	t := s.T()
	// 非核心表
	for i := 1; i < 10; i++ {
		us := &User1{
			ID:   int64(i),
			Name: "test",
		}
		err := s.gormDB.WithContext(t.Context()).Create(us).Error
		require.NoError(t, err)
	}
	assert.True(t, s.nonCoresDB.Count > 0)
	assert.True(t, s.coreDB.Count == 0)
	// 核心表

	for i := 11; i < 20; i++ {
		err := s.gormDB.WithContext(pkgis.WithCore(t.Context())).Create(&User1{ID: int64(i), Name: "test"}).Error
		require.NoError(t, err)
	}
	assert.True(t, s.coreDB.Count > 0)
}

func TestIsolationSuite(t *testing.T) {
	suite.Run(t, new(IsolationSuite))
}
