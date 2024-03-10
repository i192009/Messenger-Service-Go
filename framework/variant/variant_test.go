package variant

import (
	"testing"
)

var Json1 string = `
{
    "debug": true,

    "service": {
        "name": "xdm"
    },

    "mysql": [
        {
            "index": 0,
            "hostname": "localhost",
            "port": 3306,
            "username": "root",
            "password": "root",
            "database": "xdm"
        },
        {
            "index": 1,
            "hostname": "localhost",
            "port": 3306,
            "username": "root",
            "password": "root",
            "database": "xdm"
        }
    ],
	
    "log": {
        "path": "logs",
        "file": "xdm.log",
        "level": "debug"
    }

}`

var Json2 string = `
{
	"service": {
		"name": "kkkk"
	},

	"mysql": [
        {
            "index": 2,
            "hostname": "localhost",
            "port": 3306,
            "username": "root",
            "password": "root",
            "database": "xdm"
        },
        {
            "index": 3,
            "hostname": "localhost",
            "port": 3306,
            "username": "root",
            "password": "root",
            "database": "xdm"
        }
	],

    "mongo": [
        {
            "connectstring": "mongodb://localhost",
            "database": "xdm",
            "default": true
        },
        {
            "connectstring": "mongodb://localhost",
            "database": "xdm-data"
        }
    ],

    "log": {
        "path": "logs",
        "file": "xdm.log",
        "level": "info"
    },

    "web": {
        "listen_addr": "",
        "listen_port": 8742,
        "prefix_path": "/xdm",
        "trusted_proxies": []
    },

    "grpc": {
        "listen_addr": "",
        "listen_port": 9742,
        "connections": {
            "account": "localhost:9400",
            "user": "localhost:9401"
        }
    },

    "obs": {
        "ak": "VKQA5OEEYYYF3XE7JSAD",
        "sk": "EznD8ZfHFTOq9wPbh2vCaalqlt30voUCf5A1ba27",
        "ep": "obs.cn-east-3.myhuaweicloud.com",
        "bucket": {
            "name": "zixel-xdm",
            "folder": "local"
        }
    }
}`

func TestLoad(f *testing.T) {
	v1, err := LoadJson([]byte(Json1))
	if err != nil {
		f.Errorf("load json field %v", err)
	}

	v2, err := LoadJson([]byte(Json2))
	if err != nil {
		f.Errorf("load json field %v", err)
	}

	m := Merge(v1, v2)
	Print(m, "    ", "    ")
}

func TestConvert(f *testing.T) {
	var intValue int = 245
	var strValue string = "245"

	f.Log(New(intValue).ToString())
	f.Log(New(intValue).ToBoolean())
	f.Log(New(strValue).ToInt())
	f.Log(New(strValue).ToDecimal())
}
