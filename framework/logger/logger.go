package logger

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/sony/sonyflake"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

var (
	threadIDCounter  int
	threadIDs        = make(map[uint64]string)
	mutex            sync.Mutex
	log              *logrus.Logger
	counter          uint64
	snowflakeSetting = sonyflake.Settings{
		StartTime:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		MachineID:      nil,
		CheckMachineID: nil,
	}
	snowflakeIdGenerator = sonyflake.NewSonyflake(snowflakeSetting)
)

func init() {
	log = logrus.New()
	log.SetReportCaller(true)
	log.SetFormatter(&LogFormater{})
	log.SetOutput(os.Stdout)
}

func Get() *logrus.Logger {
	return log
}

type LogFormater struct{}

func (m *LogFormater) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	timestamp := entry.Time.Format("2006-01-02 15:04:05.000")
	sequenceNumber := atomic.AddUint64(&counter, 1)
	uniqueId, _ := getSnowflakeString()
	hostId, _ := os.Hostname()
	threadId := getThreadID()
	level := getLevel(entry.Level)
	var newLog string
	if len(entry.Message) > 0 && entry.Message[0] == '#' {
		fileName := filepath.Base(entry.Caller.File)
		lineNumber := entry.Caller.Line
		functionName := entry.Caller.Function
		newLog = fmt.Sprintf("[L]%s|%d_%s| %s |%s|%s|%s|%d|%s|%s|%s|\"%s\"| \n",
			timestamp, sequenceNumber, uniqueId, level, hostId, threadId, fileName, lineNumber, functionName, "System", "Nil", entry.Message[1:])
	} else if entry.HasCaller() {
		fileName := filepath.Base(entry.Caller.File)
		lineNumber := entry.Caller.Line
		functionName := entry.Caller.Function
		// Only get the last part of the function name
		if lastIndex := strings.LastIndex(functionName, "."); lastIndex >= 0 {
			functionName = functionName[lastIndex+1:]
		}
		newLog = fmt.Sprintf("[L]%s|%d_%s| %s |%s|%s|%s|%d|%s|%s|%s|\"%s\"| \n",
			timestamp, sequenceNumber, uniqueId, level, hostId, threadId, fileName, lineNumber, functionName, "System", "Nil", entry.Message)
	} else {
		newLog = fmt.Sprintf("[L]%s|%d_%s| %s |%s|%s|%s|%d|%s|%s|%s|\"%s\"| \n",
			timestamp, sequenceNumber, uniqueId, level, hostId, threadId, "Nil", "Nil", "Nil", "System", "Nil", entry.Message[1:])
	}

	b.WriteString(newLog)
	return b.Bytes(), nil
}

func InitLogger(LogLevel string, LogFile string) error {
	var err error
	var logWriter *rotatelogs.RotateLogs

	suffix := filepath.Ext(LogFile)
	prefix := strings.TrimRight(LogFile, suffix)

	logfmt := prefix + ".%Y%m%d%H%M" + suffix
	/// 生成日志文件
	logWriter, err = rotatelogs.New(
		/// 日志文件格式名
		logfmt,
		/// 指向当前日志的链接文件名
		rotatelogs.WithLinkName(LogFile),
		/// 保存的数量，超过数量自动清理，设置为21，每个文件8小时分割一次。
		rotatelogs.WithRotationCount(21),
		/// 分割的时间，每8小时分割一次
		rotatelogs.WithRotationTime(time.Duration(8)*time.Hour),
	)

	if err != nil {
		return errors.New("log file open failed")
	}

	log.SetReportCaller(true)
	log.SetFormatter(&LogFormater{})
	log.SetOutput(io.MultiWriter(logWriter, os.Stdout))

	logLevel, err := logrus.ParseLevel(LogLevel)
	if err != nil {
		log.Error("Log level parse failed. set log level to error")
		logLevel = logrus.ErrorLevel
	}

	log.SetLevel(logLevel)
	return nil
}

func getLevel(level logrus.Level) string {
	switch level {
	case logrus.DebugLevel:
		return "DBG"
	case logrus.InfoLevel:
		return "INF"
	case logrus.WarnLevel:
		return "WAR"
	case logrus.ErrorLevel:
		return "ERR"
	case logrus.FatalLevel:
		return "FAT"
	case logrus.PanicLevel:
		return "PAN"
	default:
		return "UNK"
	}
}

func getGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// Parse the 4707 out of "goroutine 4707 [running]:"
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.ParseUint(idField, 10, 64)
	if err != nil {
		log.Fatalf("cannot get goroutine id: %v", err)
	}
	return id
}

func getThreadID() string {
	gid := getGoroutineID()
	mutex.Lock()
	defer mutex.Unlock()
	if threadID, ok := threadIDs[gid]; ok {
		return threadID
	}
	threadIDCounter++
	threadID := fmt.Sprintf("pool-1-thread-%d", threadIDCounter)
	threadIDs[gid] = threadID
	return threadID
}

func getSnowflakeString() (id string, err error) {
	tmp, err := snowflakeIdGenerator.NextID()
	id = fmt.Sprintf("%v", tmp)
	return
}
