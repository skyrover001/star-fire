package models

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        string    `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"uniqueIndex;not null" json:"username"`
	Password  string    `gorm:"not null" json:"-"`
	Email     string    `gorm:"index" json:"email"`
	Role      string    `gorm:"default:user;not null" json:"role"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
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
