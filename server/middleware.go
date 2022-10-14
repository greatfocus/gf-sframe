package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
)

// Order of the Middleware
// 1. set headers
// 2. check Cors
// 3. check Limits Rates
// 4. check Allowed Ip Ranges
// 5. preflight
// 6. Check Permissions
// 7. CheckAuth/WithoutAuth
// 8. CheckProcessTimeout

// SetHeaders // prepare header response
func SetHeaders() Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			(w).Header().Set("Content-Type", "application/json")
			(w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-JWT, Authorization, request-id")

			// continue
			h.ServeHTTP(w, r)
		})
	}
}

// Preflight validates request for jwt header
func Preflight() Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if (*r).Method == "OPTIONS" {
				(w).WriteHeader(http.StatusOK)
				return
			}

			// continue
			h.ServeHTTP(w, r)
		})
	}
}

// enable cors within the http handler
func CheckCors(meta *Meta) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			allowed := false
			origin := r.Header.Get("Origin")

			// check if cors is available in list
			origins := strings.Split(os.Getenv("ALLOWED_ORIGIN"), ",")
			for _, v := range origins {
				if v == origin {
					allowed = true
				}
			}

			// allow cors if found
			if !allowed {
				(w).Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				(w).Header().Set("Access-Control-Allow-Origin", origin)
			}

			// continue
			h.ServeHTTP(w, r)
		})
	}
}

// CheckLimitsRates handle limits and rates
func CheckLimitsRates() Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// limit us requests per second
			limiter := limiter.GetLimiter(limiter.getIP(r))
			if !limiter.Allow() {
				(w).WriteHeader(http.StatusTooManyRequests)
				return
			}

			// continue
			h.ServeHTTP(w, r)
		})
	}
}

// CheckAllowedIPRange allow specific IP address
func CheckAllowedIPRange(meta *Meta) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			allowed := false
			ip := limiter.getIP(r)
			// check if ip is available in list
			ips := strings.Split(os.Getenv("ALLOWED_IP"), ",")
			for _, v := range ips {
				if v == ip {
					allowed = true
				}
			}
			// allow ip if found
			if !allowed {
				(w).WriteHeader(http.StatusForbidden)
				return
			}

			// continue
			h.ServeHTTP(w, r)
		})
	}
}

// CheckPermission validate if users is allowed to access route
func CheckPermission(meta *Meta) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var allowed bool
			var pattern = r.URL.Path
			token, err := meta.JWT.GetToken(r)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				meta.Error(w, r, errors.New("Unauthorized"))
				return
			}

			for _, value := range token.Permissions {
				if value == pattern {
					allowed = true
				}
			}

			if allowed {
				w.WriteHeader(http.StatusUnauthorized)
				meta.Error(w, r, errors.New("Unauthorized"))
				return
			}

			// continue
			h.ServeHTTP(w, r)
		})
	}
}

// CheckAuth validates request for jwt header
func CheckAuth(meta *Meta) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// validate jwt
			err := meta.JWT.TokenValid(r)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				meta.Error(w, r, errors.New("Unauthorized"))
				return
			}

			// continue
			h.ServeHTTP(w, r)
		})
	}
}

// WithoutAuth access without authentications
func WithoutAuth() Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// continue
			h.ServeHTTP(w, r)
		})
	}
}

// CheckProcessTimeout put a time limit for the handler process duration and will give an error response if timeout
func CheckProcessTimeout(meta *Meta) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), time.Duration(meta.Timeout)*time.Second)
			defer cancel()

			r = r.WithContext(ctx)

			processDone := make(chan bool)
			go func() {
				h.ServeHTTP(w, r)
				processDone <- true
			}()

			select {
			case <-ctx.Done():
				meta.Logger.ErrorLogger.Println([]byte(`{"error": "process timeout"}`))
			case <-processDone:
			}
		})
	}
}

// Middleware strct
type Middleware func(http.Handler) http.Handler

// Use middleware
func Use(h http.Handler, middlewares ...Middleware) http.Handler {
	for _, m := range middlewares {
		h = m(h)
	}
	return h
}
