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

// batchValues 方法用于批量处理数据并提取值
func (s *MsSession) batchValues(data []any) {
	// 初始化 s.values 切片
	s.values = make([]any, 0)

	// 遍历 data 切片中的每一个元素
	for _, v := range data {
		// 使用反射获取当前元素的类型和值
		t := reflect.TypeOf(v)
		v := reflect.ValueOf(v)

		// 如果当前元素不是指针类型，抛出异常
		if t.Kind() != reflect.Pointer {
			panic(errors.New("data must be pointer"))
		}

		// 获取指针指向的元素类型和值
		tVar := t.Elem()
		vVar := v.Elem()

		// 遍历元素的每一个字段
		for i := 0; i < tVar.NumField(); i++ {
			fieldName := tVar.Field(i).Name // 获取字段名称
			tag := tVar.Field(i).Tag        // 获取字段标签
			sqlTag := tag.Get("msorm")      // 获取 msorm 标签的值

			// 如果没有标签，使用字段名称的小写形式
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(fieldName))
			} else {
				// 处理标签中的特殊标记
				if strings.Contains(sqlTag, "auto_increment") {
					// 如果包含 auto_increment 标记，跳过这个字段
					continue
				}
			}

			id := vVar.Field(i).Interface() // 获取字段的值

			// 如果字段名为 id 且值为自动生成的 id，跳过这个字段
			if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
				continue
			}

			// 将字段的值添加到 s.values 切片中
			s.values = append(s.values, vVar.Field(i).Interface())
		}
	}
}

// Close 关闭数据库连接
func (db *WebDb) Close() error {
	// 调用数据库连接的 Close 方法关闭数据库连接
	return db.db.Close()
}

// Where 方法用于添加 WHERE 条件
func (s *MsSession) Where(field string, value any) *MsSession {
	// 生成 WHERE 子句
	if s.whereParam.String() == "" { // 如果 whereParam 为空，则添加 "where"
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)              // 添加字段名
	s.whereParam.WriteString(" = ")              // 添加等号
	s.whereParam.WriteString(" ? ")              // 添加占位符
	s.whereValues = append(s.whereValues, value) // 将值添加到 whereValues 列表中
	return s                                     // 返回当前会话以支持链式调用
}

// Like 方法用于添加 LIKE 条件
func (s *MsSession) Like(field string, value any) *MsSession {
	// 生成 LIKE 子句
	if s.whereParam.String() == "" { // 如果 whereParam 为空，则添加 "where"
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)                               // 添加字段名
	s.whereParam.WriteString(" like ")                            // 添加 LIKE 关键字
	s.whereParam.WriteString(" ? ")                               // 添加占位符
	s.whereValues = append(s.whereValues, "%"+value.(string)+"%") // 将值添加到 whereValues 列表中，并添加通配符 "%"
	return s                                                      // 返回当前会话以支持链式调用
}

// LikeRight 方法用于添加 LIKE 条件，匹配右侧
func (s *MsSession) LikeRight(field string, value any) *MsSession {
	// 生成 LIKE 子句，匹配右侧
	if s.whereParam.String() == "" { // 如果 whereParam 为空，则添加 "where"
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)                           // 添加字段名
	s.whereParam.WriteString(" like ")                        // 添加 LIKE 关键字
	s.whereParam.WriteString(" ? ")                           // 添加占位符
	s.whereValues = append(s.whereValues, value.(string)+"%") // 将值添加到 whereValues 列表中，并添加右侧通配符 "%"
	return s                                                  // 返回当前会话以支持链式调用
}

// LikeLeft 方法用于添加 LIKE 条件，匹配左侧
func (s *MsSession) LikeLeft(field string, value any) *MsSession {
	// 生成 LIKE 子句，匹配左侧
	if s.whereParam.String() == "" { // 如果 whereParam 为空，则添加 "where"
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)                           // 添加字段名
	s.whereParam.WriteString(" like ")                        // 添加 LIKE 关键字
	s.whereParam.WriteString(" ? ")                           // 添加占位符
	s.whereValues = append(s.whereValues, "%"+value.(string)) // 将值添加到 whereValues 列表中，并添加左侧通配符 "%"
	return s                                                  // 返回当前会话以支持链式调用
}

