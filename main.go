package main

import (
	"flag"
	"github.com/toringzhang/wework_chan/server"
	"log"
	"net/http"
)

var (
	addr           string
	token          string
	encodingAESKey string
	corpID         string
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&addr, "addr", ":8080", "listen address")
	flag.StringVar(&token, "token", "", "Token")
	flag.StringVar(&encodingAESKey, "key", "", "EncodingAESKey")
	flag.StringVar(&corpID, "corp", "", "EncodingAESKey")
	flag.Parse()

	s := server.NewWework(token, encodingAESKey, corpID)
	http.HandleFunc("/api/v1/verify", server.LogHandler(s.VerifyMessage))
	// 反向代理，企业微信必须是可信IP
	http.HandleFunc("/cgi-bin/gettoken", server.LogHandler(server.ReverseProxyHandler("https://qyapi.weixin.qq.com")))
	http.HandleFunc("/cgi-bin/message/send", server.LogHandler(server.ReverseProxyHandler("https://qyapi.weixin.qq.com")))

	log.Printf("server at %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
