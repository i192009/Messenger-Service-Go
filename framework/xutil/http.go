package xutil

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"gitlab.zixel.cn/go/framework/logger"
)

var (
	log         = logger.Get()
	serverIp, _ = getLocalIP()
)

type HttpClient struct {
	client *http.Client
}

type RemoteFileInfo struct {
	FileName   string
	FileSize   int64
	ModifyTime time.Time
	CreateTime time.Time
}

func GetRemoteFileinfo(downloadUrl string, info *RemoteFileInfo) error {
	request, err := http.NewRequest("HEAD", downloadUrl, nil)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return errors.New("response status code error")
	}

	o, err := url.Parse(downloadUrl)
	if err != nil {
		return err
	}

	info.FileName = filepath.Base(o.Path)
	info.FileSize, err = strconv.ParseInt(response.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}

	info.ModifyTime, err = time.Parse(response.Header.Get("Last-Modified"), time.RFC1123)
	if err != nil {
		return err
	}

	info.CreateTime, err = time.Parse(response.Header.Get("Date"), time.RFC1123)
	if err != nil {
		return err
	}

	return nil
}

func DownloadFile(downloadUrl string, saveTo string) error {

	var info RemoteFileInfo
	if err := GetRemoteFileinfo(downloadUrl, &info); err != nil {
		return err
	}

	resp, err := http.Get(downloadUrl)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	out, err := os.Create(saveTo)
	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func DirectInvoke(host string, method string, reqUri string, reqBody io.Reader, header http.Header) (res []byte, rpnCode int, err error) {
	req, err := http.NewRequestWithContext(context.Background(), strings.ToUpper(method),
		fmt.Sprintf("http://%v%v", host, reqUri), reqBody)
	req.Header = header
	if err != nil {
		return
	}
	var rpn *http.Response
	rpn, err = http.DefaultClient.Do(req)
	if err != nil {
		err = errors.New(fmt.Sprintf("http invoke failed,error=%v", err))
		return
	}
	defer rpn.Body.Close()

	res, err = io.ReadAll(rpn.Body)
	if err != nil {
		err = errors.New(fmt.Sprintf("read http response failed,error=%v", err))
		return
	}

	log.Debugf("http invoke success,req=%v,rpn=%v", *req, *rpn)

	rpnCode = rpn.StatusCode
	return
}

type loggingRoundTripper struct {
	pro http.RoundTripper
}

func (lrt *loggingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t := time.Now()
	counter := request.Context().Value("counter").(*atomic.Int64)
	counter.Add(1)

	zixelInvokerLevel := request.Header.Get("Zixel-Log-InvokerLevel")
	zixelResponseLevel := request.Header.Get("Zixel-Log-ResponseLevel")
	zixelRequestId := request.Header.Get("Zixel-Log-RequestId")
	zixelLogProtocol := "http-" + request.Method
	zixelLogRequestIgnored := request.Header.Get("Zixel-Log-RequestIgnored")
	zixelLogResponseIgnored := request.Header.Get("Zixel-Log-ResponseIgnored")

	callerLevel := zixelInvokerLevel
	responseLevel := zixelInvokerLevel + "." + fmt.Sprintf("%02d", counter.Load())
	zixelInvokerLevel = responseLevel
	zixelResponseLevel = responseLevel

	var params string
	var body string
	if zixelLogRequestIgnored != "true" {
		// Read the request body
		reqBodyBytes, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
		body = string(reqBodyBytes)
		params = request.URL.Query().Encode()
		// Convert request to base64
	}

	request.Header.Set("Zixel-Log-InvokerLevel", zixelInvokerLevel)
	request.Header.Set("Zixel-Log-ResponseLevel", zixelResponseLevel)
	request.Header.Set("Zixel-Log-RequestId", zixelRequestId)
	request.Header.Set("Zixel-Log-Protocol", zixelLogProtocol)
	request.Header.Set("Zixel-Log-RequestIgnored", zixelLogRequestIgnored)
	request.Header.Set("Zixel-Log-ResponseIgnored", zixelLogResponseIgnored)

	// Send the request
	response, err := lrt.pro.RoundTrip(request)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Zixel-Log-InvokerLevel", callerLevel)
	request.Header.Set("Zixel-Log-ResponseLevel", responseLevel)

	var resp []byte
	if zixelLogResponseIgnored != "true" {
		// Read the response body
		respBodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		response.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes)) // Reset the response body to its original state
		resp = respBodyBytes

	}

	latency := float64(time.Since(t)) / float64(time.Millisecond)

	logLine := fmt.Sprintf("[trace-log]%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%d|%s|%s\n",
		zixelRequestId,                      // RequestId
		t.Format("2006-01-02 15:04:05.000"), // Timestamp
		"http",                              // Caller service name
		request.Method,                      // Calling method
		request.URL.Path,                    // Calling method name
		fmt.Sprintf("{\"params\":\"%s\",\"body\":%s}", params, body), // Caller params
		request.RemoteAddr,                   // Caller IP
		callerLevel,                          // Caller level number
		request.URL.Hostname(),               // Responder service name
		request.Method,                       // Responder method
		string(resp),                         // Return parameters
		serverIp,                             // Responder IP
		responseLevel,                        // Responder level number
		response.StatusCode,                  // System return code
		http.StatusText(response.StatusCode), // System return message
		fmt.Sprintf("%.2fms", latency),       // Response time
	)

	fmt.Print(logLine)

	return response, nil
}

func NewLoggingClient() *http.Client {
	return &http.Client{
		Transport: &loggingRoundTripper{
			pro: http.DefaultTransport,
		},
	}
}

func GetHttpClient() *HttpClient {
	return &HttpClient{
		client: NewLoggingClient(),
	}
}

func (httpClient *HttpClient) Do(c *gin.Context, request *http.Request) (*http.Response, error) {
	for k, v := range c.Request.Header {
		request.Header[k] = v
	}
	request = request.WithContext(c.Request.Context())
	resp, err := httpClient.client.Do(request)
	return resp, err
}

func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}
