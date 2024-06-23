package main

import (
	"fmt"
	"github.com/ygb616/web/orm"
	"testing"
)

func TestName(t *testing.T) {
	fmt.Println(orm.Name("User"))
}
