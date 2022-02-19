package server

import (
	"encoding/json"
	"net/http"
)

// Response data
type Response struct {
	Data string `json:"data,omitempty"`
}

// response returns payload
func response(w http.ResponseWriter, r *http.Request, data interface{}, message string) {
	body, _ := json.Marshal(data)
	res := Response{
		Data: string(body),
	}
	_ = json.NewEncoder(w).Encode(res)
}

// Success returns object as json
func Success(w http.ResponseWriter, r *http.Request, data interface{}) {
	if data != nil {
		response(w, r, data, "success")
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	response(w, r, nil, "success")
}

// Error returns error as json
func Error(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		response(w, r, struct {
			Error string `json:"error"`
		}{Error: err.Error()}, "error")
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	response(w, r, nil, "error")
}
