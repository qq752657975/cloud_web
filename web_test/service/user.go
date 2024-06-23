package service

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ygb616/web/orm"
	"net/url"
)

//type User struct {
//	Id       int64  `msorm:"id,auto_increment"`
//	UserName string `msorm:"user_name"`
//	Password string `msorm:"password"`
//	Age      int    `msorm:"age"`
//}

type User struct {
	Id       int64
	UserName string
	Password string
	Age      int
}

func SaveUser() {
	dataSourceName := fmt.Sprintf("root:root@tcp(localhost:3306)/cloud_web?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	db.Prefix = "web_"
	user := &User{
		Id:       10000,
		UserName: "mszlu",
		Password: "123456",
		Age:      30,
	}
	id, _, err := db.New(&User{}).Insert(user)
	if err != nil {
		panic(err)
	}
	fmt.Println(id)

	db.Close()
}
