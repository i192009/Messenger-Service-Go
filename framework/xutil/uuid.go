package xutil

import (
	"fmt"
	"time"

	"github.com/sony/sonyflake"
)

var snowflakeSetting = sonyflake.Settings{
	StartTime:      time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	MachineID:      nil,
	CheckMachineID: nil,
}
var snowflakeIdGenerator = sonyflake.NewSonyflake(snowflakeSetting)

func GetSnowflakeId() (id uint64, err error) {
	return snowflakeIdGenerator.NextID()
}

func GetSnowflakeId64() (id int64, err error) {
	tmp, err := snowflakeIdGenerator.NextID()
	id = int64(tmp)
	return
}

func GetSnowflakeString() (id string, err error) {
	tmp, err := snowflakeIdGenerator.NextID()
	id = fmt.Sprintf("%v", tmp)
	return
}
