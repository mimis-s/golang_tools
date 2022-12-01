package dfs

import "fmt"

type DFSHandler interface {
	TryMakeBucket() error
	PutObject(path, fileName string, payload []byte) error
	GetObject(path, fileName string) ([]byte, error)
}

type DFSType = string

var (
	DFSType_Minio       DFSType = "minio"
	DFSType_S3          DFSType = "s3"
	DFSType_CloudStorge DFSType = "cloudStorage"
)

type Config struct {
	Type       DFSType `yaml:"type"` // s3/minio/cloud_storage
	Enable     bool    `yaml:"enable"`
	Bucket     string  `yaml:"bucket"`
	ExpireDays int     `yaml:"expire_days"` // 桶数据过期时间

	// minio,s3参数
	Url   string `yaml:"url"`
	KeyID string `yaml:"key_id"`
	Key   string `yaml:"key"`

	// CloudStorage参数
	Base64Json string `yaml:"base64_json"`
}

var mapHandler = map[string]func(config *Config) (DFSHandler, error){
	DFSType_Minio:       newMinioHandler,
	DFSType_S3:          newS3Handler,
	DFSType_CloudStorge: newCloudStorageHandler,
}

func NewDFSHandler(config *Config) (DFSHandler, error) {
	if !config.Enable {
		return nil, fmt.Errorf("config set diable dfs but now invoke new")
	}
	dfsFun, ok := mapHandler[config.Type]
	if ok {
		return dfsFun(config)
	} else {
		return nil, fmt.Errorf("invalid dfs type:%v", config.Type)
	}
}
