package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// Response : http response func
func Response(w http.ResponseWriter, r *http.Request, data interface{}, httpCode int) {
	switch data.(type) {
	case string:
		w.WriteHeader(httpCode)
		w.Write([]byte(data.(string)))
		return
	default:
		jsonByte, err := json.Marshal(data)
		if err != nil {
			log.Printf("marshal data failed, %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpCode)
		w.Write(jsonByte)
	}
}

// ResponseOk : http response func
func ResponseOk(w http.ResponseWriter, r *http.Request, msg string, errCode int) {
	type resp struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	jsonByte, _ := json.Marshal(resp{
		ErrCode: errCode,
		ErrMsg:  msg,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(jsonByte))
}
