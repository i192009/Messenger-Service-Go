package database

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func ConnectMysql(connectString string) (m *sql.DB, err error) {
	if m, err = sql.Open("mysql", connectString); err != nil {
		return
	}

	if err = m.Ping(); err != nil {
		return
	}

	m.SetConnMaxLifetime(0)
	m.SetMaxIdleConns(20)
	m.SetMaxOpenConns(20)

	return
}

type MysqlResult interface {
	GetValue(row int, col int) any
	GetValueByName(row int, field string) any
	GetRowByIndex(index string, value any) []int
	GetRows() int
	GetCols() int
	GetCol(name string) int
}

type MysqlValue interface {
	IsNull() bool
	ToString() string
	ToInteger() int
	ToInt32() int32
	ToInt64() int64
	ToUnsigned() uint
	ToUInt32() uint32
	ToUInt64() uint64
	ToReal32() float32
	ToReal64() float64
	ToBoolean() bool
	ToTime() time.Time
	ToTimestamp() int64
}

type QueryResult struct {
	Cols []string
	Rows [][]any

	indexes map[string]map[any][]int
}

type FieldValue struct {
	holder any
}

func NewQueryResult(rows *sql.Rows, indexes ...string) (tbl QueryResult, err error) {
	if rows == nil {
		err = sql.ErrNoRows
		return
	}

	defer rows.Close()
	// for rows.Next() {

	// }

	tbl.Cols, err = rows.Columns()
	for rows.Next() {
		// 需要转换为指针接受scan数据
		values := make([]any, len(tbl.Cols))
		valuesP := make([]any, len(values))
		for i := range values {
			valuesP[i] = &values[i]
		}
		if err = rows.Scan(valuesP...); err != nil {
			return
		}

		tbl.Rows = append(tbl.Rows, values)
	}

	return
}

func (t *QueryResult) BuildIndexes(indexes ...string) {
	if len(indexes) > 0 {
		t.indexes = make(map[string]map[any][]int)

		for _, index := range indexes {
			col := t.GetCol(index)
			if col == -1 {
				continue
			}

			// 字段下的IndexMap
			indexMap := make(map[any][]int)
			// 遍历表，填充索引
			for row := 0; row < t.GetRows(); row++ {
				fieldValue := t.GetValue(row, col)
				if set, ok := indexMap[fieldValue]; ok {
					indexMap[fieldValue] = append(set, row)
				} else {
					indexMap[fieldValue] = append(make([]int, 0, 1), row)
				}
			}

			t.indexes[index] = indexMap
		}
	}
}

func (t QueryResult) GetCol(name string) int {
	for i := 0; i < len(t.Cols); i++ {
		if t.Cols[i] == name {
			return i
		}
	}

	return -1
}

func (t QueryResult) GetValue(row int, col int) MysqlValue {
	if row < 0 || row >= len(t.Rows) {
		return nil
	}

	if col < 0 || col >= len(t.Cols) {
		return nil
	}

	Row := t.Rows[row]
	if col >= len(Row) {
		return nil
	}

	return &FieldValue{holder: Row[col]}
}

func (t QueryResult) GetValueByName(row int, name string) MysqlValue {
	return t.GetValue(row, t.GetCol(name))
}

func (t QueryResult) GetRowByIndex(index string, value any) []int {
	if len(t.indexes) == 0 {
		return nil
	}

	if indexMap, ok := t.indexes[index]; !ok {
		return nil
	} else if rowSet, ok := indexMap[value]; !ok {
		return nil
	} else {
		return rowSet
	}
}

func (v FieldValue) IsNull() bool {
	return v.holder == nil
}

func (t QueryResult) GetRows() int {
	return len(t.Rows)
}

func (t QueryResult) GetCols() int {
	return len(t.Cols)
}

func (v FieldValue) ToString() string {
	switch r := v.holder.(type) {
	case int, int8, int16, int32, int64:
		return fmt.Sprint(r)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(r)
	case float32, float64:
		return fmt.Sprint(r)
	case time.Time:
		return r.Format(time.RFC3339)
	case string, []uint8:
		tmpStr := fmt.Sprintf("%s", r)
		return tmpStr
	}

	return ""
}

