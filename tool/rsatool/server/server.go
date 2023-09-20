package server

import (
	"encoding/json"
	"errors"
	"github.com/rongzer/blockchain/tool/rsatool/entity"
	rsa2 "github.com/rongzer/blockchain/tool/rsatool/rsa"
	"github.com/rongzer/blockchain/tool/rsatool/sqlite"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	certPath	 = ""
	formatString = "2006-01-02 15:04:05"
	adminCompany = "rongzer"
	adminUser	 = "admin"
	codes		 = "codes.txt"
)

type Server struct {
	sql *sqlite.UserSql
}

type Resp struct {
	Code string  `json:"code"`
	Data string  `json:"data"`
}

func NewServer(sql *sqlite.UserSql) *Server {
	return &Server{sql: sql}
}

// 用户注册接口  创建证书并写进数据库
func (s *Server)RegistUser(w http.ResponseWriter, r *http.Request)  {
	rp := Resp{"200", "user regist success"}
	var user entity.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	// 查询数据是否存在，存在驳回请求
	users, err := s.sql.QueryCompanyUsers(user.Company)
	if err == nil {
		for _, u := range users {
			if u.Name == user.Name {
				rp.Code = "500"
				rp.Data = errors.New("this user has been registered").Error()
				_ = json.NewEncoder(w).Encode(rp)
				return
			}
		}
	}

	path := filepath.Join(certPath, user.Company, user.Name)
	rsa := rsa2.NewRsa(path)
	priv, err := rsa.GetPrivateKey()
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	user.PrivKey = priv

	pub, err := rsa.GetPublicKey()
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	user.PubKey = pub
	s.sql.CreateUser(user)

	_ = json.NewEncoder(w).Encode(rp)
}

func (s *Server) GetUser(w http.ResponseWriter, r *http.Request) {

	_ = r.ParseForm()
	rp := Resp{"200", "user user success"}
	values := r.Form["name"]
	if len(values) <= 0 {
		rp.Code = "500"
		rp.Data = errors.New("empty name is invalid").Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	name := values[0]

	user, err := s.sql.QueryUser(name)
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	userBytes, err := json.Marshal(user)
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	rp.Data = string(userBytes)
	_ = json.NewEncoder(w).Encode(rp)
}

func (s *Server) GetCompanyUsers(w http.ResponseWriter, r *http.Request) {

	_ = r.ParseForm()
	rp := Resp{"200", "user company user success"}
	values :=  r.Form["company"]
	if len(values) <= 0 {
		rp.Code = "500"
		rp.Data = errors.New("empty company is invalid").Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}
	company := values[0]

	users, err := s.sql.QueryCompanyUsers(company)
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	usersBytes, err := json.Marshal(users)
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	rp.Data = string(usersBytes)
	_ = json.NewEncoder(w).Encode(rp)
}

func (s *Server) GetActiveCode(w http.ResponseWriter, r *http.Request) {

	_ = r.ParseForm()
	rp := Resp{"200", "get active code success"}

	companys := r.Form["company"]
	names := r.Form["name"]
	durations := r.Form["duration"]
	if len(companys)<= 0 || len(names) <= 0 || len(durations) <= 0{
		rp.Code = "500"
		rp.Data = errors.New("empty company or name or duration is empty").Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	company := companys[0]
	name := names[0]
	duration := durations[0]

	log.Print("input company : ", company, " name : ", name, " duration : ", duration)

	d, err := time.ParseDuration(duration)
	if err != nil {
		rp.Code = "500"
		rp.Data = errors.New("duration is invalid " + err.Error()).Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	now := time.Now()
	end := now.Add(d)

	nowFormat := now.Format(formatString)
	endFormat := end.Format(formatString)

	timeFormat := nowFormat + "~" + endFormat
	log.Print(timeFormat)

	// 获取当前用户
	user, err := s.getCompanyUser(company, name)
	if err != nil || user == nil{
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	log.Print("user company : ", user.Company, " user name : ", user.Name)

	// 获取admin 用户进行签名
	admin, err := s.getCompanyUser(adminCompany, adminUser)
	if err != nil || admin == nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	adminPath := filepath.Join(certPath, adminCompany, adminUser)
	adminRsa := rsa2.NewRsa(adminPath)
	_, err = adminRsa.GetPrivateKey()
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	userPath := filepath.Join(certPath, user.Company, user.Name)
	userRsa := rsa2.NewRsa(userPath)
	_, err = userRsa.GetPrivateKey()
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	encryptoMsg, err := userRsa.Encrypto([]byte(timeFormat))
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	sig, err := adminRsa.SignData(encryptoMsg)
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	//log.Print("sig : ", string(sig))
	verifyCode := string(encryptoMsg) + "+" + string(sig)
	//log.Print("verifyCode : ", verifyCode)

	verifyCodePath := filepath.Join(certPath, user.Company, user.Name, codes)
	log.Print("path : ", verifyCodePath)
	file, err := os.Create(verifyCodePath)
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	_, err = file.WriteString(verifyCode)
	if err != nil {
		rp.Code = "500"
		rp.Data = err.Error()
		_ = json.NewEncoder(w).Encode(rp)
		return
	}

	file.Sync()
	defer file.Close()

	rp.Data = verifyCode
	_ = json.NewEncoder(w).Encode(rp)
}

func (s *Server) getCompanyUser(company string, name string) (*entity.User, error){

	users, err := s.sql.QueryCompanyUsers(company)
	if err != nil {
		return nil, err
	}

	var user *entity.User
	for _, u := range users {
		if u.Name == name {
			user = u
		}
	}

	if user == nil {
		return nil, errors.New("can not find the user with name " + name)
	}

	return user, nil
}