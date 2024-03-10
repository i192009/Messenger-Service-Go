package database

import (
	"database/sql"
	"fmt"
	"github.com/acmestack/gobatis"
	"github.com/acmestack/gobatis/datasource"
	"github.com/acmestack/gobatis/factory"
	"strconv"
	"time"

	"gitlab.zixel.cn/go/framework/config"
	"gitlab.zixel.cn/go/framework/logger"
	"gitlab.zixel.cn/go/framework/variant"
	"go.mongodb.org/mongo-driver/mongo"
)

var log = logger.Get()

// private
var db_mongo_cli *mongo.Client // mongo database connection
var db_mysql_cli []*sql.DB     // mysql database connection

// type alias "FieldList"
type FieldList = []string

func Fields(fields ...string) FieldList {
	return fields
}

// init database
func init() {
	log.Info("initialize database ...")
	if config.Exists("mongo") {
		log.Info("initialize mongo ...")
		mongoConfigs := config.GetArray("mongo")
		for _, info := range mongoConfigs {
			db := variant.New(info)

			v := db.Get("connectstring")
			if !v.IsString() {
				log.Fatal("mongo connectString does not exists")
			}
			connectString := v.ToString()
			log.Info("mongo connectstring = '", connectString, "'")
			v = db.Get("mongo.timeout")
			timeoutSecond := 10.0
			if v.IsDecimal() {
				timeoutSecond = v.ToDecimal()
			}

			if err := ConnectMongo(connectString, time.Second*time.Duration(timeoutSecond)); err != nil {
				log.Fatal(err)
			}

			v = db.Get("database")
			if !v.IsString() {
				log.Fatal("mongo database does not exists")
			}

			mongoDatabase := v.ToString()
			v = db.Get("default")
			isDefault := false
			if v.IsBoolean() {
				isDefault = v.ToBoolean()
			}

			SetDatabase(mongoDatabase, isDefault)
		}
	}

	if config.Exists("mysql") {
		log.Info("initialize mysql ...")

		databases := config.GetArray("mysql")
		db_mysql_cli = make([]*sql.DB, len(databases))

		for _, info := range databases {
			db := variant.New(info)
			val := db.Get("index")
			if !val.IsNumeric() {
				log.Fatal("mysql.index isn't integer value")
			}

			idx := val.ToInt()
			if idx < 0 {
				log.Fatal("mysql.index out of range")
			}

			if val = db.Get("username"); !val.IsString() {
				log.Fatal("mysql.username is not exists")
			}
			username := val.ToString()

			if val = db.Get("password"); !val.IsString() {
				log.Fatal("mysql.password is not exists")
			}
			password := val.ToString()

			if val = db.Get("hostname"); !val.IsString() {
				log.Fatal("mysql.hostname is not exists")
			}
			hostname := val.ToString()

			if val = db.Get("database"); !val.IsString() {
				log.Fatal("mysql.database is not exists")
			}
			dbname := val.ToString()

			if val = db.Get("port"); !val.IsString() && !val.IsNumeric() {
				log.Fatal("mysql.port is not exists")
			}
			port := val.ToString()

			args := ""
			if val = db.Get("args"); val.IsString() {
				args = val.ToString()
			}

			connectString := fmt.Sprintf(
				"%s:%s@tcp(%s)/%s?%s",
				username,
				password,
				hostname+":"+port,
				dbname,
				args,
			)

			var err error
			db_mysql_cli[idx], err = ConnectMysql(connectString)
			if err != nil {
				log.Fatal("connect string = ", connectString, " error = ", err.Error())
			}
		}
	}

	if config.Exists("redis") {
		log.Info("initialize redis ...")

		databases := config.GetObject("redis")
		if databases == nil {
			log.Fatal("get redis config failed!")
			return
		}
		db := variant.New(databases)
		host := make([]string, 0)
		v := db.Get("host")
		if v.IsString() {
			hostStr := v.ToString()
			v = db.Get("port")
			if !v.IsNumeric() {
				log.Fatal("redis port does not exists")
			}
			host = append(host, fmt.Sprintf("%v:%v", hostStr, v.ToInt()))
		} else if v.IsArray() {
			for _, val := range v.ToArray() {
				hostStr, ok := val.(string)
				if !ok {
					log.Fatal("redis host string is not valid")
					return
				}
				host = append(host, hostStr)
			}
		} else {
			log.Fatal("redis host does not exists")
			return
		}
		log.Info("redis host = '", host, "'")
		poolSize := 8
		minIdelSize := 0
		poolTimeout := -1
		pool := v.Get("pool")
		if pool.IsObject() {
			poolObj := variant.New(pool)
			if poolObj.Get("max-active").IsNumeric() {
				poolSize = int(poolObj.Get("max-active").ToInt())
			}
			if poolObj.Get("min-idle").IsNumeric() {
				minIdelSize = int(poolObj.Get("min-idle").ToInt())
			}
			if poolObj.Get("max-wait").IsNumeric() {
				poolTimeout = int(poolObj.Get("max-wait").ToInt())
			}
		}
		v = db.Get("timeout")
		timeout := int64(10000)
		if v.IsNumeric() {
			timeout = v.ToInt()
		}
		v = db.Get("retryInterval")
		retryInterval := int64(10)
		if v.IsNumeric() {
			retryInterval = v.ToInt()
		}
		v = db.Get("password")
		password := ""
		if v.IsString() {
			password = v.ToString()
		}
		v = db.Get("masterName")
		masterName := ""
		if v.IsString() {
			masterName = v.ToString()
		}
		v = db.Get("db")
		dbNum := 0
		if v.IsNumeric() {
			dbNum = int(v.ToInt())
		}
		config := RedisConnConfig{
			Addr:          host,
			Password:      password,
			MasterName:    masterName,
			Db:            dbNum,
			PoolSize:      poolSize,
			MinIdelSize:   minIdelSize,
			PoolTimeout:   time.Millisecond * time.Duration(poolTimeout),
			Timeout:       time.Millisecond * time.Duration(timeout),
			RetryInterval: time.Millisecond * time.Duration(retryInterval),
		}
		if err := ConnectRedis(config); err != nil {
			log.Fatal(err)
			return
		}
	}
}

func GetMongoClient() *mongo.Client {
	return db_mongo_cli
}

func GetMysqlClient(index int) *sql.DB {
	idx := index % len(db_mysql_cli)
	return db_mysql_cli[idx]
}

// Mysqlconnect go-batis连接3
func Mysqlconnect(index int) factory.Factory {
	databases := config.GetArray("mysql")
	db := variant.New(databases[index])
	port := db.Get("port").ToInt()
	strInt64 := strconv.FormatInt(port, 10)
	id16, _ := strconv.Atoi(strInt64)
	return gobatis.NewFactory(
		gobatis.SetMaxConn(100),
		gobatis.SetMaxIdleConn(50),
		gobatis.SetDataSource(&datasource.MysqlDataSource{
			Host:     db.Get("hostname").ToString(),
			Port:     id16,
			DBName:   db.Get("database").ToString(),
			Username: db.Get("username").ToString(),
			Password: db.Get("password").ToString(),
			Charset:  "utf8",
		}))
}
