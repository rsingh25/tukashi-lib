package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"runtime/debug"

	"github.com/rsingh25/tukashi-lib/database"
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

func writeInternalServerError(w http.ResponseWriter, r *http.Request, message string) {
	WriteJsonResponse(w, r, http.StatusInternalServerError, map[string]string{
		"message": message,
		"traceID": "",
	})
}

// encoding error is not retured but handled in the function itself.
func WriteJsonResponse[T any](w http.ResponseWriter, r *http.Request, status int, v T, headers ...http.Header) {
	if len(headers) > 0 {
		maps.Copy(w.Header(), headers[0])
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		appLog.Error("encode error", "error", err.Error(), "method", r.Method, "url", r.URL, "stack", string(debug.Stack()))
	}
}

// Exec converts an error-returning handler to a standard http.HandlerFunc.
// It creates a db transaction if required and provides a query wrapper.
func Exec[RespType any](f func(*http.Request, *database.Queries) Resp[RespType], db database.Service, withTx bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var qtx *database.Queries
		var tx *sql.Tx
		var err error

		if withTx {
			tx, qtx, err = db.BeginTx(r.Context(), nil)
			if err != nil {
				panic(err)
			}
			defer tx.Rollback()
		} else {
			qtx = db.Queries()
		}

		resp := f(r, qtx)

		if resp.Err != nil {
			//TODO Handle error types
			panic(resp.Err)
		} else {
			if withTx {
				tx.Commit()
			}
			WriteJsonResponse(w, r, resp.Status, resp.Val)
			return
		}
	}
}

// ValidateReqExec converts an error-returning handler to a standard http.HandlerFunc.
// It validates and decodes the request into an onject.
// It creates a db transaction if required and provides a query wrapper.
func ValidateReqExec[RespType any, ReqType Validator](f func(ReqType, *http.Request, *database.Queries) Resp[RespType], db database.Service, withTx bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, problems, err := DecodeValid[ReqType](r)

		if len(problems) > 0 {
			WriteJsonResponse(w, r, http.StatusUnprocessableEntity, problems)
			return
		} else if err != nil {
			panic(err)
		}

		var qtx *database.Queries
		var tx *sql.Tx

		if withTx {
			tx, qtx, err = db.BeginTx(r.Context(), nil)
			if err != nil {
				panic(err)
			}
			defer tx.Rollback()
		} else {
			qtx = db.Queries()
		}

		resp := f(body, r, qtx)

		if resp.Err != nil {
			//TODO Handle error types
			panic(resp.Err)
		} else {
			if withTx {
				tx.Commit()
			}
			WriteJsonResponse(w, r, resp.Status, resp.Val)
			return
		}
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
