package main

import (
	"flag"
	"log"
	"net/http"

	server "github.com/toringzhang/wework_chan/server/wework"
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
	http.HandleFunc("/api/v1/verify", s.VerifyMessage)

	log.Printf("server at %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
