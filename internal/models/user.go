package models

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"` // 不返回给客户端
	Email     string    `json:"email"`
	Role      string    `json:"role"` // "admin" 或 "user"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserStore struct {
	users map[string]*User // username -> user
}

func NewUserStore() *UserStore {
	store := &UserStore{
		users: make(map[string]*User),
	}

	// e.g. user:admin pwd: admin123
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
	store.users[admin.Username] = admin

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
	store.users[user.Username] = user

	return store
}

func (s *UserStore) GetUser(username string) (*User, error) {
	user, ok := s.users[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (s *UserStore) ValidatePassword(username, password string) (*User, error) {
	user, err := s.GetUser(username)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	return user, nil
}

func (s *UserStore) AddUser(user *User) error {
	if _, ok := s.users[user.Username]; ok {
		return errors.New("username already exists")
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	s.users[user.Username] = user
	return nil
}

func (s *UserStore) GetUserByID(id string) (*User, error) {
	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}
