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

type CloudStorageConfig struct {
	Base64Json string `yaml:"base64_json"`
	// Type                    string `yaml:"type" json:"type"`
	// ProjectId               string `yaml:"project_id" json:"project_id"`
	// Private_key_id          string `yaml:"private_key_id" json:"private_key_id"`
	// Private_key             string `yaml:"private_key" json:"private_key"`
	// ClientEmail             string `yaml:"client_email" json:"client_email"`
	// ClientID                string `yaml:"client_id" json:"client_id"`
	// AuthUri                 string `yaml:"auth_uri" json:"auth_uri"`
	// TokenUri                string `yaml:"token_uri" json:"token_uri"`
	// AuthProviderX509CertUrl string `yaml:"auth_provider_x509_cert_url" json:"auth_provider_x509_cert_url"`
	// ClientX509CertUrl       string `yaml:"client_x509_cert_url" json:"client_x509_cert_url"`
}

type cloudStorageHandler struct {
	bucket            string
	expireDays        int
	config            *CloudStorageConfig
	cloudStorgeClient *storage.Client
	once              *sync.Once
}

func newCloudStorageHandler(bucket string, expireDays int, config *CloudStorageConfig) (DFSHandler, error) {
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
		bucket:            bucket,
		expireDays:        expireDays,
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