func (v FieldValue) ToInteger() int {
	switch r := v.holder.(type) {
	case int:
		return r
	case int8:
		return int(r)
	case int16:
		return int(r)
	case int32:
		return int(r)
	case int64:
		return int(r)
	case uint:
		return int(r)
	case uint8:
		return int(r)
	case uint16:
		return int(r)
	case uint32:
		return int(r)
	case uint64:
		return int(r)
	case float32:
		return int(r)
	case float64:
		return int(r)
	case time.Time:
		return int(r.Unix())
	case string, []uint8:
		tmpStr := fmt.Sprintf("%s", r)
		s, err := strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			return 0
		}

		return int(s)
	}

	return 0
}

func (v FieldValue) ToInt32() int32 {
	switch r := v.holder.(type) {
	case int:
		return int32(r)
	case int8:
		return int32(r)
	case int16:
		return int32(r)
	case int32:
		return int32(r)
	case int64:
		return int32(r)
	case uint:
		return int32(r)
	case uint8:
		return int32(r)
	case uint16:
		return int32(r)
	case uint32:
		return int32(r)
	case uint64:
		return int32(r)
	case float32:
		return int32(r)
	case float64:
		return int32(r)
	case time.Time:
		return int32(r.Unix())
	case string, []uint8:
		tmpStr := fmt.Sprintf("%s", r)
		s, err := strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			return 0
		}

		return int32(s)
	}

	return 0
}

func (v FieldValue) ToInt64() int64 {
	switch r := v.holder.(type) {
	case int:
		return int64(r)
	case int8:
		return int64(r)
	case int16:
		return int64(r)
	case int32:
		return int64(r)
	case int64:
		return int64(r)
	case uint:
		return int64(r)
	case uint8:
		return int64(r)
	case uint16:
		return int64(r)
	case uint32:
		return int64(r)
	case uint64:
		return int64(r)
	case float32:
		return int64(r)
	case float64:
		return int64(r)
	case time.Time:
		return int64(r.Unix())
	case string, []uint8:
		tmpStr := fmt.Sprintf("%s", r)
		s, err := strconv.ParseInt(tmpStr, 10, 64)
		if err != nil {
			return 0
		}

		return int64(s)
	}

	return 0
}

func (v FieldValue) ToUnsigned() uint {
	value := reflect.ValueOf(v.holder)
	return uint(value.Uint())
}

func (v FieldValue) ToUInt32() uint32 {
	value := reflect.ValueOf(v.holder)
	return uint32(value.Uint())
}

func (v FieldValue) ToUInt64() uint64 {
	value := reflect.ValueOf(v.holder)
	return value.Uint()
}

func (v FieldValue) ToReal32() float32 {
	value := reflect.ValueOf(v.holder)
	return float32(value.Float())
}

func (v FieldValue) ToReal64() float64 {
	value := reflect.ValueOf(v.holder)
	return value.Float()
}

func (v FieldValue) ToBoolean() bool {
	value := reflect.ValueOf(v.holder)
	return value.Bool()
}

func (v FieldValue) ToTime() time.Time {
	switch r := v.holder.(type) {
	case string, []uint8:
		s := fmt.Sprintf("%s", r)
		DefaultTimeLoc := time.Local
		t, err := time.ParseInLocation("2006-01-02 15:04:05", s, DefaultTimeLoc)
		if err != nil {
			return time.Time{}
		}
		return t
	case int, int32, int64, uint, uint32, uint64:
		return time.Unix(v.ToInt64(), 0)
	}

	return time.Time{}
}

func (v FieldValue) ToTimestamp() int64 {
	switch r := v.holder.(type) {
	case string, []uint8:
		tmpStr := fmt.Sprintf("%s", r)
		t, err := time.Parse(time.RFC3339, tmpStr)
		if err != nil {
			return t.Unix()
		}
		return t.Unix()
	case int, int32, int64, uint, uint32, uint64:
		return v.ToInt64()
	}

	return 0
}

func (v FieldValue) MarshalText() (data []byte, err error) {
	data = []byte(v.ToString())
	return
}

func (v *FieldValue) UnmarshalText(text []byte) (err error) {
	v.holder = string(text)
	return
}