// Group 方法用于添加 GROUP BY 子句
func (s *MsSession) Group(field ...string) *MsSession {
	// 生成 GROUP BY 子句
	s.whereParam.WriteString(" group by ")             // 添加 GROUP BY 关键字
	s.whereParam.WriteString(strings.Join(field, ",")) // 添加字段名，并用逗号分隔
	return s                                           // 返回当前会话以支持链式调用
}

// OrderDesc 方法用于添加 ORDER BY DESC 子句
func (s *MsSession) OrderDesc(field ...string) *MsSession {
	// 生成 ORDER BY DESC 子句
	s.whereParam.WriteString(" order by ")             // 添加 ORDER BY 关键字
	s.whereParam.WriteString(strings.Join(field, ",")) // 添加字段名，并用逗号分隔
	s.whereParam.WriteString(" desc ")                 // 添加 DESC 关键字
	return s                                           // 返回当前会话以支持链式调用
}

// OrderAsc 方法用于添加 ORDER BY ASC 子句
func (s *MsSession) OrderAsc(field ...string) *MsSession {
	// 生成 ORDER BY ASC 子句
	s.whereParam.WriteString(" order by ")             // 添加 ORDER BY 关键字
	s.whereParam.WriteString(strings.Join(field, ",")) // 添加字段名，并用逗号分隔
	s.whereParam.WriteString(" asc ")                  // 添加 ASC 关键字
	return s                                           // 返回当前会话以支持链式调用
}

// Order 方法用于添加 ORDER BY 子句，可以指定多个字段和排序方式
// 示例：Order("aa", "desc", "bb", "asc")
func (s *MsSession) Order(field ...string) *MsSession {
	// 检查字段数量是否为偶数
	if len(field)%2 != 0 { // 如果字段数量不是偶数，则抛出异常
		panic("field num not true")
	}
	s.whereParam.WriteString(" order by ") // 添加 ORDER BY 关键字
	for index, v := range field {          // 遍历字段和排序方式
		s.whereParam.WriteString(v + " ")         // 添加字段名或排序方式
		if index%2 != 0 && index < len(field)-1 { // 在每对字段和排序方式之间添加逗号
			s.whereParam.WriteString(",")
		}
	}
	return s // 返回当前会话以支持链式调用
}

// And 方法用于添加 AND 条件
func (s *MsSession) And() *MsSession {
	s.whereParam.WriteString(" and ") // 添加 AND 关键字
	return s                          // 返回当前会话以支持链式调用
}

// Or 方法用于添加 OR 条件
func (s *MsSession) Or() *MsSession {
	s.whereParam.WriteString(" or ") // 添加 OR 关键字
	return s                         // 返回当前会话以支持链式调用
}

// Count 方法用于获取记录的数量
func (s *MsSession) Count() (int64, error) {
	return s.Aggregate("count", "*") // 调用 Aggregate 方法，使用 "count" 函数和 "*" 字段
}

