package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/justinas/nosurf"
)

type OidcClaims struct {
	Sub         string `json:"sub,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Iss         string `json:"iss,omitempty"`
	Exp         int    `json:"exp,omitempty"`
	Username    string `json:"username,omitempty"`
	Email       string `json:"email,omitempty"`
	AttmgtRole  string `json:"custom:attmgt,omitempty"`
	Name        string `json:"name,omitempty"`
}

type TraceID struct{}
type UserEmail struct{}
type UserName struct{}
type UserPhone struct{}
type AttmgtRole struct{}
type Middleware func(http.Handler) http.Handler

// NewMwChain(m1, m2, m3)(myHandler) will chained as m1(m2(m3(myHandler)))
func NewMwChain(mw ...Middleware) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		next := h
		for k := len(mw) - 1; k >= 0; k-- {
			next = mw[k](next)
		}
		return next
	}
}

func NewMwChainFunc(mw ...Middleware) func(http.HandlerFunc) http.Handler {
	return func(h http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler := h
			for k := len(mw) - 1; k >= 0; k-- {
				curH := mw[k]
				nextH := handler
				// update the chain
				handler = func(w http.ResponseWriter, r *http.Request) {
					curH(nextH).ServeHTTP(w, r)
				}
			}
			// Execute the assembled processor chain
			handler.ServeHTTP(w, r)
		})
	}
}

/* TO BE TAKEN CARE BY APIGW
func WithCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Replace "*" with specific origins if needed
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false") // Set to "true" if credentials are required

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Proceed with the next handler
		next.ServeHTTP(w, r)
	})
}
*/

func WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		appLog.Debug("Http Req Started", "method", r.Method, "url", r.URL)
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		appLog.Debug("Http Req Served", "method", r.Method, "url", r.URL, "duraton", duration)
	})
}

/* THIS IS NOT POSSIBLE IN LAMBDA
// ServerSentEventsLogging: This will log a message with initial http request and when response is closed.
func WithSseLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Debug("SSE Req Revieved", "method", r.Method, "url", r.URL)
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		logger.Debug("SSE Req Completed", "method", r.Method, "url", r.URL, "duration", duration)
	})
}
*/

/*
func WithTime(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		orgContext := r.Context()
		newContext := context.WithValue(orgContext, "time", &t)
		newRequest := r.WithContext(newContext)
		next.ServeHTTP(w, newRequest)
	})
}
*/

func WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//apiKey := r.Header.Get("Authorization")
		//if apiKey != "api-key-test" {
		//	http.Error(w, "invalid api-key", http.StatusForbidden)
		//	return
		//}
		next.ServeHTTP(w, r)
	})
}

func WithApiKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h1 := r.Header.Get("X-API-KEY")
			if h1 != apiKey {
				http.Error(w, "invalid api-key", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// This middleware is applicabel to request coming from AWS LB.
func WithAlbAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		oidcEncoded := strings.Split(r.Header.Get("X-Amzn-Oidc-Data"), ".")[1]

		oidcDecoded, err := base64.StdEncoding.DecodeString(oidcEncoded)
		if err != nil {
			appLog.Error("Error decoding oidcData:", "err", err)
			http.Error(w, "invalid token", http.StatusForbidden)
			return
		}

		var oidcClaim OidcClaims
		err = json.Unmarshal(oidcDecoded, &oidcClaim)
		if err != nil {
			http.Error(w, "invalid token", http.StatusForbidden)
			appLog.Error("Error unmarshalling oidcData:", "err", err)
			return
		}

		ctx := context.WithValue(r.Context(), TraceID{}, r.Header.Get("X-Amzn-Trace-Id"))
		ctx = context.WithValue(ctx, UserEmail{}, oidcClaim.Email)
		ctx = context.WithValue(ctx, UserName{}, oidcClaim.Name)
		ctx = context.WithValue(ctx, UserPhone{}, oidcClaim.PhoneNumber)
		ctx = context.WithValue(ctx, AttmgtRole{}, oidcClaim.AttmgtRole)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// WithMsg middleware with decorators
func WithMsg(msg string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("Example Message:", msg)
			next.ServeHTTP(w, r)
		})
	}
}

// NoSurf is the csrf protection middleware
func WithNoSurf(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		csrfHandler := nosurf.New(next)

		csrfHandler.SetBaseCookie(http.Cookie{
			HttpOnly: true,
			Path:     "/",
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
		return csrfHandler
	}
}

// This is similar to http.TimeoutHandler() but does not send a 503 with html payload
func WithTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}

func WithPanicRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				buf := make([]byte, 10<<10)
				n := runtime.Stack(buf, false)
				appLog.Error("Panic recovered",
					"method", r.Method,
					"url", r.URL,
					"err", err,
					"trace-id", r.Context().Value(TraceID{}),
					"strack-trace", string(buf[:n]))

				message := "Internal Server Error"
				switch v := err.(type) {
				case string:
					message = v
				case error:
					message = v.Error() //TODO leaking the error message
				}
				write5xxResponse(w, r, message)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
