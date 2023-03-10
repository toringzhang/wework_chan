package wxbizmsgcrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const (
	ValidateSignatureError int = -40001
	ParseXmlError          int = -40002
	ComputeSignatureError  int = -40003
	IllegalAesKey          int = -40004
	ValidateCorpidError    int = -40005
	EncryptAESError        int = -40006
	DecryptAESError        int = -40007
	IllegalBuffer          int = -40008
	EncodeBase64Error      int = -40009
	DecodeBase64Error      int = -40010
	GenXmlError            int = -40010
	ParseJsonError         int = -40012
	GenJsonError           int = -40013
	IllegalProtocolType    int = -40014
)

type ProtocolType int

const (
	XmlType ProtocolType = 1
)

type CryptError struct {
	ErrCode int
	ErrMsg  string
}

func NewCryptError(errCode int, errMsg string) *CryptError {
	return &CryptError{ErrCode: errCode, ErrMsg: errMsg}
}

type WXBizMsg4Recv struct {
	ToUserName string `xml:"ToUserName"`
	Encrypt    string `xml:"Encrypt"`
	AgentId    string `xml:"AgentID"`
}

type CDATA struct {
	Value string `xml:",cdata"`
}

type WXBizMsg4Send struct {
	XMLName   xml.Name `xml:"xml"`
	Encrypt   CDATA    `xml:"Encrypt"`
	Signature CDATA    `xml:"MsgSignature"`
	Timestamp string   `xml:"TimeStamp"`
	Nonce     CDATA    `xml:"Nonce"`
}

func NewWXBizMsg4Send(encrypt, signature, timestamp, nonce string) *WXBizMsg4Send {
	return &WXBizMsg4Send{Encrypt: CDATA{Value: encrypt}, Signature: CDATA{Value: signature}, Timestamp: timestamp, Nonce: CDATA{Value: nonce}}
}

type ProtocolProcessor interface {
	parse(srcData []byte) (*WXBizMsg4Recv, *CryptError)
	serialize(msgSend *WXBizMsg4Send) ([]byte, *CryptError)
}

type WXBizMsgCrypt struct {
	token             string
	encodingAesKey    string
	receiverId        string
	protocolProcessor ProtocolProcessor
}

type XmlProcessor struct {
}

func (xp *XmlProcessor) parse(data []byte) (*WXBizMsg4Recv, *CryptError) {
	var msg4Recv WXBizMsg4Recv
	err := xml.Unmarshal(data, &msg4Recv)
	if nil != err {
		return nil, NewCryptError(ParseXmlError, "xml to msg fail")
	}
	return &msg4Recv, nil
}

func (xp *XmlProcessor) serialize(msg4Send *WXBizMsg4Send) ([]byte, *CryptError) {
	xmlMsg, err := xml.Marshal(msg4Send)
	if nil != err {
		return nil, NewCryptError(GenXmlError, err.Error())
	}
	return xmlMsg, nil
}

func NewWXBizMsgCrypt(token, encodingAesKey, receiverId string, protocolType ProtocolType) (*WXBizMsgCrypt, error) {
	var protocolProcessor ProtocolProcessor
	if protocolType != XmlType {
		return nil, errors.New("unsupported protocol")
	} else {
		protocolProcessor = new(XmlProcessor)
	}

	return &WXBizMsgCrypt{token: token, encodingAesKey: encodingAesKey + "=", receiverId: receiverId, protocolProcessor: protocolProcessor}, nil
}

func (w *WXBizMsgCrypt) randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func (w *WXBizMsgCrypt) pKCS7Padding(plaintext string, blockSize int) []byte {
	padding := blockSize - (len(plaintext) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	var buffer bytes.Buffer
	buffer.WriteString(plaintext)
	buffer.Write(padText)
	return buffer.Bytes()
}

func (w *WXBizMsgCrypt) pKCS7Unpadding(plaintext []byte, blockSize int) ([]byte, *CryptError) {
	plaintextLen := len(plaintext)
	if nil == plaintext || plaintextLen == 0 {
		return nil, NewCryptError(DecryptAESError, "pKCS7Unpadding error nil or zero")
	}
	if plaintextLen%blockSize != 0 {
		return nil, NewCryptError(DecryptAESError, "pKCS7Unpadding text not a multiple of the block size")
	}
	paddingLen := int(plaintext[plaintextLen-1])
	return plaintext[:plaintextLen-paddingLen], nil
}

func (w *WXBizMsgCrypt) cbcEncrypter(plaintext string) ([]byte, *CryptError) {
	aesKey, err := base64.StdEncoding.DecodeString(w.encodingAesKey)
	if nil != err {
		return nil, NewCryptError(DecodeBase64Error, err.Error())
	}
	const blockSize = 32
	padMsg := w.pKCS7Padding(plaintext, blockSize)

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, NewCryptError(EncryptAESError, err.Error())
	}

	ciphertext := make([]byte, len(padMsg))
	iv := aesKey[:aes.BlockSize]

	mode := cipher.NewCBCEncrypter(block, iv)

	mode.CryptBlocks(ciphertext, padMsg)
	base64Msg := make([]byte, base64.StdEncoding.EncodedLen(len(ciphertext)))
	base64.StdEncoding.Encode(base64Msg, ciphertext)

	return base64Msg, nil
}

