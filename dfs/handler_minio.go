package dfs

import (
	"bytes"
	"context"
	"io/ioutil"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

type MinioConfig struct {
	Url    string `yaml:"url"`
	Bucket string `yaml:"bucket"`
	KeyID  string `yaml:"key_id"`
	Key    string `yaml:"key"`
}

type minioHandler struct {
	bucket      string
	expireDays  int
	config      *MinioConfig
	minioClient *minio.Client
	once        *sync.Once
}

func newMinioHandler(bucket string, expireDays int, config *MinioConfig) (DFSHandler, error) {
	minioClient, err := minio.New(config.Url, &minio.Options{
		Creds:  credentials.NewStaticV4(config.KeyID, config.Key, ""),
		Secure: false,
	})

	if err != nil {
		return nil, err
	}

	handler := &minioHandler{bucket: bucket, expireDays: expireDays, config: config, minioClient: minioClient, once: new(sync.Once)}
	handler.minioClient = minioClient
	return handler, nil
}

func (h *minioHandler) TryMakeBucket() error {
	var err error
	makeFun := func() {
		ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
		defer cf()
		err = h.minioClient.MakeBucket(ctx, h.bucket, minio.MakeBucketOptions{})
		if err != nil {
			// Check to see if we already own this bucket (which happens if you run this twice)
			exists, errBucketExists := h.minioClient.BucketExists(ctx, h.bucket)
			if errBucketExists == nil {
				if exists {
					err = nil
					return
				} else {
					return
				}
			}

			return
		} else {
			if h.expireDays > 0 {
				config := lifecycle.NewConfiguration()
				config.Rules = []lifecycle.Rule{
					{
						ID:     "expire-bucket",
						Status: "Enabled",
						Expiration: lifecycle.Expiration{
							Days: lifecycle.ExpirationDays(h.expireDays),
						},
					},
				}
				err = h.minioClient.SetBucketLifecycle(context.Background(), h.bucket, config)
				if err != nil {
					return
				}
			}
		}
		return
	}

	h.once.Do(makeFun)

	return err
}
func (h *minioHandler) PutObject(path, fileName string, payload []byte) error {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
	defer cf()
	contentType := "text/plain"

	if path[len(path)-1] != '/' {
		path += "/"
	}

	info, err := h.minioClient.PutObject(ctx, h.bucket, path+fileName, bytes.NewReader(payload), int64(len(payload)), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return err
	}

	if info.Size != 0 {

	}

	return nil
}

func (h *minioHandler) GetObject(path, fileName string) ([]byte, error) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
	defer cf()
	if path[len(path)-1] != '/' {
		path += "/"
	}
	obj, err := h.minioClient.GetObject(ctx, h.bucket, path+fileName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	payload, err := ioutil.ReadAll(obj)
	if err != nil {
		return nil, err
	}

	return payload, nil
}