// Aggregate 方法用于执行聚合函数，如 count、sum、avg 等
func (s *MsSession) Aggregate(funcName string, field string) (int64, error) {
	var fieldSb strings.Builder                                               // 创建字符串构建器，用于构建聚合函数的字段部分
	fieldSb.WriteString(funcName)                                             // 写入聚合函数名
	fieldSb.WriteString("(")                                                  // 写入左括号
	fieldSb.WriteString(field)                                                // 写入字段名
	fieldSb.WriteString(")")                                                  // 写入右括号
	query := fmt.Sprintf("select %s from %s ", fieldSb.String(), s.tableName) // 构建查询语句
	var sb strings.Builder                                                    // 创建字符串构建器，用于构建完整的查询语句
	sb.WriteString(query)                                                     // 写入查询语句的前半部分
	sb.WriteString(s.whereParam.String())                                     // 写入 WHERE 子句
	s.db.logger.Info(sb.String())                                             // 记录生成的查询语句到日志中

	stmt, err := s.db.db.Prepare(sb.String()) // 预处理 SQL 语句
	if err != nil {                           // 如果预处理过程中发生错误
		return 0, err // 返回错误
	}
	row := stmt.QueryRow(s.whereValues...) // 执行查询，获取单行结果
	if row.Err() != nil {                  // 如果查询过程中发生错误
		return 0, err // 返回错误
	}
	var result int64        // 定义变量用于存储查询结果
	err = row.Scan(&result) // 扫描查询结果到 result 变量
	if err != nil {         // 如果扫描过程中发生错误
		return 0, err // 返回错误
	}
	return result, nil // 返回查询结果
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

// InsertBatch 方法用于批量插入数据到数据库表中
func (s *MsSession) InsertBatch(data []any) (int64, int64, error) {
	// 如果数据为空，返回错误
	if len(data) == 0 {
		return -1, -1, errors.New("no data insert")
	}

	// 使用反射获取第一个数据项的字段名称、标签和值，并构建 SQL 语句
	s.fieldNames(data[0])

	// 构建插入语句的前半部分
	query := fmt.Sprintf("insert into %s (%s) values ", s.tableName, strings.Join(s.fieldName, ","))
	var sb strings.Builder
	sb.WriteString(query)

	// 构建插入语句的 values 部分
	for index := range data {
		sb.WriteString("(")
		sb.WriteString(strings.Join(s.placeHolder, ","))
		sb.WriteString(")")
		if index < len(data)-1 {
			sb.WriteString(",") // 如果不是最后一个数据项，添加逗号
		}
	}

	// 使用反射批量处理数据，提取值
	s.batchValues(data)

	// 记录生成的插入语句到日志中
	s.db.logger.Info(sb.String())

	// 声明 SQL 语句预处理对象和错误变量
	var stmt *sql.Stmt
	var err error

	// 判断是否开启事务
	if s.beginTx {
		// 如果开启了事务，使用事务的预处理
		stmt, err = s.tx.Prepare(sb.String())
	} else {
		// 如果没有开启事务，使用数据库连接的预处理
		stmt, err = s.db.db.Prepare(sb.String())
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

// Update 方法用于更新数据库中的记录
func (s *MsSession) Update(data ...any) (int64, int64, error) {
	// 如果参数数量超过2个，返回错误
	if len(data) > 2 {
		return -1, -1, errors.New("param not valid")
	}

	// 如果没有参数，使用已有的 updateParam 和 whereParam 构建更新语句
	if len(data) == 0 {
		// 构建更新语句
		query := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
		var sb strings.Builder
		sb.WriteString(query)                 // 写入更新语句的前半部分
		sb.WriteString(s.whereParam.String()) // 写入 WHERE 子句
		s.db.logger.Info(sb.String())         // 记录生成的更新语句到日志中

		// 预处理 SQL 语句
		var stmt *sql.Stmt
		var err error
		if s.beginTx {
			stmt, err = s.tx.Prepare(sb.String()) // 使用事务的预处理
		} else {
			stmt, err = s.db.db.Prepare(sb.String()) // 使用数据库连接的预处理
		}
		if err != nil {
			return -1, -1, err // 如果预处理过程中发生错误，返回错误
		}

		// 执行更新操作
		s.values = append(s.values, s.whereValues...) // 将 WHERE 子句的值添加到 s.values 中
		r, err := stmt.Exec(s.values...)              // 执行更新操作
		if err != nil {
			return -1, -1, err // 如果执行过程中发生错误，返回错误
		}

		// 获取更新结果
		id, err := r.LastInsertId() // 获取最后插入的 ID
		if err != nil {
			return -1, -1, err // 如果获取最后插入 ID 过程中发生错误，返回错误
		}
		affected, err := r.RowsAffected() // 获取受影响的行数
		if err != nil {
			return -1, -1, err // 如果获取受影响行数过程中发生错误，返回错误
		}
		return id, affected, nil // 返回最后插入的 ID 和受影响的行数，以及 nil 错误表示成功
	}

	single := true // 初始化单字段更新标志
	if len(data) == 2 {
		single = false // 如果参数数量为 2，设置单字段更新标志为 false
	}

	// 构建更新语句的 set 部分
	if !single {
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",") // 如果已有 SET 子句，添加逗号分隔
		}
		s.updateParam.WriteString(data[0].(string)) // 添加字段名
		s.updateParam.WriteString(" = ? ")          // 添加占位符
		s.values = append(s.values, data[1])        // 添加值
	} else {
		updateData := data[0]            // 获取更新对象
		t := reflect.TypeOf(updateData)  // 获取对象类型
		v := reflect.ValueOf(updateData) // 获取对象值
		if t.Kind() != reflect.Pointer {
			panic(errors.New("updateData must be pointer")) // 如果对象不是指针类型，抛出异常
		}
		tVar := t.Elem() // 获取指针指向的元素类型
		vVar := v.Elem() // 获取指针指向的元素值
		for i := 0; i < tVar.NumField(); i++ {
			fieldName := tVar.Field(i).Name // 获取字段名称
			tag := tVar.Field(i).Tag        // 获取字段标签
			sqlTag := tag.Get("msorm")      // 获取 msorm 标签的值
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(fieldName)) // 如果没有标签，使用字段名称的小写形式
			} else {
				if strings.Contains(sqlTag, "auto_increment") {
					// 自增长的主键 id 跳过
					continue
				}
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")] // 处理标签中的逗号
				}
			}
			id := vVar.Field(i).Interface() // 获取字段的值
			if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
				continue // 如果字段名为 id 且值为自动生成的 id，跳过这个字段
			}
			if s.updateParam.String() != "" {
				s.updateParam.WriteString(",") // 如果已有 SET 子句，添加逗号分隔
			}
			s.updateParam.WriteString(sqlTag)                      // 添加字段名
			s.updateParam.WriteString(" = ? ")                     // 添加占位符
			s.values = append(s.values, vVar.Field(i).Interface()) // 添加字段值
		}
	}

	// 构建完整的更新语句
	query := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
	var sb strings.Builder
	sb.WriteString(query)                 // 写入更新语句的前半部分
	sb.WriteString(s.whereParam.String()) // 写入 WHERE 子句
	s.db.logger.Info(sb.String())         // 记录生成的更新语句到日志中

	// 预处理 SQL 语句
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(sb.String()) // 使用事务的预处理
	} else {
		stmt, err = s.db.db.Prepare(sb.String()) // 使用数据库连接的预处理
	}
	if err != nil {
		return -1, -1, err // 如果预处理过程中发生错误，返回错误
	}

	// 执行更新操作
	s.values = append(s.values, s.whereValues...) // 将 WHERE 子句的值添加到 s.values 中
	r, err := stmt.Exec(s.values...)              // 执行更新操作
	if err != nil {
		return -1, -1, err // 如果执行过程中发生错误，返回错误
	}

	// 获取更新结果
	id, err := r.LastInsertId() // 获取最后插入的 ID
	if err != nil {
		return -1, -1, err // 如果获取最后插入 ID 过程中发生错误，返回错误
	}
	affected, err := r.RowsAffected() // 获取受影响的行数
	if err != nil {
		return -1, -1, err // 如果获取受影响行数过程中发生错误，返回错误
	}
	return id, affected, nil // 返回最后插入的 ID 和受影响的行数，以及 nil 错误表示成功
}