func (w *WXBizMsgCrypt) cbcDecrypter(base64EncryptMsg string) ([]byte, *CryptError) {
	aesKey, err := base64.StdEncoding.DecodeString(w.encodingAesKey)
	if nil != err {
		return nil, NewCryptError(DecodeBase64Error, err.Error())
	}

	encryptMsg, err := base64.StdEncoding.DecodeString(base64EncryptMsg)
	if nil != err {
		return nil, NewCryptError(DecodeBase64Error, err.Error())
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, NewCryptError(DecryptAESError, err.Error())
	}

	if len(encryptMsg) < aes.BlockSize {
		return nil, NewCryptError(DecryptAESError, "encryptMsg size is not valid")
	}

	iv := aesKey[:aes.BlockSize]

	if len(encryptMsg)%aes.BlockSize != 0 {
		return nil, NewCryptError(DecryptAESError, "encryptMsg not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	mode.CryptBlocks(encryptMsg, encryptMsg)

	return encryptMsg, nil
}

func (w *WXBizMsgCrypt) calSignature(timestamp, nonce, data string) string {
	sortArr := []string{w.token, timestamp, nonce, data}
	sort.Strings(sortArr)
	var buffer bytes.Buffer
	for _, value := range sortArr {
		buffer.WriteString(value)
	}

	sha := sha1.New()
	sha.Write(buffer.Bytes())
	signature := fmt.Sprintf("%x", sha.Sum(nil))
	return signature
}

func (w *WXBizMsgCrypt) ParsePlainText(plaintext []byte) ([]byte, uint32, []byte, []byte, *CryptError) {
	const blockSize = 32
	plaintext, err := w.pKCS7Unpadding(plaintext, blockSize)
	if nil != err {
		return nil, 0, nil, nil, err
	}

	textLen := uint32(len(plaintext))
	if textLen < 20 {
		return nil, 0, nil, nil, NewCryptError(IllegalBuffer, "plain is to small 1")
	}
	random := plaintext[:16]
	msgLen := binary.BigEndian.Uint32(plaintext[16:20])
	if textLen < (20 + msgLen) {
		return nil, 0, nil, nil, NewCryptError(IllegalBuffer, "plain is to small 2")
	}

	msg := plaintext[20 : 20+msgLen]
	receiverId := plaintext[20+msgLen:]

	return random, msgLen, msg, receiverId, nil
}

func (w *WXBizMsgCrypt) VerifyURL(msgSignature, timestamp, nonce, echoStr string) ([]byte, *CryptError) {
	signature := w.calSignature(timestamp, nonce, echoStr)

	if strings.Compare(signature, msgSignature) != 0 {
		return nil, NewCryptError(ValidateSignatureError, "signature not equal")
	}

	plaintext, err := w.cbcDecrypter(echoStr)
	if nil != err {
		return nil, err
	}

	_, _, msg, receiverId, err := w.ParsePlainText(plaintext)
	if nil != err {
		return nil, err
	}

	if len(w.receiverId) > 0 && strings.Compare(string(receiverId), w.receiverId) != 0 {
		fmt.Println(string(receiverId), w.receiverId, len(receiverId), len(w.receiverId))
		return nil, NewCryptError(ValidateCorpidError, "receiverId is not equal")
	}

	return msg, nil
}

func (w *WXBizMsgCrypt) EncryptMsg(replyMsg, timestamp, nonce string) ([]byte, *CryptError) {
	randStr := w.randString(16)
	var buffer bytes.Buffer
	buffer.WriteString(randStr)

	msgLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLenBuf, uint32(len(replyMsg)))
	buffer.Write(msgLenBuf)
	buffer.WriteString(replyMsg)
	buffer.WriteString(w.receiverId)

	tmpCiphertext, err := w.cbcEncrypter(buffer.String())
	if nil != err {
		return nil, err
	}
	ciphertext := string(tmpCiphertext)

	signature := w.calSignature(timestamp, nonce, ciphertext)

	msg4Send := NewWXBizMsg4Send(ciphertext, signature, timestamp, nonce)
	return w.protocolProcessor.serialize(msg4Send)
}

func (w *WXBizMsgCrypt) DecryptMsg(msgSignature, timestamp, nonce string, postData []byte) ([]byte, *CryptError) {
	msg4Recv, cryptErr := w.protocolProcessor.parse(postData)
	if nil != cryptErr {
		return nil, cryptErr
	}

	signature := w.calSignature(timestamp, nonce, msg4Recv.Encrypt)

	if strings.Compare(signature, msgSignature) != 0 {
		return nil, NewCryptError(ValidateSignatureError, "signature not equal")
	}

	plaintext, cryptErr := w.cbcDecrypter(msg4Recv.Encrypt)
	if nil != cryptErr {
		return nil, cryptErr
	}

	_, _, msg, receiverId, cryptErr := w.ParsePlainText(plaintext)
	if nil != cryptErr {
		return nil, cryptErr
	}

	if len(w.receiverId) > 0 && strings.Compare(string(receiverId), w.receiverId) != 0 {
		return nil, NewCryptError(ValidateCorpidError, "receiver_id is not equil")
	}

	return msg, nil
}
