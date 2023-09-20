package sqlite

import (
	"errors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/rongzer/blockchain/tool/rsatool/entity"
)

type UserSql struct {
	db *gorm.DB
}

func (u *UserSql) Open() error {
	db, err := gorm.Open("sqlite3", "rsa.db")
	if err != nil {
		return err
	}

	db.AutoMigrate(&entity.User{})
	u.db = db
	return nil
}

func (u *UserSql) Close() error {

	return nil
}

func (u *UserSql) CreateUser(user entity.User) {
	u.db.Create(&user)
}

func (u *UserSql) UpdateUser(user entity.User) {
	u.db.Update(&user)
}

func (u *UserSql) DeleteUser(user entity.User)  {
	u.db.Delete(&user)
}

func (u *UserSql) QueryUser(name string) (*entity.User, error) {
	user := &entity.User{}
	u.db.Where("name = ?", name).First(user)
	if user.ID <= 0 {
		return nil, errors.New("not exist user with name")
	}

	return user, nil
}

func (u *UserSql) QueryCompanyUsers(name string) ([]*entity.User, error) {
	var users []*entity.User
	u.db.Where("company = ?", name).Find(&users)
	if len(users) <= 0 {
		return nil, errors.New("not exist user with company name")
	}

	return users, nil
}