// UpdateParam 方法用于设置单个字段的更新参数
func (s *MsSession) UpdateParam(field string, value any) *MsSession {
	// 如果 updateParam 不是空字符串，则添加逗号分隔符
	if s.updateParam.String() != "" {
		s.updateParam.WriteString(",")
	}
	// 添加字段名
	s.updateParam.WriteString(field)
	// 添加等号和占位符
	s.updateParam.WriteString(" = ? ")
	// 将值添加到 values 列表中
	s.values = append(s.values, value)
	// 返回 MsSession 实例，以支持链式调用
	return s
}

// UpdateMap 方法用于设置多个字段的更新参数
func (s *MsSession) UpdateMap(data map[string]any) *MsSession {
	// 遍历 map 中的每一个键值对
	for k, v := range data {
		// 如果 updateParam 不是空字符串，则添加逗号分隔符
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		// 添加字段名
		s.updateParam.WriteString(k)
		// 添加等号和占位符
		s.updateParam.WriteString(" = ? ")
		// 将值添加到 values 列表中
		s.values = append(s.values, v)
	}
	// 返回 MsSession 实例，以支持链式调用
	return s
}

// SelectOne 方法用于从数据库中选择一条记录，并将结果映射到 data 结构体中
func (s *MsSession) SelectOne(data any, fields ...string) error {
	t := reflect.TypeOf(data)        // 获取 data 的类型
	if t.Kind() != reflect.Pointer { // 检查 data 是否为指针类型
		return errors.New("data must be pointer") // 如果 data 不是指针类型，返回错误
	}

	// 构建查询字段
	fieldStr := "*"      // 默认查询所有字段
	if len(fields) > 0 { // 如果指定了字段
		fieldStr = strings.Join(fields, ",") // 使用指定的字段
	}

	// 构建查询语句
	query := fmt.Sprintf("select %s from %s ", fieldStr, s.tableName) // 构建查询语句
	var sb strings.Builder                                            // 创建字符串构建器
	sb.WriteString(query)                                             // 写入查询语句的前半部分
	sb.WriteString(s.whereParam.String())                             // 写入 WHERE 子句
	s.db.logger.Info(sb.String())                                     // 记录生成的查询语句到日志中

	// 预处理 SQL 语句
	stmt, err := s.db.db.Prepare(sb.String()) // 预处理 SQL 语句
	if err != nil {                           // 如果预处理过程中发生错误
		return err // 返回错误
	}

	// 执行查询
	rows, err := stmt.Query(s.whereValues...) // 执行查询
	if err != nil {                           // 如果查询过程中发生错误
		return err // 返回错误
	}

	// 获取查询结果的列名
	columns, err := rows.Columns() // 获取查询结果的列名
	if err != nil {                // 如果获取列名过程中发生错误
		return err // 返回错误
	}

	// 创建用于存储查询结果的切片
	values := make([]any, len(columns))    // 创建存储查询结果的切片
	fieldScan := make([]any, len(columns)) // 创建存储查询字段扫描结果的切片
	for i := range fieldScan {             // 遍历 fieldScan
		fieldScan[i] = &values[i] // 将 values 中的每个元素的地址赋给 fieldScan
	}

	// 如果有查询结果，处理第一条记录
	if rows.Next() { // 如果有查询结果
		err := rows.Scan(fieldScan...) // 扫描查询结果
		if err != nil {                // 如果扫描记录过程中发生错误
			return err // 返回错误
		}

		// 获取 data 的类型和值
		tVar := t.Elem()                       // 获取指针指向的元素类型
		vVar := reflect.ValueOf(data).Elem()   // 获取指针指向的元素值
		for i := 0; i < tVar.NumField(); i++ { // 遍历 data 结构体的每个字段
			name := tVar.Field(i).Name // 获取字段名称
			tag := tVar.Field(i).Tag   // 获取字段标签
			sqlTag := tag.Get("msorm") // 获取 msorm 标签的值
			if sqlTag == "" {          // 如果没有标签
				sqlTag = strings.ToLower(Name(name)) // 使用字段名称的小写形式
			} else {
				if strings.Contains(sqlTag, ",") { // 如果标签中包含逗号
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")] // 处理标签中的逗号
				}
			}

			// 将查询结果映射到 data 结构体中
			for j, colName := range columns { // 遍历查询结果的列名
				if sqlTag == colName { // 如果查询结果的列名与字段标签匹配
					target := values[j]                    // 获取查询结果值
					targetValue := reflect.ValueOf(target) // 获取查询结果值的反射类型
					fieldType := tVar.Field(i).Type        // 获取字段类型
					// 转换查询结果值的类型
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					vVar.Field(i).Set(result) // 将查询结果值赋值给 data 结构体的字段
				}
			}
		}
	}
	return nil // 返回 nil 表示成功
}

