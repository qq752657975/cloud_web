package gateway

import "strings"

// TreeNode 定义树节点结构体
type TreeNode struct {
	Name       string      // 节点名称
	Children   []*TreeNode // 子节点列表
	RouterName string      // 路由名称
	IsEnd      bool        // 是否为路径末尾
	GwName     string      // 网关名称
}

// Put 方法用于向树中插入路径
// path: /user/get/:id
func (t *TreeNode) Put(path string, gwName string) {
	root := t                        // 保存根节点引用
	strs := strings.Split(path, "/") // 将路径按斜杠分割成字符串数组
	for index, name := range strs {  // 遍历分割后的路径部分
		if index == 0 { // 忽略第一个空字符串
			continue
		}
		children := t.Children          // 获取当前节点的子节点
		isMatch := false                // 标记是否匹配到已有节点
		for _, node := range children { // 遍历子节点
			if node.Name == name { // 如果子节点名称匹配
				isMatch = true // 标记为匹配
				t = node       // 进入匹配的子节点
				break          // 结束当前循环
			}
		}
		if !isMatch { // 如果没有匹配到子节点
			isEnd := false            // 标记是否为路径末尾
			if index == len(strs)-1 { // 如果是路径的最后一个部分
				isEnd = true // 标记为路径末尾
			}
			// 创建新节点并加入子节点列表
			node := &TreeNode{Name: name, Children: make([]*TreeNode, 0), IsEnd: isEnd, GwName: gwName}
			children = append(children, node) // 将新节点添加到子节点列表
			t.Children = children             // 更新当前节点的子节点列表
			t = node                          // 进入新创建的子节点
		}
	}
	t = root // 还原到根节点
}

// Get 方法用于从树中获取路径对应的节点
// path: /user/get/1
// /hello
func (t *TreeNode) Get(path string) *TreeNode {
	strs := strings.Split(path, "/") // 将路径按斜杠分割成字符串数组
	routerName := ""                 // 初始化路由名称
	for index, name := range strs {  // 遍历分割后的路径部分
		if index == 0 { // 忽略第一个空字符串
			continue
		}
		children := t.Children          // 获取当前节点的子节点
		isMatch := false                // 标记是否匹配到已有节点
		for _, node := range children { // 遍历子节点
			if node.Name == name || node.Name == "*" || strings.Contains(node.Name, ":") { // 如果子节点名称匹配
				isMatch = true                // 标记为匹配
				routerName += "/" + node.Name // 更新路由名称
				node.RouterName = routerName  // 设置节点的路由名称
				t = node                      // 进入匹配的子节点
				if index == len(strs)-1 {     // 如果是路径的最后一个部分
					return node // 返回匹配的节点
				}
				break // 结束当前循环
			}
		}
		if !isMatch { // 如果没有匹配到子节点
			for _, node := range children { // 遍历子节点
				if node.Name == "**" { // 检查是否有 "**" 节点
					routerName += "/" + node.Name // 更新路由名称
					node.RouterName = routerName  // 设置节点的路由名称
					return node                   // 返回匹配的 "**" 节点
				}
			}
		}
	}
	return nil // 没有匹配的节点，返回 nil
}
