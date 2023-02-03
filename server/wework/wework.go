package server

import (
	"log"
	"net/http"

	"github.com/toringzhang/wework_chan/pkg/wxbizmsgcrypt"
)

type wework struct {
	// token
	token string
	// encodingAESKey
	encodingAESKey string
	// receiverId 企业ID
	receiverId string
	wxbmc      *wxbizmsgcrypt.WXBizMsgCrypt
}

type Interface interface {
	VerifyMessage(w http.ResponseWriter, r *http.Request)
}

func NewWework(token, encodingAESKey, receiverId string) Interface {
	wxbmc, err := wxbizmsgcrypt.NewWXBizMsgCrypt(token, encodingAESKey, receiverId, wxbizmsgcrypt.XmlType)
	if err != nil {
		log.Printf("NewWXBizMsgCrypt failed, %v", err)
		return nil
	}
	return &wework{
		token:          token,
		encodingAESKey: encodingAESKey,
		receiverId:     receiverId,
		wxbmc:          wxbmc,
	}
}

func (ww *wework) VerifyMessage(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	msgSignature := r.FormValue("msg_signature")
	timestamp := r.FormValue("timestamp")
	nonce := r.FormValue("nonce")
	echoStr := r.FormValue("echostr")
	echoData, cryptErr := ww.wxbmc.VerifyURL(msgSignature, timestamp, nonce, echoStr)
	if cryptErr != nil {
		ResponseOk(w, r, cryptErr.ErrMsg, cryptErr.ErrCode)
		return
	}
	Response(w, r, string(echoData), http.StatusOK)
}
