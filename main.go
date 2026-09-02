// 开启星火算力计划，支持个人用户的PC大模型（星火）汇入算力银河为其他需要的用户提供大模型服务，共享分成。
// server东侧接受client的注册，西侧接受用户端的大模型请求，通过分配算法将这些请求转发到client端。
// client注册和问答都是通过客户端websocket的方式，不需要西侧client用户提供任何互联网入口。
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	configs "star-fire/config"
	"star-fire/internal/models"
	"star-fire/pkg/public"
	"star-fire/routes"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// maskAPIKey 脱敏 APIKey：仅显示前 4 + 后 4，中间用 **** 代替。
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// printUsage 打印所有 CLI 命令的用法说明。
func printUsage() {
	fmt.Println("星火算力平台 CLI 用法:")
	fmt.Println("  starfire [命令] [参数...]")
	fmt.Println("  不带任何命令时启动 HTTP 服务。")
	fmt.Println()
	fmt.Println("通用:")
	fmt.Println("  starfire --help | -h | help        显示本帮助")
	fmt.Println()
	fmt.Println("会员 / 余额:")
	fmt.Println("  starfire set-bonus <金额>          设置注册赠送余额，如 starfire set-bonus 10")
	fmt.Println("  starfire get-bonus                 查看当前注册赠送余额")
	fmt.Println("  starfire set-membership <用户名> <normal|vip|svip> [天数] [consumer|contributor]")
	fmt.Println("                                     设置会员等级；天数默认 365，传 0 表示永久；类型默认 consumer")
	fmt.Println("  starfire get-membership <用户名>   查看用户会员信息")
	fmt.Println("  starfire list-users [page] [size]  分页列出用户（默认第 1 页，每页 20）")
	fmt.Println()
	fmt.Println("数据库迁移:")
	fmt.Println("  starfire migrate-db [-sqlite=<路径>] [-force] [-users-only]")
	fmt.Println("                                     将 SQLite 数据迁移到 MySQL（目标由 DB_* 环境变量指定）")
	fmt.Println()
	fmt.Println("Direct 后端管理:")
	fmt.Println("  starfire add-backend <id> <名称> <base_url> <api_key> <优先级> <并发> <模型列表>")
	fmt.Println("                                     添加 Direct 后端；模型支持 name 或 name:ippm:oppm:cippm")
	fmt.Println("                                     示例: starfire add-backend vllm-1 \"vLLM 集群 1\" http://vllm-1:8000/v1 sk-xxx 1 8 qwen3-32b:1.5:2.0:0.3,qwen3-8b")
	fmt.Println("  starfire list-backends             列出所有 Direct 后端（含模型价格）")
	fmt.Println("  starfire enable-backend <id>       启用 Direct 后端（重启后生效）")
	fmt.Println("  starfire disable-backend <id>      禁用 Direct 后端（重启后生效）")
	fmt.Println("  starfire del-backend <id>          删除 Direct 后端（重启后生效）")
	fmt.Println()
	fmt.Println("提示: 后端管理命令的变更会在服务运行期间 30 秒内自动加载。")
}

