package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	Result string `json:"result,omitempty"`
}

// Request params
type Request struct {
	Data string `json:"data,omitempty"`
}

// Params
type Params struct {
	ID     string      `json:"id,omitempty"`
	Params interface{} `json:"params,omitempty"`
}

// Start the server
func (m *Meta) Start() {
	// Generate encryption keys
	publicKey, privatekey := generatePKI()
	m.ServerPublicKey = publicKey
	m.serverPrivateKey = privatekey
	clientPublicKey := os.Getenv("CLIENT_PUBLICKEY")
	if clientPublicKey != "" {
		block, _ := pem.Decode([]byte(clientPublicKey))
		key, _ := x509.ParsePKCS1PublicKey(block.Bytes)
		m.clientPublicKey = key
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
	fs := http.FileServer(http.Dir("./upload"))
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

	// generate self-sing key
	// crt := os.Getenv("APP_PATH") + "/ssl/api-server.crt"
	// key := os.Getenv("APP_PATH") + "/ssl/api-server.key"
	// err = GenerateSelfSignedCert(crt, key)
	// if err != nil {
	// 	log.Fatal(fmt.Println(err))
	// }

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
	req := Params{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		derr := errors.New("invalid payload request")
		w.WriteHeader(http.StatusBadRequest)
		m.Error(w, r, derr)
		return nil, err
	}

	if m.clientPublicKey != nil {
		request := Request{}
		err = json.Unmarshal(body, &request)
		if err != nil {
			derr := errors.New("invalid payload request")
			w.WriteHeader(http.StatusBadRequest)
			m.Error(w, r, derr)
			return nil, err
		}

		req, err = serverDecrypt(request.Data, m.serverPrivateKey)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			m.Error(w, r, err)
			return nil, err
		}
	} else {
		err = json.Unmarshal(body, &req)
		if err != nil {
			derr := errors.New("invalid payload request")
			w.WriteHeader(http.StatusBadRequest)
			m.Error(w, r, derr)
			return nil, err
		}
	}

	err = m.checkRequestId(req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		m.Error(w, r, err)
		return nil, err
	}
	return req.Params, nil
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
			Result: result,
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
func serverDecrypt(cipherText string, key *rsa.PrivateKey) (Params, error) {
	params := Params{}
	ct, _ := base64.StdEncoding.DecodeString(cipherText)
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, key, ct, label)
	if err != nil {
		derr := errors.New("invalid payload request")
		return params, derr
	}

	// validate special characters
	var data = string(plaintext)
	var payload = regexp.MustCompile(`^[a-zA-Z0-9_]*$`)
	var isValid = payload.MatchString(data)
	if !isValid {
		derr := errors.New("invalid payload request")
		return params, derr
	}

	err = json.Unmarshal(plaintext, &params)
	if err != nil {
		derr := errors.New("invalid payload request")
		return params, derr
	}
	return params, nil
}

// generatePKI provides encryption keys
func generatePKI() (*rsa.PublicKey, *rsa.PrivateKey) {
	// generate key
	privatekey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Printf("Cannot generate RSA key\n")
		os.Exit(1)
	}

	return &privatekey.PublicKey, privatekey
}

// encrypt payload
func serverEncrypt(secretMessage string, key *rsa.PublicKey) (string, error) {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, key, []byte(secretMessage), label)
	if err != nil {
		derr := errors.New("invalid payload request")
		return "", derr
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Connect method make a database connection
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

// GenerateSelfSignedCert creates a self-signed certificate and key for the given host.
// Host may be an IP or a DNS name
// The certificate will be created with file mode 0644. The key will be created with file mode 0600.
// If the certificate or key files already exist, they will be overwritten.
// Any parent directories of the certPath or keyPath will be created as needed with file mode 0755.
func GenerateSelfSignedCert(certPath, keyPath string) error {
	host := os.Getenv("SERVER_HOST")
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("%s@%d", host, time.Now().Unix()),
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 365),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		template.DNSNames = append(template.DNSNames, host)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Generate cert
	certBuffer := bytes.Buffer{}
	if err := pem.Encode(&certBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	// Generate key
	keyBuffer := bytes.Buffer{}
	if err := pem.Encode(&keyBuffer, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return err
	}

	// Write cert
	if err := os.MkdirAll(filepath.Dir(certPath), os.FileMode(0755)); err != nil {
		return err
	}
	if err := ioutil.WriteFile(certPath, certBuffer.Bytes(), os.FileMode(0644)); err != nil {
		return err
	}

	// Write key
	if err := os.MkdirAll(filepath.Dir(keyPath), os.FileMode(0755)); err != nil {
		return err
	}
	if err := ioutil.WriteFile(keyPath, keyBuffer.Bytes(), os.FileMode(0600)); err != nil {
		return err
	}

	return nil
}
