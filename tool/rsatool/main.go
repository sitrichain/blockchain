package main

import (
	"github.com/rongzer/blockchain/tool/rsatool/server"
	"github.com/rongzer/blockchain/tool/rsatool/sqlite"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

const (
	ip = ":8880"
)

var mainCmd = &cobra.Command{
	Use: "rsatool",
	Short: "base command",
	Long: "Evenry software has main cmd, same with rsatool",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("rsatool v0.9 -- HEAD")
	},
}

func main() {

	//if mainCmd.Execute() != nil {
	//	os.Exit(1)
	//}

	// server 启动
	sql := &sqlite.UserSql{}
	err := sql.Open()
	if err != nil {
		log.Fatal(err)
	}

	s := server.NewServer(sql)

	http.HandleFunc("/regUser", s.RegistUser)
	http.HandleFunc("/getUser", s.GetUser)
	http.HandleFunc("/getCompanyUsers", s.GetCompanyUsers)
	http.HandleFunc("/getActiveCode", s.GetActiveCode)
	log.Print("server start at :", ip)

	err = http.ListenAndServe(ip, nil)
	if err != nil {
		log.Fatal(err)
	}

	// licences 验证启动
	//l := licences.NewLicences("tencent")
	//err := l.CheckLicences()
	//if err != nil {
	//	os.Exit(-1)
	//}
	//
	//log.Print("licence activate success")

}
