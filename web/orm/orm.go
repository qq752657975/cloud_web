package orm

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // 用于 MySQL 的驱动
	myLog "github.com/ygb616/web/log"
	"reflect"
	"strings"
	"time"
)

// WebDb 结构体用于封装数据库连接和日志记录器
type WebDb struct {
	db     *sql.DB       // 数据库连接
	logger *myLog.Logger // 日志记录器
	Prefix string        // 表名前缀
}

// MsSession 结构体用于管理数据库会话
type MsSession struct {
	db          *WebDb          // 数据库连接封装对象
	tx          *sql.Tx         // 数据库事务
	beginTx     bool            // 标志是否已开启事务
	tableName   string          // 操作的表名
	fieldName   []string        // 字段名称列表
	placeHolder []string        // 占位符列表
	values      []any           // 字段对应的值
	updateParam strings.Builder // 更新语句的参数构建器
	whereParam  strings.Builder // WHERE 子句的参数构建器
	whereValues []any           // WHERE 子句的值
}

// Open 函数打开数据库连接并返回 WebDb 实例
func Open(driverName string, source string) *WebDb {
	db, err := sql.Open(driverName, source) // 打开数据库连接
	if err != nil {
		panic(err) // 如果连接失败，抛出异常
	}
	// 设置最大空闲连接数为 5
	db.SetMaxIdleConns(5)
	// 设置最大连接数为 100
	db.SetMaxOpenConns(100)
	// 设置连接最大存活时间为 3 分钟
	db.SetConnMaxLifetime(time.Minute * 3)
	// 设置空闲连接最大存活时间为 1 分钟
	db.SetConnMaxIdleTime(time.Minute * 1)

	// 创建 WebDb 实例
	msDb := &WebDb{
		db:     db,
		logger: myLog.Default(),
	}
	// 测试数据库连接是否可用
	err = db.Ping()
	if err != nil {
		panic(err) // 如果测试连接失败，抛出异常
	}
	return msDb // 返回 WebDb 实例
}

// New 方法创建新的 MsSession 实例
func (db *WebDb) New(data any) *MsSession {
	m := &MsSession{
		db: db,
	}
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer")) // 如果 data 不是指针，抛出异常
	}
	tVar := t.Elem()
	if m.tableName == "" {
		// 设置表名为前缀加上结构体名称的小写形式
		m.tableName = m.db.Prefix + strings.ToLower(Name(tVar.Name()))
	}
	return m // 返回 MsSession 实例
}

// Table 方法设置 MsSession 的表名
func (s *MsSession) Table(name string) *MsSession {
	s.tableName = name // 设置表名
	return s
}

// SetMaxIdleConns 设置最大空闲连接数
func (db *WebDb) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(n) // 设置数据库连接的最大空闲连接数
}

// fieldNames 方法使用反射获取结构体的字段名称、标签和值，并构建 SQL 语句
func (s *MsSession) fieldNames(data any) {
	// 使用反射获取 data 的类型和值
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer")) // 如果 data 不是指针，抛出异常
	}
	tVar := t.Elem() // 获取指针指向的元素类型
	vVar := v.Elem() // 获取指针指向的元素值
	if s.tableName == "" {
		// 设置表名为前缀加上结构体名称的小写形式
		s.tableName = s.db.Prefix + strings.ToLower(Name(tVar.Name()))
	}
	// 遍历结构体的字段
	for i := 0; i < tVar.NumField(); i++ {
		fieldName := tVar.Field(i).Name // 获取字段名称
		tag := tVar.Field(i).Tag        // 获取字段标签
		sqlTag := tag.Get("msorm")      // 获取 msorm 标签的值
		if sqlTag == "" {
			// 如果没有标签，使用字段名称的小写形式
			sqlTag = strings.ToLower(Name(fieldName))
		} else {
			// 处理标签中的特殊标记
			if strings.Contains(sqlTag, "auto_increment") {
				// 如果包含 auto_increment 标记，跳过这个字段
				continue
			}
			if strings.Contains(sqlTag, ",") {
				// 如果标签中包含逗号，取逗号前的部分
				sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
			}
		}
		id := vVar.Field(i).Interface() // 获取字段的值
		if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
			// 如果字段名为 id 且值为自动生成的 id，跳过这个字段
			continue
		}
		// 将字段名、占位符和值添加到相应的切片中
		s.fieldName = append(s.fieldName, sqlTag)
		s.placeHolder = append(s.placeHolder, "?")
		s.values = append(s.values, vVar.Field(i).Interface())
	}
}

// IsAutoId 判断 id 是否为自动生成的 id
func IsAutoId(id any) bool {
	t := reflect.TypeOf(id)
	switch t.Kind() {
	case reflect.Int64:
		if id.(int64) <= 0 {
			return true
		}
	case reflect.Int32:
		if id.(int32) <= 0 {
			return true
		}
	case reflect.Int:
		if id.(int) <= 0 {
			return true
		}
	default:
		return false
	}
	return false
}

// Name 将驼峰式命名转换为带下划线的命名
func Name(name string) string {
	var names = name[:]
	lastIndex := 0
	var sb strings.Builder
	for index, value := range names {
		if value >= 65 && value <= 90 {
			// 如果是大写字母
			if index == 0 {
				continue
			}
			// 写入前面的部分和下划线
			sb.WriteString(name[lastIndex:index])
			sb.WriteString("_")
			lastIndex = index
		}
	}
	// 写入最后的部分
	sb.WriteString(name[lastIndex:])
	return sb.String()
}

// Close 关闭数据库连接
func (db *WebDb) Close() error {
	// 调用数据库连接的 Close 方法关闭数据库连接
	return db.db.Close()
}

// Insert 方法用于插入数据到数据库表中
func (s *MsSession) Insert(data any) (int64, int64, error) {
	// 每一个操作是独立的，互不影响的 session
	// 使用反射获取结构体的字段名称、标签和值，并构建 SQL 语句
	s.fieldNames(data)

	// 构建插入语句
	query := fmt.Sprintf(
		"insert into %s (%s) values (%s)",
		s.tableName,                      // 表名
		strings.Join(s.fieldName, ","),   // 字段名称，用逗号分隔
		strings.Join(s.placeHolder, ","), // 占位符，用逗号分隔
	)

	// 记录日志
	s.db.logger.Info(query)

	// 声明 SQL 语句预处理对象和错误变量
	var stmt *sql.Stmt
	var err error

	// 判断是否开启事务
	if s.beginTx {
		// 如果开启了事务，使用事务的预处理
		stmt, err = s.tx.Prepare(query)
	} else {
		// 如果没有开启事务，使用数据库连接的预处理
		stmt, err = s.db.db.Prepare(query)
	}

	// 如果预处理过程中发生错误，返回错误
	if err != nil {
		return -1, -1, err
	}

	// 执行插入操作
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err // 如果执行过程中发生错误，返回错误
	}

	// 获取最后插入的 ID
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err // 如果获取最后插入 ID 过程中发生错误，返回错误
	}

	// 获取受影响的行数
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err // 如果获取受影响行数过程中发生错误，返回错误
	}

	// 返回最后插入的 ID 和受影响的行数，以及 nil 错误表示成功
	return id, affected, nil
}
