package dfs

import (
	"bytes"
	"context"
	"io/ioutil"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type s3Handler struct {
	bucket     string
	expireDays int
	config     *Config
	s3Client   *s3.S3
	once       *sync.Once
}

func newS3Handler(config *Config) (DFSHandler, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(config.KeyID, config.Key, ""),
		Endpoint:         aws.String(config.Url),
		Region:           aws.String(endpoints.ApSoutheast1RegionID),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(false), // virtual-host style方式，不要修改
	})
	if err != nil {
		return nil, err
	}

	handler := &s3Handler{
		bucket:     config.Bucket,
		expireDays: config.ExpireDays,
		config:     config,
		s3Client:   s3.New(sess),
		once:       new(sync.Once),
	}

	return handler, nil
}

func (h *s3Handler) TryMakeBucket() error {
	bucket := h.bucket
	expireDays := h.expireDays
	var err error
	makeFun := func() {
		ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
		defer cf()
		_, err := h.s3Client.CreateBucketWithContext(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket),
		})
		if err != nil {
			// Check to see if we already own this bucket (which happens if you run this twice)
			errBucketExists := h.s3Client.WaitUntilBucketExists(&s3.HeadBucketInput{
				Bucket: aws.String(bucket),
			})
			if errBucketExists == nil {
				err = nil
			}
			return
		} else {
			if expireDays > 0 {
				config := &s3.LifecycleConfiguration{}
				config.Rules = []*s3.Rule{
					{
						ID:     aws.String("expire-bucket"),
						Status: aws.String("Enabled"),
						Expiration: &s3.LifecycleExpiration{
							Days: aws.Int64(int64(expireDays)),
						},
					},
				}
				_, err = h.s3Client.PutBucketLifecycle(&s3.PutBucketLifecycleInput{
					Bucket:                 aws.String(bucket),
					LifecycleConfiguration: config,
				})
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

func (h *s3Handler) GetObject(path, fileName string) ([]byte, error) {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
	defer cf()
	if path[len(path)-1] != '/' {
		path += "/"
	}

	out, err := h.s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(h.bucket),
		Key:    aws.String(path + fileName),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	payload, err := ioutil.ReadAll(out.Body)
	return payload, err
}

func (h *s3Handler) PutObject(path, fileName string, payload []byte) error {
	ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
	defer cf()
	contentType := "text/plain"

	if path[len(path)-1] != '/' {
		path += "/"
	}

	_, err := h.s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(h.bucket),
		Key:         aws.String(path + fileName),
		Body:        bytes.NewReader(payload),
		ContentType: aws.String(contentType),
	})

	return err
}
