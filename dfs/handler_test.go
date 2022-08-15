package dfs

import (
	"fmt"
	"testing"
)

func TestDFS(t *testing.T) {

	data := &Config{
		Type:         DFSType_CloudStorge,
		Enable:       true,
		Bucket:       "nox_football_id",
		ExpireDays:   1,
		CloudStorage: &CloudStorageConfig{},
	}

	// 测试谷歌云(这个是谷歌云下载的key文件的base64编码字符串)
	testTxt := "******"

	data.CloudStorage = &CloudStorageConfig{
		testTxt,
	}

	handler, err := NewDFSHandler(data)
	if err != nil {
		panic(err)
	}
	err = handler.TryMakeBucket()
	if err != nil {
		panic(err)
	}
	fmt.Printf("连接成功\n")

	err = handler.PutObject("/test", "123", []byte("woshishui"))
	if err != nil {
		panic(err)
	}

	obj, err := handler.GetObject("/test", "123")
	if err != nil {
		panic(err)
	}
	fmt.Printf("成功下载\n")

	fmt.Printf("%v\n", string(obj))
}
