package server

import (
	"crypto/x509"
	"net/http"
	"time"

	"github.com/google/uuid"

	sframeModel "github.com/greatfocus/gf-sframe/model"
)

// Info struct
type Info struct {
	MetaHandler func(http.ResponseWriter, *http.Request)
	meta        *Meta
}

// Init method
func (i *Info) Init(meta *Meta) {
	i.meta = meta
}

func (i Info) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		i.getInfo(w, r)
		return
	}

	// catch all
	// if no method is satisfied return an error
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Header().Add("Allow", "GET, POST, PUT, DELETE")
}

// getInfo method
func (i *Info) getInfo(w http.ResponseWriter, r *http.Request) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(i.meta.ServerPublicKey)
	if err != nil {
		i.meta.Logger.ErrorLogger.Printf("Error: %v\n", err)
		w.WriteHeader(http.StatusExpectationFailed)
		i.meta.Error(w, r, err)
		return
	}

	uuid := uuid.New().String()
	res := sframeModel.Meta{
		PublicKey: string(publicKeyBytes),
		RequestID: uuid,
	}
	i.meta.Cache.Set(uuid, uuid, time.Duration(1))
	i.meta.Success(w, r, res)
}
