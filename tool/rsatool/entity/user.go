package entity

import "github.com/jinzhu/gorm"

type User struct {
	gorm.Model
	Company string `json:"company"`
	Name 	string `json:"name"`
	PrivKey string `json:"priv_key"`
	PubKey  string `json:"pub_key"`
}
