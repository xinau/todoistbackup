package store

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/xinau/todoistbackup/internal/client"
	"github.com/xinau/todoistbackup/internal/config"
)

type Store struct {
	client *minio.Client
	bucket string
}

func NewStore(config *config.StoreConfig) (*Store, error) {
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: !config.Insecure,
		Region: config.Region,
	})
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(context.TODO(), config.Bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		err := client.MakeBucket(context.TODO(), config.Bucket, minio.MakeBucketOptions{
			Region: config.Region,
		})
		if err != nil {
			return nil, err
		}
	}

	return &Store{
		client: client,
		bucket: config.Bucket,
	}, nil
}

func (s *Store) ListVersions(ctx context.Context) (map[string]struct{}, error) {
	versions := make(map[string]struct{})
	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		WithMetadata: true,
		Recursive:    true,
	}) {
		version := object.UserMetadata["Version"]
		if len(version) == 0 {
			var err error
			version, err = ToVersion(object.Key)
			if err != nil {
				log.Printf("getting backup version: %s", err)
				continue
			}
		}
		versions[version] = struct{}{}
	}
	return versions, nil
}

func (s *Store) PutBackup(ctx context.Context, backup *client.Backup, reader io.Reader) error {
	key := FromVersion(backup.Version)
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, backup.Metadata.Size, minio.PutObjectOptions{
		ContentType:        backup.Metadata.ContentType,
		ContentDisposition: backup.Metadata.ContentDisposition,
		Internal: minio.AdvancedPutOptions{
			SourceETag:  backup.Metadata.ETag,
			SourceMTime: backup.Metadata.LastModified,
		},
		UserMetadata: map[string]string{
			"Version": backup.Version,
		},
	})
	return err
}

func FromVersion(str string) string {
	str = strings.ReplaceAll(str, " ", "-")
	str = strings.ReplaceAll(str, ":", "-")
	return fmt.Sprintf("todoist-backup-%s.zip", str)
}

func ToVersion(str string) (string, error) {
	str = strings.TrimPrefix(str, "todoist-backup-")
	str = strings.TrimSuffix(str, ".zip")

	parts := strings.Split(str, "-")
	if len(parts) != 5 {
		return "", fmt.Errorf("parsing version %q", str)
	}

	return fmt.Sprintf("%s-%s-%s %s:%s", parts[0], parts[1], parts[2], parts[3], parts[4]), nil
}