func main() {
	// 命令行模式：配置注册赠送余额（不启动 HTTP 服务）
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h", "help":
			printUsage()
			return
		case "set-bonus":
			// 用法: starfire set-bonus <金额>
			if len(os.Args) < 3 {
				log.Fatal("用法: starfire set-bonus <金额>，例如 starfire set-bonus 10")
			}
			amount, err := strconv.ParseFloat(os.Args[2], 64)
			if err != nil || amount < 0 {
				log.Fatal("金额必须是大于等于 0 的数字")
			}
			server := models.NewServer()
			if err := server.SystemConfigDB.Set(models.ConfigKeyRegisterBonus,
				strconv.FormatFloat(amount, 'f', -1, 64)); err != nil {
				log.Fatalf("设置失败: %v", err)
			}
			log.Printf("✓ 已设置注册赠送余额为 %.2f 元", amount)
			return
		case "get-bonus":
			server := models.NewServer()
			bonus := server.SystemConfigDB.GetFloat(models.ConfigKeyRegisterBonus, 0)
			log.Printf("当前注册赠送余额: %.2f 元", bonus)
			return
		case "set-membership":
			// 用法: starfire set-membership <username> <normal|vip|svip> [天数] [consumer|contributor]
			// 天数可选，默认 365 天；传 0 表示永久
			// 类型可选，默认 consumer（消费者会员）；contributor 表示贡献者会员
			if len(os.Args) < 4 {
				log.Fatal("用法: starfire set-membership <username> <normal|vip|svip> [天数] [consumer|contributor]")
			}
			username := os.Args[2]
			level := os.Args[3]
			if level != models.MembershipNormal && level != models.MembershipVIP && level != models.MembershipSVIP {
				log.Fatal("会员等级必须是 normal / vip / svip")
			}
			days := 365
			if len(os.Args) >= 5 {
				d, err := strconv.Atoi(os.Args[4])
				if err != nil || d < 0 {
					log.Fatal("天数必须是大于等于 0 的整数")
				}
				days = d
			}
			memType := "consumer"
			if len(os.Args) >= 6 {
				memType = os.Args[5]
				if memType != "consumer" && memType != "contributor" {
					log.Fatal("会员类型必须是 consumer / contributor")
				}
			}
			server := models.NewServer()
			user, err := server.UserDB.GetUser(username)
			if err != nil {
				log.Fatalf("用户不存在: %v", err)
			}
			var expireAt time.Time
			if days > 0 {
				expireAt = time.Now().AddDate(0, 0, days)
			}
			typeName := "消费者"
			if memType == "contributor" {
				typeName = "贡献者"
				if err := server.UserDB.SetContributorMembership(user.ID, level, expireAt); err != nil {
					log.Fatalf("设置会员失败: %v", err)
				}
			} else {
				if err := server.UserDB.SetMembership(user.ID, level, expireAt); err != nil {
					log.Fatalf("设置会员失败: %v", err)
				}
			}
			if days > 0 {
				log.Printf("✓ 已设置用户 %s 为%s %s 会员，%d 天后到期（%s）", username, typeName, level, days, expireAt.Format("2006-01-02"))
			} else {
				log.Printf("✓ 已设置用户 %s 为%s %s 会员（永久）", username, typeName, level)
			}
			return
		case "get-membership":
			// 用法: starfire get-membership <username>
			if len(os.Args) < 3 {
				log.Fatal("用法: starfire get-membership <username>")
			}
			server := models.NewServer()
			user, err := server.UserDB.GetUser(os.Args[2])
			if err != nil {
				log.Fatalf("用户不存在: %v", err)
			}
			effective := server.UserDB.GetEffectiveMembership(user.ID)
			expireStr := "未开通"
			if user.MembershipExpireAt != nil {
				expireStr = user.MembershipExpireAt.Format("2006-01-02")
			}
			contributorEffective := server.UserDB.GetEffectiveContributorMembership(user.ID)
			contributorExpireStr := "未开通"
			if user.ContributorMembershipExpireAt != nil {
				contributorExpireStr = user.ContributorMembershipExpireAt.Format("2006-01-02")
			}
			log.Printf("用户 %s 消费者会员: %s（有效:%s），到期: %s",
				user.Username, user.Membership, effective, expireStr)
			log.Printf("用户 %s 贡献者会员: %s（有效:%s），到期: %s，连接数上限: %d",
				user.Username, user.ContributorMembership, contributorEffective, contributorExpireStr,
				models.GetMaxConnections(contributorEffective))
			return
		case "list-users":
			// 用法: starfire list-users [page] [size]
			server := models.NewServer()
			page, size := 1, 20
			if len(os.Args) >= 3 {
				if p, err := strconv.Atoi(os.Args[2]); err == nil && p > 0 {
					page = p
				}
			}
			if len(os.Args) >= 4 {
				if s, err := strconv.Atoi(os.Args[3]); err == nil && s > 0 {
					size = s
				}
			}
			users, total, err := server.UserDB.ListUsers(page, size)
			if err != nil {
				log.Fatalf("查询用户失败: %v", err)
			}
			log.Printf("共 %d 个用户（第 %d 页，每页 %d）:", total, page, size)
			for _, u := range users {
				effective := server.UserDB.GetEffectiveMembership(u.ID)
				expireStr := "未开通"
				if u.MembershipExpireAt != nil {
					expireStr = u.MembershipExpireAt.Format("2006-01-02")
				}
				contributorEffective := server.UserDB.GetEffectiveContributorMembership(u.ID)
				contributorExpireStr := "未开通"
				if u.ContributorMembershipExpireAt != nil {
					contributorExpireStr = u.ContributorMembershipExpireAt.Format("2006-01-02")
				}
				log.Printf("  %-20s 消费:%s(有效:%s) 到期:%s | 贡献:%s(有效:%s) 到期:%s 余额:%.2f",
					u.Username, u.Membership, effective, expireStr,
					u.ContributorMembership, contributorEffective, contributorExpireStr, u.Balance)
			}
			return
		case "migrate-db":
			// 用法: starfire migrate-db [-sqlite <路径>] [-force] [-users-only]
			// 将现有 SQLite 数据完整迁移到 MySQL（目标由 DB_DRIVER/DB_DSN/DB_* 环境变量指定）。
			// 目标表非空时默认跳过（幂等，可断点重跑）；-force 清空目标后重迁；
			// -users-only 只迁移 users 表（其余表不建、不拷贝）。
			sqlitePath := ""
			force := false
			usersOnly := false
			for _, arg := range os.Args[2:] {
				switch {
				case arg == "-force":
					force = true
				case arg == "-users-only":
					usersOnly = true
				case strings.HasPrefix(arg, "-sqlite="):
					sqlitePath = strings.TrimPrefix(arg, "-sqlite=")
				case arg == "-sqlite":
					log.Fatal("用法: starfire migrate-db [-sqlite=<路径>] [-force] [-users-only]")
				default:
					log.Fatalf("未知参数: %s", arg)
				}
			}
			if sqlitePath != "" {
				os.Setenv("DATABASE_PATH", sqlitePath)
				configs.Config.DBPath = sqlitePath
			}
			if err := models.MigrateSQLiteToMySQL(force, usersOnly); err != nil {
				log.Fatalf("迁移失败: %v", err)
			}
			return
		case "add-backend":
			// 用法: starfire add-backend <id> <name> <base_url> <api_key> <priority> <max_conns> <model1,model2,...>
			// 模型参数支持 "name" 或 "name:ippm:oppm:cippm"（价格可选，缺省 0）。
			// 示例: starfire add-backend vllm-1 "vLLM 集群 1" http://vllm-1:8000/v1 sk-xxx 1 8 qwen3-32b:1.5:2.0:0.3,qwen3-8b
			if len(os.Args) < 9 {
				log.Fatal("用法: starfire add-backend <id> <name> <base_url> <api_key> <priority> <max_conns> <model1,model2,...>")
			}
			id := os.Args[2]
			name := os.Args[3]
			baseURL := os.Args[4]
			apiKey := os.Args[5]
			priority, err := strconv.Atoi(os.Args[6])
			if err != nil {
				log.Fatal("priority 必须是整数")
			}
			maxConns, err := strconv.Atoi(os.Args[7])
			if err != nil || maxConns <= 0 {
				log.Fatal("max_conns 必须是大于 0 的整数")
			}
			modelSpecs := strings.Split(os.Args[8], ",")
			var ms []*public.Model
			for _, spec := range modelSpecs {
				spec = strings.TrimSpace(spec)
				if spec == "" {
					continue
				}
				parts := strings.Split(spec, ":")
				m := &public.Model{Name: strings.TrimSpace(parts[0])}
				if m.Name == "" {
					continue
				}
				// 可选价格 name:ippm:oppm:cippm
				if len(parts) >= 4 {
					ippm, e1 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
					oppm, e2 := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
					cippm, e3 := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
					if e1 != nil || e2 != nil || e3 != nil {
						log.Fatalf("模型 %s 价格格式错误，应为 name:ippm:oppm:cippm", m.Name)
					}
					m.IPPM, m.OPPM, m.CIPPM = ippm, oppm, cippm
				}
				ms = append(ms, m)
			}
			if len(ms) == 0 {
				log.Fatal("至少需要一个模型")
			}
			server := models.NewServer()
			b := &models.DirectBackend{
				ID:       id,
				Name:     name,
				BaseURL:  baseURL,
				APIKey:   apiKey,
				Format:   "openai",
				Enabled:  true,
				MaxConns: maxConns,
				Priority: priority,
				Models:   ms,
			}
			if err := server.DirectBackendDB.Save(b); err != nil {
				log.Fatalf("添加后端失败: %v", err)
			}
			log.Printf("✓ 已添加后端 %s (%s)，模型: %s，优先级: %d，并发上限: %d", id, name, strings.Join(modelSpecs, ","), priority, maxConns)
			log.Printf("  服务会在 30 秒内自动加载变更")
			return
		case "list-backends":
			// 用法: starfire list-backends
			server := models.NewServer()
			backends, err := server.DirectBackendDB.List()
			if err != nil {
				log.Fatalf("查询后端失败: %v", err)
			}
			if len(backends) == 0 {
				log.Printf("暂无 Direct 后端")
				return
			}
			log.Printf("共 %d 个 Direct 后端:", len(backends))
			for _, b := range backends {
				status := "禁用"
				if b.Enabled {
					status = "启用"
				}
				models := make([]string, 0, len(b.Models))
				for _, m := range b.Models {
					models = append(models, fmt.Sprintf("%s(%.4g/%.4g/%.4g)", m.Name, m.IPPM, m.OPPM, m.CIPPM))
				}
				// APIKey 脱敏：仅显示前 4 + 后 4
				masked := maskAPIKey(b.APIKey)
				log.Printf("  %-16s %-20s 格式:%s 状态:%s 优先级:%d 并发:%d 模型:[%s] key:%s",
					b.ID, b.Name, b.Format, status, b.Priority, b.MaxConns, strings.Join(models, ","), masked)
			}
			return
		case "enable-backend":
			// 用法: starfire enable-backend <id>
			if len(os.Args) < 3 {
				log.Fatal("用法: starfire enable-backend <id>")
			}
			server := models.NewServer()
			if err := server.DirectBackendDB.SetEnabled(os.Args[2], true); err != nil {
				log.Fatalf("启用后端失败: %v", err)
			}
			log.Printf("✓ 已启用后端 %s（服务会在 30 秒内自动加载）", os.Args[2])
			return
		case "disable-backend":
			// 用法: starfire disable-backend <id>
			if len(os.Args) < 3 {
				log.Fatal("用法: starfire disable-backend <id>")
			}
			server := models.NewServer()
			if err := server.DirectBackendDB.SetEnabled(os.Args[2], false); err != nil {
				log.Fatalf("禁用后端失败: %v", err)
			}
			log.Printf("✓ 已禁用后端 %s（服务会在 30 秒内自动加载）", os.Args[2])
			return
		case "del-backend":
			// 用法: starfire del-backend <id>
			if len(os.Args) < 3 {
				log.Fatal("用法: starfire del-backend <id>")
			}
			server := models.NewServer()
			if err := server.DirectBackendDB.Delete(os.Args[2]); err != nil {
				log.Fatalf("删除后端失败: %v", err)
			}
			log.Printf("✓ 已删除后端 %s（服务会在 30 秒内自动加载）", os.Args[2])
			return
		}
	}

	server := models.NewServer()
	r := gin.Default()
	routes.SetupRoutes(r, server)

	srv := &http.Server{
		Addr:    configs.Config.ServerPort,
		Handler: r,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Println("Starting server on", configs.Config.ServerPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Error while starting server:", err)
	}
}