// Select 方法用于从数据库中选择多条记录，并将结果映射到 data 结构体中
func (s *MsSession) Select(data any, fields ...string) ([]any, error) {
	t := reflect.TypeOf(data)        // 获取 data 的类型
	if t.Kind() != reflect.Pointer { // 检查 data 是否为指针类型
		return nil, errors.New("data must be pointer") // 如果 data 不是指针类型，返回错误
	}

	// 构建查询字段
	fieldStr := "*"      // 默认查询所有字段
	if len(fields) > 0 { // 如果指定了字段
		fieldStr = strings.Join(fields, ",") // 使用指定的字段
	}

	// 构建查询语句
	query := fmt.Sprintf("select %s from %s ", fieldStr, s.tableName) // 构建查询语句
	var sb strings.Builder                                            // 创建字符串构建器
	sb.WriteString(query)                                             // 写入查询语句的前半部分
	sb.WriteString(s.whereParam.String())                             // 写入 WHERE 子句
	s.db.logger.Info(sb.String())                                     // 记录生成的查询语句到日志中

	// 预处理 SQL 语句
	stmt, err := s.db.db.Prepare(sb.String()) // 预处理 SQL 语句
	if err != nil {                           // 如果预处理过程中发生错误
		return nil, err // 返回错误
	}

	// 执行查询
	rows, err := stmt.Query(s.whereValues...) // 执行查询
	if err != nil {                           // 如果查询过程中发生错误
		return nil, err // 返回错误
	}

	// 获取查询结果的列名
	columns, err := rows.Columns() // 获取查询结果的列名
	if err != nil {                // 如果获取列名过程中发生错误
		return nil, err // 返回错误
	}

	result := make([]any, 0) // 创建用于存储查询结果的切片

	// 遍历查询结果
	for {
		if rows.Next() { // 如果有查询结果
			data := reflect.New(t.Elem()).Interface() // 创建新的 data 实例
			values := make([]any, len(columns))       // 创建存储查询结果的切片
			fieldScan := make([]any, len(columns))    // 创建存储查询字段扫描结果的切片
			for i := range fieldScan {                // 遍历 fieldScan
				fieldScan[i] = &values[i] // 将 values 中的每个元素的地址赋给 fieldScan
			}

			err := rows.Scan(fieldScan...) // 扫描查询结果
			if err != nil {                // 如果扫描记录过程中发生错误
				return nil, err // 返回错误
			}

			// 获取 data 的类型和值
			tVar := t.Elem()                       // 获取指针指向的元素类型
			vVar := reflect.ValueOf(data).Elem()   // 获取指针指向的元素值
			for i := 0; i < tVar.NumField(); i++ { // 遍历 data 结构体的每个字段
				name := tVar.Field(i).Name // 获取字段名称
				tag := tVar.Field(i).Tag   // 获取字段标签
				sqlTag := tag.Get("msorm") // 获取 msorm 标签的值
				if sqlTag == "" {          // 如果没有标签
					sqlTag = strings.ToLower(Name(name)) // 使用字段名称的小写形式
				} else {
					if strings.Contains(sqlTag, ",") { // 如果标签中包含逗号
						sqlTag = sqlTag[:strings.Index(sqlTag, ",")] // 处理标签中的逗号
					}
				}

				// 将查询结果映射到 data 结构体中
				for j, colName := range columns { // 遍历查询结果的列名
					if sqlTag == colName { // 如果查询结果的列名与字段标签匹配
						target := values[j]                    // 获取查询结果值
						targetValue := reflect.ValueOf(target) // 获取查询结果值的反射类型
						fieldType := tVar.Field(i).Type        // 获取字段类型
						// 转换查询结果值的类型
						result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
						vVar.Field(i).Set(result) // 将查询结果值赋值给 data 结构体的字段
					}
				}
			}
			result = append(result, data) // 将 data 实例添加到结果切片中
		} else {
			break // 如果没有更多的查询结果，退出循环
		}
	}

	return result, nil // 返回查询结果和 nil 错误表示成功
}

