package web

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"runtime/debug"
)

// Validator is an object that can be validated.
type Validator interface {
	// Valid checks the object and returns any problems. If len(problems) == 0 then the object is valid.
	Valid(ctx context.Context) (problems map[string]string)
}

type Resp[T any] struct {
	Val    T
	Err    error
	Status int
}

func write5xxResponse(w http.ResponseWriter, r *http.Request, message string, headers ...http.Header) {
	WriteJsonResponse(w, r, http.StatusInternalServerError, map[string]string{
		"message": message,
		"traceID": "",
	}, headers...)
}

func Write4xxResponse(w http.ResponseWriter, r *http.Request, status int, problems map[string]string, headers ...http.Header) {
	WriteJsonResponse(w, r, status, problems, headers...)
}

// encoding error is not retured but handled in the function itself.
func WriteJsonResponse[T any](w http.ResponseWriter, r *http.Request, status int, v T, headers ...http.Header) {
	if len(headers) > 0 {
		maps.Copy(w.Header(), headers[0])
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("App-Trace-ID", r.Context().Value(TraceID{}).(string))

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		appLog.Error("encode error", "error", err.Error(), "method", r.Method, "url", r.URL, "stack", string(debug.Stack()))
	}
}

func DecodeValid[T Validator](r *http.Request) (T, map[string]string, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, nil, fmt.Errorf("decode json: %w", err)
	}
	if problems := v.Valid(r.Context()); len(problems) > 0 {
		return v, problems, fmt.Errorf("invalid %T: %d problems", v, len(problems))
	}
	return v, nil, nil
}
