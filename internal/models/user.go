package models

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID         string  `gorm:"primaryKey" json:"id"` // 应用层分配的字符串 ID（MySQL 下 autoIncrement 标签无效且具误导性，故移除）
	Username   string  `gorm:"uniqueIndex;not null" json:"username"`
	Password   string  `gorm:"not null" json:"-"`
	Email      string  `gorm:"index" json:"email"`
	Role       string  `gorm:"default:user;not null" json:"role"`
	Balance    float64 `gorm:"default:0;not null" json:"balance"`     // 账户余额（元）
	TotalSpent float64 `gorm:"default:0;not null" json:"total_spent"` // 累计消费（元）
	// 消费者会员等级：normal / vip / svip，默认普通会员。
	// 用于消费者端（调用 API 推理）的限流等权益。
	Membership         string     `gorm:"default:normal;not null" json:"membership"`
	MembershipExpireAt *time.Time `json:"membership_expire_at"` // 消费者会员到期时间（nil 表示未开通；MySQL 严格模式禁止 0000-00-00 零日期，故用指针+NULL）
	// 贡献者会员等级：normal / vip / svip，默认普通会员。
	// 用于贡献者端（贡献算力、接入 client）的连接数上限等权益。
	// 与消费者会员相互独立，可分别购买/升级。
	ContributorMembership         string     `gorm:"default:normal;not null" json:"contributor_membership"`
	ContributorMembershipExpireAt *time.Time `json:"contributor_membership_expire_at"` // 贡献者会员到期时间（nil 表示未开通）
	CreatedAt                     time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt                     time.Time  `gorm:"not null" json:"updated_at"`
}

// UserDB
type UserDB struct {
	db *gorm.DB
}

// NewUserDB
func NewUserDB(db *gorm.DB) *UserDB {
	db.AutoMigrate(&User{})
	return &UserDB{db: db}
}

// GetUser
func (udb *UserDB) GetUser(username string) (*User, error) {
	var user User
	result := udb.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}
	return &user, nil
}

// ValidatePassword
func (udb *UserDB) ValidatePassword(username, password string) (*User, error) {
	user, err := udb.GetUser(username)
	fmt.Println("Validating user:", username, "Error:", err)
	if err != nil {
		if err.Error() == "user not found" {
			var userByEmail User
			result := udb.db.Where("email = ?", username).First(&userByEmail)
			fmt.Println("result is:", result)
			if result.Error != nil {
				return nil, errors.New("user not found")
			}
			user = &userByEmail
		} else {
			return nil, err
		}
	}

	fmt.Println("Validating password:", password, "for user:", user.Username, "user.Password:", user.Password)
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	fmt.Println("err==", err)
	if err != nil {
		return nil, errors.New("password is incorrect")
	}

	return user, nil
}

// AddUser
func (udb *UserDB) AddUser(user *User) error {
	var count int64
	udb.db.Model(&User{}).Where("username = ?", user.Username).Count(&count)
	if count > 0 {
		return errors.New("user is already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now()
	}
	user.UpdatedAt = time.Now()

	return udb.db.Create(user).Error
}

// GetUserByID
func (udb *UserDB) GetUserByID(id string) (*User, error) {
	var user User
	result := udb.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}
	return &user, nil
}

// UpdateUser
func (udb *UserDB) UpdateUser(user *User) error {
	user.UpdatedAt = time.Now()
	return udb.db.Save(user).Error
}