// Delete 方法用于从数据库中删除记录
func (s *MsSession) Delete() (int64, error) {
	// 构建删除语句
	query := fmt.Sprintf("delete from %s ", s.tableName) // 构建删除语句
	var sb strings.Builder                               // 创建字符串构建器
	sb.WriteString(query)                                // 写入删除语句的前半部分
	sb.WriteString(s.whereParam.String())                // 写入 WHERE 子句
	s.db.logger.Info(sb.String())                        // 记录生成的删除语句到日志中

	// 预处理 SQL 语句
	var stmt *sql.Stmt // 声明 SQL 语句预处理对象
	var err error      // 声明错误变量
	if s.beginTx {
		stmt, err = s.tx.Prepare(sb.String()) // 使用事务的预处理
	} else {
		stmt, err = s.db.db.Prepare(sb.String()) // 使用数据库连接的预处理
	}
	if err != nil { // 如果预处理过程中发生错误
		return 0, err // 返回错误
	}

	// 执行删除操作
	r, err := stmt.Exec(s.whereValues...) // 执行删除操作，将值传递给占位符
	if err != nil {                       // 如果执行过程中发生错误
		return 0, err // 返回错误
	}

	// 获取受影响的行数
	return r.RowsAffected() // 返回受影响的行数
}

// Exec 方法用于执行 SQL 语句，如插入、更新或删除操作
func (s *MsSession) Exec(query string, values ...any) (int64, error) {
	var stmt *sql.Stmt // 声明 SQL 语句预处理对象
	var err error      // 声明错误变量
	if s.beginTx {     // 如果开启了事务
		stmt, err = s.tx.Prepare(query) // 使用事务的预处理
	} else {
		stmt, err = s.db.db.Prepare(query) // 使用数据库连接的预处理
	}
	if err != nil { // 如果预处理过程中发生错误
		return 0, err // 返回错误
	}

	// 执行 SQL 语句
	r, err := stmt.Exec(values...) // 执行 SQL 语句，并传递参数值
	if err != nil {                // 如果执行过程中发生错误
		return 0, err // 返回错误
	}

	// 判断是否为插入语句
	if strings.Contains(strings.ToLower(query), "insert") { // 如果查询语句中包含 "insert"
		return r.LastInsertId() // 返回最后插入的 ID
	}

	// 返回受影响的行数
	return r.RowsAffected() // 返回受影响的行数
}

