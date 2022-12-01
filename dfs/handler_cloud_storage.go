package dfs

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type cloudStorageHandler struct {
	bucket            string
	expireDays        int
	config            *Config
	cloudStorgeClient *storage.Client
	once              *sync.Once
}

func newCloudStorageHandler(config *Config) (DFSHandler, error) {
	ctx := context.Background()

	str, err := base64.StdEncoding.DecodeString(config.Base64Json)
	if err != nil {
		return nil, err
	}

	clientOption := []option.ClientOption{
		option.WithCredentialsJSON(str),
	}

	client, err := storage.NewClient(ctx, clientOption...)
	if err != nil {
		return nil, err
	}

	handler := &cloudStorageHandler{
		bucket:            config.Bucket,
		expireDays:        config.ExpireDays,
		config:            config,
		cloudStorgeClient: client,
		once:              new(sync.Once),
	}
	return handler, err
}

func (h *cloudStorageHandler) TryMakeBucket() error {
	var err error
	if h.expireDays <= 0 {
		return nil
	}
	makeFun := func() {
		ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
		defer cf()

		_, err = h.cloudStorgeClient.Bucket(h.bucket).Update(ctx, storage.BucketAttrsToUpdate{
			Lifecycle: &storage.Lifecycle{
				Rules: []storage.LifecycleRule{
					{
						Action: storage.LifecycleAction{Type: "Delete"},
						Condition: storage.LifecycleCondition{
							AgeInDays: int64(h.expireDays),
						},
					},
				},
			},
		})
		if err != nil {
			return
		}
	}

	h.once.Do(makeFun)

	return err
}

func (h *cloudStorageHandler) GetObject(path, fileName string) ([]byte, error) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
	defer cf()

	if path[len(path)-1] != '/' {
		path += "/"
	}
	obj := h.cloudStorgeClient.Bucket(h.bucket).Object(path + fileName)
	sReader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer sReader.Close()

	payload, err := ioutil.ReadAll(sReader)
	return payload, err
}

func (h *cloudStorageHandler) PutObject(path, fileName string, payload []byte) error {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
	defer cf()
	contentType := "text/plain"

	if path[len(path)-1] != '/' {
		path += "/"
	}
	sWriter := h.cloudStorgeClient.Bucket(h.bucket).Object(path + fileName).NewWriter(ctx)
	sWriter.ContentType = contentType
	defer sWriter.Close()

	_, err := sWriter.Write(payload)
	return err
}
