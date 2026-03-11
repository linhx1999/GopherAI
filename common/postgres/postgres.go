package postgres

import (
	"GopherAI/config"
	"GopherAI/model"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitPostgres 初始化 PostgreSQL 连接
func InitPostgres() error {
	conf := config.GetConfig()
	host := conf.PostgresConfig.PostgresHost
	port := conf.PostgresConfig.PostgresPort
	user := conf.PostgresConfig.PostgresUser
	password := conf.PostgresConfig.PostgresPassword
	dbname := conf.PostgresConfig.PostgresDatabase
	sslmode := conf.PostgresConfig.PostgresSSLMode

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=Asia/Shanghai",
		host, user, password, dbname, port, sslmode)

	var log logger.Interface
	if gin.Mode() == "debug" {
		log = logger.Default.LogMode(logger.Info)
	} else {
		log = logger.Default
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: log,
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db

	// 启用 pgvector 扩展
	if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return fmt.Errorf("failed to enable pgvector: %w", err)
	}

	if err := resetLegacyTables(); err != nil {
		return fmt.Errorf("failed to reset legacy tables: %w", err)
	}

	// 自动迁移数据库表
	return DB.AutoMigrate(
		new(model.User),
		new(model.Session),
		new(model.File),
		new(model.DocumentChunk),
		new(model.Message),
	)
}

// CreateVectorIndex 创建向量索引（IVFFlat 算法，余弦相似度）
// 需要在有一定数据量后创建才能获得最佳性能
func CreateVectorIndex(dimension int) error {
	// 检查索引是否已存在
	var exists bool
	err := DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes 
			WHERE indexname = 'document_chunks_embedding_idx'
		)
	`).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}

	if exists {
		return nil
	}

	// 创建 IVFFlat 索引（lists 参数根据数据量调整，通常为 行数/1000）
	// 对于小数据集，可以先不创建索引，等待数据量增长后再创建
	sql := fmt.Sprintf(`
		CREATE INDEX document_chunks_embedding_idx 
		ON document_chunks 
		USING ivfflat (embedding vector_cosine_ops)
		WITH (lists = 100)
	`)
	return DB.Exec(sql).Error
}

// DropVectorIndex 删除向量索引
func DropVectorIndex() error {
	return DB.Exec("DROP INDEX IF EXISTS document_chunks_embedding_idx").Error
}

// InsertUser 插入用户
func InsertUser(user *model.User) (*model.User, error) {
	err := DB.Create(&user).Error
	return user, err
}

// GetUserByUsername 根据用户名查询用户
func GetUserByUsername(username string) (*model.User, error) {
	user := new(model.User)
	err := DB.Where("username = ?", username).First(user).Error
	return user, err
}

// GetUserByUserID 根据业务 UserID 查询用户
func GetUserByUserID(userID string) (*model.User, error) {
	user := new(model.User)
	err := DB.Where("user_id = ?", userID).First(user).Error
	return user, err
}

func resetLegacyTables() error {
	tableColumns := map[string][]string{
		"users":           {"id", "user_id", "username"},
		"sessions":        {"id", "session_id", "user_ref_id"},
		"files":           {"id", "file_id", "user_ref_id"},
		"document_chunks": {"id", "chunk_id", "file_ref_id"},
		"messages":        {"id", "message_id", "session_ref_id", "user_ref_id"},
	}

	for table, columns := range tableColumns {
		if !DB.Migrator().HasTable(table) {
			continue
		}
		for _, column := range columns {
			if DB.Migrator().HasColumn(table, column) {
				continue
			}
			log.Printf("detected legacy schema on table %s, dropping identity-related tables", table)
			return dropIdentityTables()
		}
	}

	return nil
}

func dropIdentityTables() error {
	tables := []string{"messages", "document_chunks", "files", "sessions", "users"}
	for _, table := range tables {
		if !DB.Migrator().HasTable(table) {
			continue
		}
		if err := DB.Migrator().DropTable(table); err != nil {
			return err
		}
	}
	return nil
}
