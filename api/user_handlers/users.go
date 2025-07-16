// api/user_handlers/register.go
package user_handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"star-fire/internal/models"
	"star-fire/pkg/utils"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// 存储验证码（生产环境建议使用Redis）
var (
	codeStore = make(map[string]codeInfo)
	codeMutex sync.Mutex
)

type codeInfo struct {
	code      string
	expiresAt time.Time
}

type UserHandler struct {
	server *models.Server
}

func NewUserHandler(server *models.Server) *UserHandler {
	return &UserHandler{
		server: server,
	}
}

// SendVerificationCode 发送验证码
func (uh *UserHandler) SendVerificationCode(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供有效的邮箱地址"})
		return
	}

	// 生成6位验证码
	rand.Seed(time.Now().UnixNano())
	code := fmt.Sprintf("%06d", rand.Intn(1000000))

	// 存储验证码(10分钟有效期)
	codeMutex.Lock()
	codeStore[req.Email] = codeInfo{
		code:      code,
		expiresAt: time.Now().Add(10 * time.Minute),
	}
	codeMutex.Unlock()

	// 发送邮件
	subject := "星火算力计划 - 注册验证码"
	body := fmt.Sprintf("<p>您好，</p><p>您的注册验证码是：<strong>%s</strong></p><p>验证码有效期为10分钟。</p>", code)

	if err := utils.SendEmail(req.Email, subject, body, uh.server.MailService.FromAddress,
		uh.server.MailService.SMTPServer, uh.server.MailService.SMTPUsername, uh.server.MailService.SMTPPassword,
		uh.server.MailService.SMTPPort); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送验证码失败:" + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送"})
}

// Register 用户注册
func (uh *UserHandler) Register(c *gin.Context, server *models.Server) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
		Code     string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供完整的注册信息"})
		return
	}

	// 验证码校验
	codeMutex.Lock()
	codeInfo, exists := codeStore[req.Email]
	valid := exists && codeInfo.code == req.Code && time.Now().Before(codeInfo.expiresAt)
	if valid {
		delete(codeStore, req.Email) // 使用后删除验证码
	}
	codeMutex.Unlock()

	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码无效或已过期"})
		return
	}

	// 检查邮箱是否已注册
	if server.UserDB.UserExistsByEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邮箱已被注册"})
		return
	}

	// 检查用户名是否已存在
	if server.UserDB.UserExistsByUsername(req.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已被使用"})
		return
	}

	// 加密密码
	fmt.Println("req===", req, "req.Password===", req.Password)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 创建用户
	user := models.User{
		Email:    req.Email,
		Username: req.Username,
		Password: string(hashedPassword),
	}

	// search the max user ID and set new user ID
	maxID, err := server.UserDB.GetMaxUserID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户ID失败"})
		return
	}
	user.ID = fmt.Sprintf("%d", maxID+1) // 假设ID是数字类型，转换为字符串
	if err := server.UserDB.SaveUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户注册失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "注册成功"})
}
