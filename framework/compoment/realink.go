package compoment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.zixel.cn/go/framework/xutil"
)

// 启动修复断开或冗余的曲面细分
type RepairMesh_t struct {
	// 连接容差
	Tolerance float64 `json:"tolerance,omitempty"`
	//在修复过程结束时，裂纹导致非流形边缘
	CrackNonManifold bool `json:"crackNonManifold,omitempty"`
	//如果为真，则重新定位模型
	Orient bool `json:"orient,omitempty"`
}

type Decimate_t struct { //通过移除一些顶点来减少多边形数量
	//曲面顶点与简化曲面之间的最大距离
	SurfacicTolerance float64 `json:"surfacicTolerance,omitempty"`
	//直线顶点与简化直线之间的最大距离
	LineicTolerance float64 `json:"lineicTolerance,omitempty"`
	//原始法线与在简化表面上插值的法线之间的最大夹角
	NormalTolerance float64 `json:"normalTolerance,omitempty"`
	//在UV空间中，原始texcord与那些在简化表面上插值的最大距离
	TexCoordTolerance float64 `json:"texCoordTolerance,omitempty"`
	//如果为 True，则在很小的面积出不处理法线和/或 texcoord 公差的约束（根据 surfacicTolerance）
	ReleaseConstraintOnSmallArea float64 `json:"releaseConstraintOnSmallArea,omitempty"`
}

// 修复CAD形状，组装面，删除重复面，优化循环和修复拓扑
type RepairCAD_t struct {
	Tolerance float64 `json:"tolerance,omitempty"`
	Orient    bool    `json:"orient,omitempty"`
}

// CAD的每个给定的部件做一个曲面细分
type Tessellate_t struct {
	//几何图形和细分曲面之间的最大距离
	MaxSag float64 `json:"maxSag,omitempty"`
	//元素最大长度
	MaxLength int `json:"maxLength,omitempty"`
	//两个相邻元素法线之间的最大夹角
	MaxAngle int `json:"maxAngle,omitempty"`
	//如果为真，则生成法线
	CreateNormals bool `json:"createNormals,omitempty"`
	//选择纹理坐标生成模式 NoUV (0)  FastUV (1)  UniformUV (2)
	UvMode int `json:"uvMode,omitempty"`
	//生成纹理坐标的UV通道
	UvChannel int `json:"uvChannel,omitempty"`
	//UV坐标空间中UV岛之间的UV填充(0-1之间)。
	UvPadding int `json:"uvPadding,omitempty"`
	//如果为真，则会生成切线
	CreateTangents bool `json:"createTangents,omitempty"`
	//如果为真，将为每个Patch边界创建空闲边
	CreateFreeEdges bool `json:"createFreeEdges,omitempty" default:"true"`
	//如果为真，则将为Back to BRep或Retessellate保留BRep形状
	KeepBRepShape bool `json:"keepBRepShape,omitempty"`
	//如果为真，已经做了曲面细分的部分将重新细分
	OverrideExistingTessellation bool `json:"overrideExistingTessellation,omitempty"`
}

type OptimizeFuns_t struct {
	RepairMesh RepairMesh_t `json:"repairMesh,omitempty"`
	Decimate   Decimate_t   `json:"decimate,omitempty"`
	RepairCAD  RepairCAD_t  `json:"repairCAD,omitempty"`
	Tessellate Tessellate_t `json:"tessellate,omitempty"`
}

type NodeMerges_t struct {
	Name            string `json:"name,omitempty"`            //需要合并的节点名称
	CaseSensitive   bool   `json:"caseSensitive,omitempty"`   //大小写敏感
	MatchWholeWorld bool   `json:"matchWholeWorld,omitempty"` //全字匹配
	RegExp          bool   `json:"regExp,omitempty"`          //启用正则表达式
}

type NodeDecimate_t struct {
	TargetStrategy    []interface{} `json:"TargetStrategy,omitempty"`    /// 减面策略：指定面数["triangleCount":10000],按比例["ratio",50.000000]
	BoundaryWeight    float32       `json:"boundaryWeight,omitempty"`    /// 1.000000,//边界权重
	NormalWeight      float32       `json:"normalWeight,omitempty"`      /// 1.000000,//法向量权重
	UVWeight          float32       `json:"UVWeight,omitempty"`          /// 1.000000,//uv 权重
	SharpNormalWeight float32       `json:"sharpNormalWeight,omitempty"` /// 1.000000,
	UVSeamWeight      float32       `json:"UVSeamWeight,omitempty"`      /// 10.000000,//uv 接缝权重
	ForbidUVFoldovers bool          `json:"forbidUVFoldovers,omitempty"` /// True,//禁止UV重叠
	ProtectTopology   bool          `json:"protectTopology,omitempty"`   /// True//保证拓扑结构
}

type NodeDecimates_t struct {
	NodeMerge    NodeMerges_t   `json:"nodeMerge,omitempty"`
	NodeDecimate NodeDecimate_t `json:"nodeDecimate,omitempty"`
}

type ProcessMesh_t struct {
	NodeMerges    []NodeMerges_t    `json:"nodeMerges,omitempty"`
	NodeDecimates []NodeDecimates_t `json:"nodeDecimates,omitempty"`
}

