package minio

import (
	"GopherAI/config"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var Client *minio.Client

// InitMinIO 初始化 MinIO 客户端并创建 Bucket
func InitMinIO() error {
	cfg := config.GetConfig()

	// 创建 MinIO 客户端
	client, err := minio.New(cfg.MinioConfig.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioConfig.AccessKey, cfg.MinioConfig.SecretKey, ""),
		Secure: cfg.MinioConfig.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to create minio client: %w", err)
	}

	Client = client

	// 创建 Bucket（如果不存在）
	ctx := context.Background()
	bucketName := cfg.MinioConfig.Bucket

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("MinIO bucket '%s' created successfully", bucketName)
	} else {
		log.Printf("MinIO bucket '%s' already exists", bucketName)
	}

	log.Println("MinIO client initialized successfully")
	return nil
}

// UploadFile 上传文件到 MinIO
func UploadFile(ctx context.Context, objectName string, reader io.Reader, objectSize int64, contentType string) error {
	cfg := config.GetConfig()
	bucketName := cfg.MinioConfig.Bucket

	_, err := Client.PutObject(ctx, bucketName, objectName, reader, objectSize, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// DownloadFile 从 MinIO 下载文件
func DownloadFile(ctx context.Context, objectName string) (*minio.Object, error) {
	cfg := config.GetConfig()
	bucketName := cfg.MinioConfig.Bucket

	object, err := Client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	return object, nil
}

// DeleteFile 从 MinIO 删除文件
func DeleteFile(ctx context.Context, objectName string) error {
	cfg := config.GetConfig()
	bucketName := cfg.MinioConfig.Bucket

	err := Client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetFileURL 获取文件的临时访问 URL（有效期 7 天）
func GetFileURL(ctx context.Context, objectName string) (string, error) {
	cfg := config.GetConfig()
	bucketName := cfg.MinioConfig.Bucket

	// 设置 URL 有效期为 7 天
	url, err := Client.PresignedGetObject(ctx, bucketName, objectName, 7*24*60*60, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get file URL: %w", err)
	}

	return url.String(), nil
}

// ListFiles 列出用户的所有文件
func ListFiles(ctx context.Context, prefix string) ([]minio.ObjectInfo, error) {
	cfg := config.GetConfig()
	bucketName := cfg.MinioConfig.Bucket

	var files []minio.ObjectInfo

	// 创建对象通道
	objectCh := Client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		files = append(files, object)
	}

	return files, nil
}