// QueryRow 方法用于执行查询并将结果映射到数据结构
func (s *MsSession) QueryRow(sql string, data any, queryValues ...any) error {
	t := reflect.TypeOf(data)        // 获取 data 的类型
	if t.Kind() != reflect.Pointer { // 检查 data 是否为指针类型
		return errors.New("data must be pointer") // 如果 data 不是指针类型，返回错误
	}
	stmt, err := s.db.db.Prepare(sql) // 预处理 SQL 语句
	if err != nil {                   // 如果预处理过程中发生错误
		return err // 返回错误
	}
	rows, err := stmt.Query(queryValues...) // 执行查询，获取结果集
	if err != nil {                         // 如果查询过程中发生错误
		return err // 返回错误
	}
	// 获取查询结果的列名
	columns, err := rows.Columns() // 获取查询结果的列名
	if err != nil {                // 如果获取列名过程中发生错误
		return err // 返回错误
	}
	values := make([]any, len(columns))    // 创建存储查询结果的切片
	fieldScan := make([]any, len(columns)) // 创建存储查询字段扫描结果的切片
	for i := range fieldScan {             // 遍历 fieldScan
		fieldScan[i] = &values[i] // 将 values 中的每个元素的地址赋给 fieldScan
	}
	if rows.Next() { // 如果有查询结果
		err := rows.Scan(fieldScan...) // 扫描查询结果
		if err != nil {                // 如果扫描记录过程中发生错误
			return err // 返回错误
		}
		tVar := t.Elem()                       // 获取指针指向的元素类型
		vVar := reflect.ValueOf(data).Elem()   // 获取指针指向的元素值
		for i := 0; i < tVar.NumField(); i++ { // 遍历 data 结构体的每个字段
			name := tVar.Field(i).Name // 获取字段名称
			tag := tVar.Field(i).Tag   // 获取字段标签
			sqlTag := tag.Get("msorm") // 获取 msorm 标签的值
			if sqlTag == "" {          // 如果没有标签
				sqlTag = strings.ToLower(Name(name)) // 使用字段名称的小写形式
			} else {
				if strings.Contains(sqlTag, ",") { // 如果标签中包含逗号
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")] // 处理标签中的逗号
				}
			}

			for j, colName := range columns { // 遍历查询结果的列名
				if sqlTag == colName { // 如果查询结果的列名与字段标签匹配
					target := values[j]                                                   // 获取查询结果值
					targetValue := reflect.ValueOf(target)                                // 获取查询结果值的反射类型
					fieldType := tVar.Field(i).Type                                       // 获取字段类型
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType) // 转换查询结果值的类型
					vVar.Field(i).Set(result)                                             // 将查询结果值赋值给 data 结构体的字段
				}
			}
		}
	}
	return nil // 返回 nil 表示成功
}

// Begin 方法用于开始一个事务
func (s *MsSession) Begin() error {
	tx, err := s.db.db.Begin() // 开始一个新的事务
	if err != nil {            // 如果开始事务时发生错误
		return err // 返回错误
	}
	s.tx = tx        // 将事务对象赋值给会话的 tx 字段
	s.beginTx = true // 将会话的 beginTx 标志设置为 true，表示事务已经开始
	return nil       // 返回 nil 表示成功
}

// Commit 方法用于提交事务
func (s *MsSession) Commit() error {
	err := s.tx.Commit() // 提交事务
	if err != nil {      // 如果提交事务时发生错误
		return err // 返回错误
	}
	s.beginTx = false // 将会话的 beginTx 标志设置为 false，表示事务已提交
	return nil        // 返回 nil 表示成功
}

// Rollback 方法用于回滚事务
func (s *MsSession) Rollback() error {
	err := s.tx.Rollback() // 回滚事务
	if err != nil {        // 如果回滚事务时发生错误
		return err // 返回错误
	}
	s.beginTx = false // 将会话的 beginTx 标志设置为 false，表示事务已回滚
	return nil        // 返回 nil 表示成功
}
