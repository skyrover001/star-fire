// 开启星火算力计划，支持个人用户的PC大模型（星火）汇入算力银河为其他需要的用户提供大模型服务，共享分成。
// server东侧接受client的注册，西侧接受用户端的大模型请求，通过分配算法将这些请求转发到client端。
// client注册和问答都是通过客户端websocket的方式，不需要西侧client用户提供任何互联网入口。
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	configs "star-fire/config"
	"star-fire/internal/models"
	"star-fire/routes"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 命令行模式：配置注册赠送余额（不启动 HTTP 服务）
	if len(os.Args) > 1 {
		switch os.Args[1] {
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