// InitDefaultUsers
func (udb *UserDB) InitDefaultUsers() error {
	var count int64
	udb.db.Model(&User{}).Count(&count)
	if count > 0 {
		return nil
	}

	adminPwd, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	admin := &User{
		ID:        "1",
		Username:  "admin",
		Password:  string(adminPwd),
		Email:     "admin@example.com",
		Role:      "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	userPwd, _ := bcrypt.GenerateFromPassword([]byte("user123"), bcrypt.DefaultCost)
	user := &User{
		ID:        "2",
		Username:  "skyrover001",
		Password:  string(userPwd),
		Email:     "user@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return udb.db.Create([]*User{admin, user}).Error
}

func (udb *UserDB) UserExistsByEmail(email string) bool {
	var count int64
	udb.db.Model(&User{}).Where("email = ?", email).Count(&count)
	return count > 0
}

func (udb *UserDB) UserExistsByUsername(username string) bool {
	var count int64
	udb.db.Model(&User{}).Where("username = ?", username).Count(&count)
	return count > 0
}

func (udb *UserDB) SaveUser(user *User) error {
	// 检查是否已有ID，决定是创建还是更新
	if user.ID == "" {
		// 为新用户生成密码哈希
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		user.Password = string(hashedPassword)
		user.CreatedAt = time.Now()
		user.UpdatedAt = time.Now()

		return udb.db.Create(user).Error
	} else {
		user.UpdatedAt = time.Now()
		return udb.db.Save(user).Error
	}
}

// get max user ID
func (udb *UserDB) GetMaxUserID() (int, error) {
	var maxID int
	result := udb.db.Model(&User{}).Select("MAX(CAST(id AS UNSIGNED))").Scan(&maxID)
	if result.Error != nil {
		return 0, result.Error
	}
	if maxID == 0 {
		return 0, nil // No users found
	}
	return maxID, nil
}

// DeductBalance deducts amount from user balance. Allows balance going negative as long as it was > 0 before deduction.
// 原子扣费：单条 UPDATE 在数据库端完成条件判断与扣减，消除并发下的 SELECT→UPDATE 竞态
// （两个请求同时读到 balance>0 后各自扣减，可能导致余额透支或 total_spent 少计）。
func (udb *UserDB) DeductBalance(userID string, amount float64) error {
	result := udb.db.Model(&User{}).
		Where("id = ? AND balance > 0", userID).
		Updates(map[string]interface{}{
			"balance":     gorm.Expr("balance - ?", amount),
			"total_spent": gorm.Expr("total_spent + ?", amount),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 用户不存在，或余额已 <= 0
		return errors.New("insufficient balance")
	}
	return nil
}

// AddBalance adds amount to user balance
func (udb *UserDB) AddBalance(userID string, amount float64) error {
	return udb.db.Model(&User{}).Where("id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error
}

// GetBalance returns user's balance and total spent
func (udb *UserDB) GetBalance(userID string) (balance float64, totalSpent float64, err error) {
	var user User
	result := udb.db.Where("id = ?", userID).First(&user)
	if result.Error != nil {
		return 0, 0, result.Error
	}
	return user.Balance, user.TotalSpent, nil
}

// GetEffectiveMembership 返回用户当前有效的会员等级。
// 若会员已过期或未开通，返回普通会员（normal）。
func (udb *UserDB) GetEffectiveMembership(userID string) string {
	var user User
	if err := udb.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return MembershipNormal
	}
	if user.Membership == "" {
		return MembershipNormal
	}
	// 会员已过期则降为普通
	if user.MembershipExpireAt != nil && time.Now().After(*user.MembershipExpireAt) {
		return MembershipNormal
	}
	return user.Membership
}

// SetMembership 设置用户会员等级和到期时间。
// 传入 expireAt 为零值表示永久（管理员手动设置时可用）。
// 内部将零值转换为 NULL 存储（MySQL 严格模式禁止 0000-00-00 零日期）。
func (udb *UserDB) SetMembership(userID, level string, expireAt time.Time) error {
	var expirePtr *time.Time
	if !expireAt.IsZero() {
		expirePtr = &expireAt
	}
	return udb.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"membership":           level,
		"membership_expire_at": expirePtr,
	}).Error
}

// GetEffectiveContributorMembership 返回用户当前有效的贡献者会员等级。
// 若贡献者会员已过期或未开通，返回普通会员（normal）。
func (udb *UserDB) GetEffectiveContributorMembership(userID string) string {
	var user User
	if err := udb.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return MembershipNormal
	}
	if user.ContributorMembership == "" {
		return MembershipNormal
	}
	// 会员已过期则降为普通
	if user.ContributorMembershipExpireAt != nil && time.Now().After(*user.ContributorMembershipExpireAt) {
		return MembershipNormal
	}
	return user.ContributorMembership
}

// SetContributorMembership 设置用户贡献者会员等级和到期时间。
// 传入 expireAt 为零值表示永久（管理员手动设置时可用）。
// 内部将零值转换为 NULL 存储（MySQL 严格模式禁止 0000-00-00 零日期）。
func (udb *UserDB) SetContributorMembership(userID, level string, expireAt time.Time) error {
	var expirePtr *time.Time
	if !expireAt.IsZero() {
		expirePtr = &expireAt
	}
	return udb.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"contributor_membership":           level,
		"contributor_membership_expire_at": expirePtr,
	}).Error
}

// ListUsers 分页列出所有用户（管理用）
func (udb *UserDB) ListUsers(page, size int) ([]*User, int64, error) {
	var users []*User
	var total int64
	udb.db.Model(&User{}).Count(&total)
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	err := udb.db.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&users).Error
	return users, total, err
}
