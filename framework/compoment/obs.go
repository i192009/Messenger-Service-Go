package compoment

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"gitlab.zixel.cn/go/framework/logger"
)

var log = logger.Get()
var cli *obs.ObsClient

type OBS struct {
	Bucket_UL string
	Bucket_DL string

	obsClient *obs.ObsClient
}

func NewOBS(AK string, SK string, EP string, Bucket string) OBS {
	var err error
	cli, err = obs.New(AK, SK, EP)
	if err != nil {
		log.Errorf("AK = %s, SK = %s, EP = %s, Bucket = %s, err = %s", AK, SK, EP, Bucket, err.Error())
		panic("obs client initialized failed.")
	}

	var o = OBS{
		obsClient: cli,
		Bucket_UL: Bucket,
		Bucket_DL: Bucket,
	}

	return o
}

func (o *OBS) SetBucket(DL, UL string) {
	o.Bucket_DL = DL
	o.Bucket_UL = UL
}

func (o *OBS) IsFileExists(objectKey string) bool {
	metadata := o.GetMetadata(objectKey)
	if _, ok := metadata["Error"]; ok {
		return false
	}

	return true
}

func (o *OBS) GetMetadata(objectKey string) (metadata map[string]any) {
	input := &obs.GetObjectMetadataInput{}
	input.Bucket = o.Bucket_DL
	input.Key = objectKey
	output, err := o.obsClient.GetObjectMetadata(input)
	metadata = make(map[string]any)
	if err == nil {
		log.Debugf("RequestId:%s, StorageClass:%s, ETag:%s, ContentType:%s, ContentLength:%d, LastModified:%s",
			output.RequestId,
			output.StorageClass,
			output.ETag,
			output.ContentType,
			output.ContentLength,
			output.LastModified,
		)

		metadata["RequestId"] = output.RequestId
		metadata["StorageClass"] = output.StorageClass
		metadata["ETag"] = output.ETag
		metadata["ContentType"] = output.ContentType
		metadata["ContentLength"] = output.ContentLength
		metadata["LastModified"] = output.LastModified
	} else {
		metadata["Error"] = err
		if obsError, ok := err.(obs.ObsError); ok {
			metadata["StatusCode"] = obsError.StatusCode
		}
	}

	return
}

func (o *OBS) GetUploadUrl(objectKey string, expires time.Duration, headers map[string]string) (signedUrl string, err error) {
	// 生成上传对象的带授权信息的URL
	putObjectInput := &obs.CreateSignedUrlInput{}
	putObjectInput.Method = obs.HttpMethodPut
	putObjectInput.Bucket = o.Bucket_UL
	putObjectInput.Key = objectKey
	putObjectInput.Expires = int(expires.Seconds())
	putObjectInput.Headers = headers

	output, err := cli.CreateSignedUrl(putObjectInput)
	if err == nil {
		log.Info("SignedUrl: ", output.SignedUrl)
	} else {
		log.Error(err)
	}

	return output.SignedUrl, err
}

func (o *OBS) GetDownloadUrl(objectKey string, expires time.Duration) (signedUrl string, err error) {
	// 生成上传对象的带授权信息的URL
	putObjectInput := &obs.CreateSignedUrlInput{}
	putObjectInput.Method = obs.HttpMethodGet
	putObjectInput.Bucket = o.Bucket_DL
	putObjectInput.Key = objectKey
	putObjectInput.Expires = int(expires.Seconds())

	output, err := cli.CreateSignedUrl(putObjectInput)
	if err == nil {
		log.Info("SignedUrl: ", output.SignedUrl)
	} else {
		log.Error(err)
	}

	return output.SignedUrl, err
}

func (o *OBS) UploadFile(localPath string, remotePath string, contentType string) error {
	input := &obs.UploadFileInput{
		ObjectOperationInput: obs.ObjectOperationInput{
			Bucket: o.Bucket_UL,
			Key:    remotePath,
		},
		// 上传的MIME类型
		ContentType: contentType,
		// 待上传的本地文件路径，需要指定到具体的文件名
		UploadFile: localPath,
		// 开启断点续传模式
		EnableCheckpoint: true,
		// 指定分段大小为9MB
		PartSize: 8 * 1024 * 1024,
		// 指定分段上传时的最大并发数
		TaskNum: 16,
	}

	output, err := cli.UploadFile(input)
	if err != nil {
		if obsError, ok := err.(obs.ObsError); ok {
			log.Debug("Code: ", obsError.Code, " Message: ", obsError.Message)
		}

		return err
	}

	log.Debug("RequestId: ", output.RequestId, " ETag: ", output.ETag)
	return nil
}

func (o *OBS) UploadObject(key string, stream io.Reader, ContentType string) error {
	input := &obs.PutObjectInput{}
	input.Bucket = o.Bucket_UL
	input.Key = key
	input.Body = stream
	input.ContentType = ContentType

	output, err := cli.PutObject(input)

	if err != nil {
		if obsError, ok := err.(obs.ObsError); ok {
			fmt.Println(obsError.Code)
			fmt.Println(obsError.Message)
		} else {
			fmt.Println(err)
		}
		return err
	}

	log.Info("RequestId: ", output.RequestId)
	log.Info("ETag:", output.ETag, "StorageClass:", output.StorageClass)

	return nil
}

