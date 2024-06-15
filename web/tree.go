package web

import "strings"

// treeNode 代表路由树中的一个节点
type treeNode struct {
	name       string      // 节点的名称，通常是路径段或通配符
	children   []*treeNode // 当前节点的子节点数组
	routerName string      // 导致该节点的完整路由路径
	isEnd      bool        // 是否是尾节点
}

// Put 方法向路由树中插入一个路径
// 示例路径: /user/get/:id
func (t *treeNode) Put(path string) {
	root := t                        // 保存根节点
	strs := strings.Split(path, "/") // 按 "/" 分割路径
	for index, name := range strs {
		if index == 0 {
			continue // 跳过第一个元素，因为它是空字符串
		}
		children := t.children
		isMatch := false
		for _, node := range children {
			if node.name == name {
				isMatch = true // 找到匹配的节点，移动到该子节点
				t = node
				break
			}
		}
		if !isMatch {
			isEnd := false
			if index == len(strs)-1 {
				isEnd = true
			}
			// 如果没有找到匹配的节点，则创建一个新节点
			node := &treeNode{name: name, children: make([]*treeNode, 0), isEnd: isEnd}
			children = append(children, node) // 将新节点添加到子节点中
			t.children = children             // 更新当前节点的子节点
			t = node                          // 移动到新节点
		}
	}
	t = root // 插入完成后，将指针重置为根节点
}

// Get 方法从路由树中检索与路径对应的节点
// 示例路径: /user/get/1
func (t *treeNode) Get(path string) *treeNode {
	stars := strings.Split(path, "/") // 按 "/" 分割路径
	routerName := ""
	for index, name := range stars {
		if index == 0 {
			continue // 跳过第一个元素，因为它是空字符串
		}
		children := t.children
		isMatch := false
		for _, node := range children {
			// 检查是否有名称匹配、通配符 "*" 或参数 ":"
			if node.name == name || node.name == "*" || strings.Contains(node.name, ":") {
				isMatch = true
				routerName += "/" + node.name
				node.routerName = routerName // 设置完整的路由路径
				t = node
				if index == len(stars)-1 {
					return node // 如果是路径的末尾，则返回该节点
				}
				break
			}
		}
		if !isMatch {
			// 如果没有找到直接匹配的节点，则检查是否有 "**" 通配符匹配
			for _, node := range children {
				if node.name == "**" {
					// /user/**
					// /user/get/userInfo
					// /user/aa/bb
					routerName += "/" + node.name
					node.routerName = routerName
					return node // 返回通配符匹配的节点
				}
			}
		}
	}
	return nil // 如果没有找到匹配的节点，则返回 nil
}
