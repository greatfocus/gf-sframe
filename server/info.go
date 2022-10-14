package server

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ServiceInfo struct
type ServiceInfo struct {
	ServiceName string `json:"serviceName,omitempty"`
	RequestID   string `json:"requestId,omitempty"`
}

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
	w.Header().Add("Allow", "GET")
}

// getInfo method
func (i *Info) getInfo(w http.ResponseWriter, r *http.Request) {
	uuid := uuid.New().String()
	res := ServiceInfo{
		ServiceName: i.meta.URI,
		RequestID:   uuid,
	}
	i.meta.Cache.Set(uuid, uuid, time.Duration(1))
	i.meta.Success(w, r, res)
}
