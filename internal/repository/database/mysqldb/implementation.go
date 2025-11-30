package mysqldb

import (
	"context"
	"time"

	aulogging "github.com/StephanHCB/go-autumn-logging"
	"github.com/eurofurence/reg-paygate-adapter/internal/entity"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/config"
	"github.com/eurofurence/reg-paygate-adapter/internal/repository/database/dbrepo"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type MysqlRepository struct {
	db  *gorm.DB
	Now func() time.Time
}

func Create() dbrepo.Repository {
	return &MysqlRepository{
		Now: time.Now,
	}
}

func (r *MysqlRepository) Open() error {
	gormConfig := gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "nexi_",
		},
		Logger: logger.Default.LogMode(logger.Silent),
	}
	connectString := config.DatabaseMysqlConnectString()

	db, err := gorm.Open(mysql.Open(connectString), &gormConfig)
	if err != nil {
		aulogging.Logger.NoCtx().Error().WithErr(err).Printf("failed to open mysql connection: %s", err.Error())
		return err
	}

	sqlDb, err := db.DB()
	if err != nil {
		aulogging.Logger.NoCtx().Error().WithErr(err).Printf("failed to configure mysql connection: %s", err.Error())
		return err
	}

	// see https://making.pusher.com/production-ready-connection-pooling-in-go/
	sqlDb.SetMaxOpenConns(100)
	sqlDb.SetMaxIdleConns(50)
	sqlDb.SetConnMaxLifetime(time.Minute * 10)

	r.db = db
	return nil
}

func (r *MysqlRepository) Close() {
	// no more db close in gorm v2
}

func (r *MysqlRepository) Migrate() error {
	err := r.db.AutoMigrate(
		&entity.ProtocolEntry{},
	)
	if err != nil {
		aulogging.Logger.NoCtx().Error().WithErr(err).Printf("failed to migrate mysql db: %s", err.Error())
		return err
	}
	return nil
}

// --- log entries ---

func (r *MysqlRepository) WriteProtocolEntry(ctx context.Context, e *entity.ProtocolEntry) error {
	err := r.db.Create(e).Error
	if err != nil {
		aulogging.Logger.Ctx(ctx).Warn().WithErr(err).Printf("mysql error during protocol entry insert: %s", err.Error())
	}
	return err
}
