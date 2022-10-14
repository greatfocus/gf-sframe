package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	gfbus "github.com/greatfocus/gf-bus"
	gfcron "github.com/greatfocus/gf-cron"
	"github.com/greatfocus/gf-sframe/broker"
	"github.com/greatfocus/gf-sframe/cache"
	"github.com/greatfocus/gf-sframe/database"
	"github.com/greatfocus/gf-sframe/logger"
)

// HandlerFunc custom server handler
type HandlerFunc func(http.ResponseWriter, *http.Request)

// Meta struct
type Meta struct {
	Env              string
	URI              string
	Mux              *http.ServeMux
	DB               *database.Conn
	Cache            *cache.Cache
	Cron             *gfcron.Cron
	JWT              *JWT
	Bus              *gfbus.Bus
	Broker           *broker.Conn
	Logger           *logger.Logger
	clientPublicKey  *rsa.PublicKey
	ServerPublicKey  *rsa.PublicKey
	serverPrivateKey *rsa.PrivateKey
	Timeout          uint64
}

// Response result
type Response struct {
	Result    string `json:"result,omitempty"`
	CipherRes string `json:"cipherRes,omitempty"`
}

// Params
type Params struct {
	ID        string      `json:"id,omitempty"`
	Params    interface{} `json:"params,omitempty"`
	CipherReq string      `json:"cipherReq,omitempty"`
}

// Start the server
func (m *Meta) Start() {
	// Get encryption keys
	privatekey, publicKey := GetServerPKI()
	m.ServerPublicKey = publicKey
	m.serverPrivateKey = privatekey
	clientPublicKey := os.Getenv("CLIENT_PUBLICKEY")
	if clientPublicKey != "" {
		clientPublicKeyString, err := base64.StdEncoding.DecodeString(clientPublicKey)
		if err == nil {
			publicBlock, _ := pem.Decode([]byte(clientPublicKeyString))
			pubKey, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
			if err == nil {
				m.clientPublicKey = pubKey.(*rsa.PublicKey)
			}

		}
	}

	// setUploadPath creates an upload path
	m.setUploadPath()

	// set default handlers
	m.defaultHandler()

	// serve creates server instance
	m.serve()
}

// setUploadPath creates an upload path
func (m *Meta) setUploadPath() {
	path := os.Getenv("UPLOAD_PATH")
	if path == "" {
		path = "./data/upload"
	}
	fs := http.FileServer(http.Dir(path))
	fileLoc := "/" + m.URI + "/resource"
	m.Mux.Handle(fileLoc+"/", http.StripPrefix(fileLoc, fs))
}

// defaultHandler create default handlers
func (m *Meta) defaultHandler() {
	infoHandler := Info{}
	infoHandler.Init(m)
	m.Mux.Handle("/"+m.URI+"/meta", Use(infoHandler,
		SetHeaders(),
		CheckLimitsRates(),
		WithoutAuth()))
}

