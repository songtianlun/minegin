package model

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"mingin/pkg/auth"
	"mingin/pkg/constvar"
	"mingin/pkg/errno"
	"mingin/pkg/logger"
	"regexp"
	"sync"
)

type UserType uint32

const (
	VISITOR UserType = 0
	USER    UserType = 1
	VIP     UserType = 2
	ADMIN   UserType = 10
)

type UserModel struct {
	BaseModel
	Email string `json:"email" gorm:"column:email;unique" validate:"max=64"`
	// Phone    string   `json:"phone" gorm:"column:phone;unique" validate:"max=16"`
	Password string   `json:"password" gorm:"column:password;not null" validate:"min=5,max=128"`
	Nickname string   `json:"nickname" gorm:"column:nickname;not null;default:-" validate:"min=1,max=128"`
	Role     UserType `json:"role" gorm:"column:role;default:1"`
}

type UserBind struct {
	Username string `json:"username" binding:"required" validate:"min=1,max=32"`
	Password string `json:"password" binding:"required" validate:"min=5,max=128"`
}

type UserInfo struct {
	ID    uint64 `json:"id"`
	Email string `json:"email"`
	// Phone     string   `json:"phone"`
	Nickname  string   `json:"nickname"`
	Role      UserType `json:"role"`
	SayHello  string   `json:"sayHello"`
	CreatedAt string   `json:"-"`
	UpdatedAt string   `json:"-"`
}

type UserList struct {
	Lock  *sync.Mutex
	IdMap map[uint64]*UserInfo
}

func (u *UserModel) Create() error {
	return DB.DB.Create(&u).Error
}

func (u *UserModel) Check() error {
	var user []UserModel
	if u.Email != "" {
		DB.DB.Where("email = ?", u.Email).Find(&user)
		if len(user) != 0 {
			return errno.ErrUserEmail
		}
		// } else if u.Phone != "" {
		// 	DB.DB.Where("phone = ?", u.Phone).Find(&user)
		// 	if len(user) != 0 {
		// 		return errno.ErrUserPhone
		// 	}
	} else {
		return errno.ErrDatabase
	}
	return nil
}

func (u *UserModel) Encrypt() (err error) {
	u.Password, err = auth.PasswordHash(u.Password)
	return
}

// Validate 数据类型校验
func (u *UserModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// GetUser 根据邮箱/手机号获取用户信息
func GetUser(username string) (*UserModel, error) {
	var d *gorm.DB
	u := &UserModel{}
	reg := regexp.MustCompile(
		`^[0-9a-z][_.0-9a-z-]{0,31}@([0-9a-z][0-9a-z-]{0,30}[0-9a-z]\.){1,4}[a-z]{2,4}$`)
	if reg.MatchString(username) {
		logger.Debug("Login with email: " + username)
		d = DB.DB.Where("email = ?", username).First(&u)
	} else {
		logger.Debug("Login with phone: " + username)
		d = DB.DB.Where("phone = ?", username).First(&u)
	}

	return u, d.Error
}

func UpdateUserByID(uid uint64, u *UserModel) (*UserModel, error) {
	var ou UserModel
	d := DB.DB.First(&ou, uid)
	ou.Nickname = u.Nickname
	d.Save(&ou)
	ou.Password = "******"
	return &ou, d.Error
}

func GetUserByID(uid uint64) (*UserModel, error) {
	var d *gorm.DB
	u := &UserModel{}

	d = DB.DB.First(&u, uid)

	return u, d.Error
}

func ListUser(offset, limit int) ([]*UserModel, int64, error) {
	if limit == 0 {
		limit = constvar.DefaultLimit
	}

	users := make([]*UserModel, 0)
	var count int64
	if err := DB.DB.Model(&UserModel{}).Count(&count).Error; err != nil {
		return users, count, err
	}

	if err := DB.DB.Offset(offset).Limit(limit).Order("id desc").Find(&users).Error; err != nil {
		return users, count, err
	}
	return users, count, nil
}
