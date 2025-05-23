package models

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID        string    `gorm:"primaryKey" json:"id"`
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
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
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
