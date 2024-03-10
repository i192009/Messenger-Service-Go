package xutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
)

var (
	// 私有化部署license配置key
	LICENSE_CONFIGMAP         = "license-config-conf"
	LICENSE_CONFIGKEY         = "license-config.json"
	LICENSE_SN                = "sn"
	LICENSE_FIXED_PRIVATE_KEY = "fixedPrivateKey"
	LICENSE_PRIVATE_KEY       = "privateKey"
	LICENSE_TOKEN             = "token"
	LICENSE_NOISE             = "noise"
	LICENSE_EXPIRE_AT         = "expireAt"
	LICENSE_FINGER            = "finger"
	LICENSE_VERIFY_CODE       = "vc"
	LICENSE_SERVER_URL        = "serverUrl"
	LICENSE_APP_ID            = "appid"
	LICENSE_APP_VERSION       = "version"
	LICENSE_APP_FOR           = "forUser"
)

func DecryptDataWithFixedPrivateKey(encryptData []byte, fixedPrivateKey string) (plainData []byte) {
	// Check for fixedPrivateKey
	if fixedPrivateKey == "" {
		log.Fatal("missing fixedPrivateKey config")
	}

	// Decode the encrypted data from base64
	var err error
	encryptData, err = base64.StdEncoding.DecodeString(string(encryptData))
	if err != nil {
		log.Fatal("decrypt configMap failed", err)
	}

	plainData, err = rsa_Decrypt(encryptData, fixedPrivateKey)
	if err != nil {
		log.Fatal("decrypt configMap failed", err)
	}
	return
}

func DecryptDataWithSn(encryptData []byte, sn string, privateKey string) (plainData []byte) {
	// 校验序列号配置
	if sn == "" {
		log.Fatal("missing sn config")
	}
	// 以序列号为噪音
	key := sha256.Sum256([]byte(sn))
	plainData = DecryptData(encryptData, key[:], privateKey)
	return
}

func DecryptData(encryptData []byte, key []byte, privateKey string) (plainData []byte) {
	// 校验rsa privateKey
	if privateKey == "" {
		log.Fatal("missing privateKey config")
	}

	// 对每个加密块进行base64解码
	var err error
	encryptData, err = base64.StdEncoding.DecodeString(string(encryptData))
	if err != nil {
		log.Fatal("decrypt configMap failed", err)
	}

	plainData, err = decryptData(privateKey, key[:], encryptData)
	if err != nil {
		log.Fatal("decrypt configMap failed", err)
	}
	// fmt.Printf("plainData=%s\n", string(plainData))
	return
}

func decryptData(privateKey string, randomKey []byte, data []byte) (decryptData []byte, err error) {
	aes_encrypted, err := rsa_Decrypt(data, privateKey)
	if err != nil {
		return nil, err
	}

	decryptData, err = AesDecrypt(aes_encrypted, randomKey)
	if err != nil {
		return nil, err
	}
	return
}

/**
 * RSADecrypt RSA解密
 * 明文长度超过限制时会进行分块加密，返回的是分块加密合并后的结果
 * 其长度等于rsa.privatKey，可据此进行分割解密
 */
func rsa_Decrypt(cipherText []byte, key string) ([]byte, error) {
	buf := bytes.NewBufferString(key)

	//pem解码
	block, _ := pem.Decode(buf.Bytes())
	//X509解码
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// 以privateKey的长度分割密文
	blockSize := privateKey.Size()
	len := len(cipherText)
	start := 0
	plainData := new(bytes.Buffer)
	for start < len {
		end := start + blockSize
		if end > len {
			end = len
		}
		//对密文进行解密
		plainText, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, cipherText[start:end])
		if err != nil {
			return nil, err
		}
		plainData.Write(plainText)
		start = end
	}

	return plainData.Bytes(), nil
}

// AesDecrypt 解密
func AesDecrypt(data []byte, key []byte) (res []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			print(err)
			err = errors.New("decrypt failed")
		}
	}()

	//创建实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//获取块的大小
	blockSize := block.BlockSize()
	//使用cbc
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	//初始化解密数据接收切片
	crypted := make([]byte, len(data))

	//执行解密
	blockMode.CryptBlocks(crypted, data)
	//去除填充
	crypted, err = pkcs7UnPadding(crypted)
	if err != nil {
		return nil, err
	}
	return crypted, nil
}

// pkcs7UnPadding 填充的反向操作
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	return data[:(length - unPadding)], nil
}