// serve creates server instance
func (m *Meta) serve() {
	timeout, err := strconv.ParseUint(os.Getenv("SERVER_TIMEOUT"), 0, 64)
	m.Timeout = timeout
	if err != nil {
		log.Fatal(fmt.Println(err))
	}

	addr := ":" + os.Getenv("SERVER_PORT")
	srv := &http.Server{
		Addr:           addr,
		ReadTimeout:    time.Duration(timeout) * time.Second,
		WriteTimeout:   time.Duration(timeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        m.Mux,
	}

	// Get key certificate
	m.Logger.InfoLogger.Println("Listening to port HTTP", addr)
	crt, key := GetServerCertificate()
	if crt != "" && key != "" {
		log.Fatal(srv.ListenAndServeTLS(crt, key))
	}
	log.Fatal(srv.ListenAndServe())
}

// Success returns object as json
func (m *Meta) Success(w http.ResponseWriter, r *http.Request, data interface{}) {
	if data != nil {
		m.response(w, r, data, "success")
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	m.response(w, r, nil, "success")
}

// Error returns error as json
func (m *Meta) Error(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		m.response(w, r, struct {
			Error string `json:"error"`
		}{Error: err.Error()}, "error")
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	m.response(w, r, nil, "error")
}

// request returns payload
func (m *Meta) Request(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		req := Params{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			derr := errors.New("invalid payload request")
			w.WriteHeader(http.StatusBadRequest)
			m.Error(w, r, derr)
			return nil, err
		}

		err = json.Unmarshal(body, &req)
		if err != nil {
			derr := errors.New("invalid payload request")
			w.WriteHeader(http.StatusBadRequest)
			m.Error(w, r, derr)
			return nil, err
		}

		err = m.checkRequestId(req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			m.Error(w, r, err)
			return nil, err
		}

		if m.clientPublicKey != nil {
			res, err := serverDecrypt(req.CipherReq, m.serverPrivateKey)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				m.Error(w, r, err)
				return nil, err
			}
			return res.Params, nil
		}
		return req.Params, nil
	}
	return nil, nil
}

// CheckRequestId validates requestID
func (m *Meta) checkRequestId(p Params) error {
	if p.ID == "" {
		return errors.New("invalid request")
	}

	_, found := m.Cache.Get(p.ID)
	if found {
		return errors.New("duplicate request")
	}

	m.Cache.Set(p.ID, p.Params, time.Duration(m.Timeout)*time.Second)
	return nil
}

// response returns payload
func (m *Meta) response(w http.ResponseWriter, r *http.Request, data interface{}, message string) {
	out, _ := json.Marshal(data)
	if m.clientPublicKey != nil {
		result, err := serverEncrypt(string(out), m.clientPublicKey)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			m.Error(w, r, err)
		}
		res := Response{
			CipherRes: result,
		}
		_ = json.NewEncoder(w).Encode(res)
	} else {
		res := Response{
			Result: string(out),
		}
		_ = json.NewEncoder(w).Encode(res)
	}

}

// decrypt payload
func serverDecrypt(cipherText string, privateKey *rsa.PrivateKey) (Params, error) {
	params := Params{}
	ct, _ := base64.StdEncoding.DecodeString(cipherText)
	unencrypted, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ct)
	if err != nil {
		derr := errors.New("invalid payload request")
		return params, derr
	}
	err = json.Unmarshal([]byte(unencrypted), &params)
	if err != nil {
		derr := errors.New("invalid payload request")
		return params, derr
	}
	return params, nil
}

// encrypt payload
func serverEncrypt(payload string, publicKey *rsa.PublicKey) (string, error) {
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, []byte(payload))
	if err != nil {
		derr := errors.New("invalid payload request")
		return "", derr
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// GetServerCertificate returns private and public key
func GetServerCertificate() (string, string) {
	var sslcert = os.Getenv("API_SSL_CERT")
	var sslkey = os.Getenv("API_SSL_KEY")

	// prepare ssl connection files
	if sslkey != "" && sslcert != "" {
		crt := database.CreateSSLCert("api-server.crt", sslcert)
		key := database.CreateSSLCert("api-server.key", sslkey)
		return crt, key
	}

	return "", ""
}

// GetServerPKI returns public key infrustructure
func GetServerPKI() (*rsa.PrivateKey, *rsa.PublicKey) {
	var privateKey = os.Getenv("API_PRIVATE_KEY")
	privateKeyString, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return nil, nil
	}
	privateBlock, _ := pem.Decode([]byte(privateKeyString))
	privKey, err := x509.ParsePKCS1PrivateKey(privateBlock.Bytes)
	if err != nil {
		return nil, nil
	}

	var publicKey = os.Getenv("API_PUBLIC_KEY")
	publicKeyString, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return nil, nil
	}
	publicBlock, _ := pem.Decode([]byte(publicKeyString))
	pubKey, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil {
		return nil, nil
	}

	return privKey, pubKey.(*rsa.PublicKey)
}