type FileConfig_t struct {
	Name      string `json:"name,omitempty"`
	Transform bool   `json:"transform,omitempty"` //是否需要文件转换
	FileSize  int64  `json:"fileSize,omitempty"`  //以m 为单位的文件大小
	RemoteUrl string `json:"remoteUrl,omitempty"`
	TargetUrl string `json:"targetUrl,omitempty"` //不转换可为空
}

type NewJobReq_v2_t struct {
	Bucket_DL     string         `json:"bucketDownload,omitempty"`          /// 待转换文件所在的桶
	Bucket_UL     string         `json:"bucketUpload,omitempty"`            /// 文件转换完成后上传的桶
	FolderConfig  []FileConfig_t `json:"folderConfig" validate:"required"`  /// 文件列表
	Optimize      OptimizeFuns_t `json:"optimize,omitempty"`                /// 优化参数
	ProcessMesh   ProcessMesh_t  `json:"processMesh,omitempty"`             /// 减面参数
	Scale         float32        `json:"scale,omitempty"`                   /// 用户单位到毫米的转化系数，比如用户的模型单位是“米”，这个地方就填上1000.
	TargetFormats []string       `json:"targetFormats" validate:"required"` /// 转换的目标格式
	UseHoops      bool           `json:"useHoops"`                          /// 是否使用Hoops，默认使用Pixyz
	Marks         []string       `json:"marks"`                             /// 标记数据环境  dev 开发环境  release是正式环境
}

type NewJobRpn_t struct {
	JobId string `json:"jobId" validate:"required"`
}

type ConvertExtern_t struct {
	Upload      *string
	Optimize    *OptimizeFuns_t
	ProcessMesh *ProcessMesh_t
	UseHoops    bool
	Marks       []string
}

type ConvertJob struct {
	RealinkServiceAddr string
	OBS_EndPoint       string
}

func NewConvertJob(serviceAddr string, obsEndPoint string) ConvertJob {
	return ConvertJob{
		RealinkServiceAddr: serviceAddr,
		OBS_EndPoint:       obsEndPoint,
	}
}

func (c ConvertJob) PostConvertJob(bucket string, sourceKey string, targetKey string, originName string, targetFormat []string, extern *ConvertExtern_t) (jobId string, err error) {
	var fileInfo xutil.RemoteFileInfo
	err = xutil.GetRemoteFileinfo(fmt.Sprintf("https://%s.%s/%s", bucket, c.OBS_EndPoint, sourceKey), &fileInfo)
	if err != nil {
		return "", err
	}

	req := NewJobReq_v2_t{
		Bucket_DL: bucket,
		Bucket_UL: bucket,
		FolderConfig: []FileConfig_t{
			{
				Name:      originName,
				Transform: true,
				FileSize:  fileInfo.FileSize,
				RemoteUrl: sourceKey,
				TargetUrl: targetKey,
			},
		},
		TargetFormats: targetFormat,
	}

	if extern != nil {
		if extern.Upload != nil {
			req.Bucket_UL = *extern.Upload
		}

		if extern.Optimize != nil {
			req.Optimize = *extern.Optimize
		}

		if extern.ProcessMesh != nil {
			req.ProcessMesh = *extern.ProcessMesh
		}

		req.UseHoops = extern.UseHoops
		req.Marks = extern.Marks
	}

	remoteUrl := c.RealinkServiceAddr + "/convert/v2/model/newJob"

	jbuf, err := json.Marshal(req)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	request, err := http.NewRequest("POST", remoteUrl, bytes.NewReader(jbuf))
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	rpn, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	if rpn.StatusCode != 200 {
		log.Error("response status code ", rpn.StatusCode, " message: ", rpn.Status)
		return "", err
	}

	body, err := io.ReadAll(rpn.Body)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	var resp NewJobRpn_t
	err = json.Unmarshal(body, &resp)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	return resp.JobId, nil
}

type GetJobDetailRpn_t struct {
	JobId      string                 `json:"jobId" bson:"jobId"`
	Status     int32                  `json:"status" bson:"status"`
	Total      int32                  `json:"total" bson:"total"`
	Processed  int32                  `json:"processed" bson:"processed"`
	Progress   int32                  `json:"progress" bson:"progress"`
	UpdateTime time.Time              `json:"updateTime" bson:"updateTime"`
	Files      []string               `json:"files,omitempty" bson:"files"`
	Result     map[string]interface{} `json:"result,omitempty" bson:"result"`
}

func (c ConvertJob) GetConvertJobProgress(jobId string) (status int32, progress int32, err error) {
	response, err := http.Get(c.RealinkServiceAddr + "/convert/model/jobDetail/" + jobId)
	if err != nil {
		log.Error(err.Error())
		return 0, 0, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error(err.Error())
		return 0, 0, err
	}

	var resp GetJobDetailRpn_t
	err = json.Unmarshal(body, &resp)
	if err != nil {
		log.Error(err.Error())
		return 0, 0, err
	}

	return resp.Status, resp.Progress, nil
}

func (c ConvertJob) GetConvertJobFiles(jobId string) (status int32, results []string, err error) {
	response, err := http.Get(c.RealinkServiceAddr + "/convert/model/jobDetail/" + jobId)
	if err != nil {
		log.Error(err.Error())
		return 0, nil, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error(err.Error())
		return 0, nil, err
	}

	var resp GetJobDetailRpn_t
	err = json.Unmarshal(body, &resp)
	if err != nil {
		log.Error(err.Error())
		return 0, nil, err
	}

	return resp.Status, resp.Files, nil
}
