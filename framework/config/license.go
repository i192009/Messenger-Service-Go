package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"gitlab.zixel.cn/go/framework/xutil"
)

// 应用启动时进行证书授权校验
func validateLicenseWhenStart() {
	log.Println("application will start while license validation passed.")
	validateLicense()
	log.Println("license validation passed! now start the application...")
}

// 定时进行证书授权校验
func validateLicenseScheduled() {
	// 1小时校验一次
	ticker := time.NewTicker(time.Hour * 1)
	go func() {
		for range ticker.C {
			log.Println("validate license scheduled...")
			validateLicense()
			log.Println("license scheduled validation passed!")
		}
	}()
}

/**
 * 证书授权校验
 * 1）证书过期时间
 * 2）环境指纹对比
 */
func validateLicense() {
	// 通过license-server获取序列号和环境指纹
	serverUrl := GetString(xutil.LICENSE_SERVER_URL, "")
	res, err := http.Get(serverUrl)
	if err != nil {
		log.Fatal("get fingerInfo from license-server failed", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal("", err)
		}
	}(res.Body)

	response := struct {
		Sn              string `json:"sn"`
		Vc              string `json:"vc"`
		FixedPrivateKey string `json:"fixedPrivateKey"`
	}{}

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		log.Fatal("read fingerInfo failed", err)
	}

	fingerInfo := map[string]string{
		xutil.LICENSE_SN:                response.Sn,
		xutil.LICENSE_VERIFY_CODE:       response.Vc,
		xutil.LICENSE_FIXED_PRIVATE_KEY: response.FixedPrivateKey,
	}

	// 添加序列号到配置中
	sn := fingerInfo[xutil.LICENSE_SN]
	o.Set(xutil.LICENSE_SN, sn)

	fixedPrivateKey := fingerInfo[xutil.LICENSE_FIXED_PRIVATE_KEY]
	o.Set(xutil.LICENSE_FIXED_PRIVATE_KEY, fixedPrivateKey)

	token := GetString(xutil.LICENSE_TOKEN, "")
	noise := GetString(xutil.LICENSE_NOISE, "")
	privateKey := GetString(xutil.LICENSE_PRIVATE_KEY, "")
	if token == "" || noise == "" || privateKey == "" {
		log.Fatal("invalid license config")
	}
	var key []byte
	key, err = base64.StdEncoding.DecodeString(noise)
	if err != nil {
		log.Fatal("invalid license noise", err)
	}
	plainData := xutil.DecryptData([]byte(token), key, privateKey)

	// 获取证书配置
	licenseConfig := make(map[string]any)
	err = json.Unmarshal(plainData, &licenseConfig)
	if err != nil {
		log.Fatal("invalid license config", err)
	}

	// 获取证书过期时间
	expireAt := licenseConfig[xutil.LICENSE_EXPIRE_AT]
	// Getting time location of Shanghai
	var loc *time.Location
	loc, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatal("err LoadLocation", err)
	}
	expireAtStr := expireAt.(string)
	var expire time.Time
	expire, err = time.ParseInLocation(time.RFC3339, expireAtStr, loc)
	if err != nil {
		log.Fatal("err ParseInLocation", err)
	}

	// 证书过期时间校验
	if time.Now().After(expire) {
		log.Fatal("license expired")
	}

	// 从license-server获取的环境指纹
	vc := fingerInfo[xutil.LICENSE_VERIFY_CODE]
	var vcData []byte
	vcData, err = base64.StdEncoding.DecodeString(vc)
	if err != nil {
		log.Fatal("base64 Decode vc failed", err)
	}
	aesKey := sha256.Sum256([]byte(sn))
	var decryptVc []byte
	decryptVc, err = xutil.AesDecrypt(vcData, aesKey[:])
	if err != nil {
		log.Fatal("AesDecrypt vc failed", err)
	}
	envConfig := make(map[string]any)
	err = json.Unmarshal(decryptVc, &envConfig)
	if err != nil {
		log.Fatal("Unmarshal decryptVc failed", err)
	}

	// 应用信息比对
	if envConfig[xutil.LICENSE_APP_ID] != licenseConfig[xutil.LICENSE_APP_ID] {
		log.Fatal("appId not match")
	}
	if envConfig[xutil.LICENSE_APP_VERSION] != licenseConfig[xutil.LICENSE_APP_VERSION] {
		log.Fatal("app version not match")
	}
	if envConfig[xutil.LICENSE_APP_FOR] != licenseConfig[xutil.LICENSE_APP_FOR] {
		log.Fatal("forUser not match")
	}

	// 环境指纹对比
	vh := sha256.Sum256([]byte(vc))
	if base64.StdEncoding.EncodeToString(vh[:]) != licenseConfig[xutil.LICENSE_FINGER] {
		log.Fatal("finger not match")
	}
}
