// 定义模型相关的结构体和使用ollama命令运行
package ollama

import "os/exec"

type Model struct {
	Name string `json:"name"` // 模型名称
}

func (model *Model) Run() error {
	return nil
	// 使用exec在后台运行ollama命令
	cmd := exec.Command("ollama", "run", model.Name)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