func (o *OBS) TransferFile(remoteUrl string, objectKey string, fileSize *int64) error {
	resp, err := http.Get(remoteUrl)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("response status code")
	}

	defer resp.Body.Close()

	if err := o.UploadObject(objectKey, resp.Body, resp.Header.Get("Content-Type")); err != nil {
		return err
	}

	if fileSize != nil {
		*fileSize, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *OBS) CopyFile(SourceKey string, TargetKey string) error {
	input := &obs.CopyObjectInput{}
	input.Bucket = o.Bucket_UL
	input.Key = TargetKey
	input.CopySourceBucket = o.Bucket_DL
	input.CopySourceKey = SourceKey

	output, err := cli.CopyObject(input)
	if err != nil {
		if obsError, ok := err.(obs.ObsError); ok {
			log.Error(obsError.Code)
			log.Error(obsError.Message)
		}

		return err
	}

	log.Info("RequestId ", output.RequestId, " ETag: ", output.ETag)
	return nil
}

func (o *OBS) RemoveFile(ObjectKey string) error {
	input := &obs.DeleteObjectInput{}
	input.Bucket = o.Bucket_UL
	input.Key = ObjectKey

	output, err := o.obsClient.DeleteObject(input)
	if err != nil {
		if obsError, ok := err.(obs.ObsError); ok {
			log.Error(obsError.Code)
			log.Error(obsError.Message)
		}

		return err
	}

	log.Info("RequestId ", output.RequestId)
	return nil
}

func (o *OBS) DownloadFile(remotePath string, localPath string) error {
	input := &obs.DownloadFileInput{}
	// 桶名
	input.Bucket = o.Bucket_DL
	// 对象建
	input.Key = remotePath
	// localfile为下载对象的本地文件全路径
	input.DownloadFile = localPath
	// 开启断点续传模式
	input.EnableCheckpoint = true
	// 指定分段大小为9MB
	input.PartSize = 8 * 1024 * 1024
	// 指定分段下载时的最大并发数
	input.TaskNum = 16
	// 开始下载
	output, err := cli.DownloadFile(input)
	if err == nil {
		log.Debug("RequestId: ", output.RequestId, " filename: ", localPath)
	} else if obsError, ok := err.(obs.ObsError); ok {
		log.Debug("Code: ", obsError.Code, " Message: ", obsError.Message)
		return err
	}

	return nil
}

type FileInfo_t struct {
	RelativePath string
	Size         int64
	ETag         string
	LastModify   time.Time
	Status       int
}

var (
	stNormal  int = 1
	stFailed  int = 2
	stSuccess int = 3
)

func (o *OBS) ListFiles(uri string) []FileInfo_t {
	files := make([]FileInfo_t, 0, 100)

	input := &obs.ListObjectsInput{}
	input.Bucket = o.Bucket_DL
	input.Prefix = uri

	for {
		res, err := cli.ListObjects(input)
		if err != nil {
			break
		}

		/// 将文件保存到Job中
		for _, obj := range res.Contents {
			file := FileInfo_t{
				RelativePath: obj.Key[len(uri):],
				Size:         obj.Size,
				ETag:         obj.ETag,
				LastModify:   obj.LastModified,
				Status:       stNormal,
			}

			files = append(files, file)
		}

		if !res.IsTruncated {
			break
		}

		input.Marker = res.NextMarker
	}

	return files
}

func (o *OBS) DownloadFiles(uri string, save string) (files []FileInfo_t, err error) {
	if len(save) < 1 {
		log.Errorf(err.Error())
		err = fmt.Errorf("%s", "Save is not a directory.")
	}

	files = o.ListFiles(uri)
	for _, obj := range files {
		savepath := filepath.Dir(filepath.Join(save, obj.RelativePath))
		// 创建保存目录
		os.MkdirAll(savepath, 0666)

		if len(obj.RelativePath) == 0 {
			continue
		}

		if err = o.DownloadFile(filepath.Join(uri, obj.RelativePath), filepath.Join(save, obj.RelativePath)); err != nil {
			log.Error(err)
			obj.Status = stFailed
		}
	}

	return
}

func (o *OBS) UploadFiles(where string, uri string) (files []FileInfo_t, err error) {
	if len(where) <= 0 {
		err = nil
		return
	}

	stat, err := os.Stat(where)

	if os.IsNotExist(err) {
		return
	}

	if err != nil || !stat.IsDir() {
		return
	}

	filepath.Walk(where, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Debug("get file [", path, "] information failed.")
			return err
		}

		if info.IsDir() {
			log.Debug("skip directory.", path)
			return nil
		}

		relative, err := filepath.Rel(where, path)
		if err != nil {
			log.Error(err)
			return nil
		}

		ObjectKey := filepath.Join(uri, relative)
		ObjectKey = filepath.ToSlash(ObjectKey)
		log.Debug("upload file to ", ObjectKey)

		if err := o.UploadFile(path, ObjectKey, mime.TypeByExtension(filepath.Ext(relative))); err != nil {
			log.Error(err)
			return nil
		}

		files = append(files, FileInfo_t{
			RelativePath: relative,
			Size:         info.Size(),
			LastModify:   info.ModTime(),
		})

		log.Debug("uploaded file ", path, " to ", ObjectKey)
		return nil
	})

	return
}
