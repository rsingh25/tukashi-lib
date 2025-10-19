package util

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	_ "time/tzdata"

	_ "github.com/joho/godotenv/autoload" //autoloads .env
)

var (
	appLog  *slog.Logger
	IST_LOC *time.Location
	err     error
)

var Logger *slog.Logger

func init() {
	var logLevel slog.Level

	logLevelStr := GetenvStr("LOG_LEVEL", "INFO")
	switch logLevelStr {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logHandleOpts := &slog.HandlerOptions{
		Level: logLevel,
	}

	defaultAttrs := []slog.Attr{
		slog.String("env", GetenvStr("APP_ENV", "")),
	}

	Logger = slog.New(slog.NewJSONHandler(os.Stdout, logHandleOpts).WithAttrs(defaultAttrs))
	slog.SetDefault(Logger)

	appLog = Logger.With("package", "util")

	IST_LOC, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		appLog.Error("Error loading location", "err", err)
		panic("Error loading location")
	}
}

func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func Must[T any](t T, err error) T {
	if err != nil {
		appLog.Error("Failed with error (panicing)", "error", err.Error(), "stack", debug.Stack())
		panic(err) //No recovery
	}
	return t
}

func Filter[T any](ts []T, f func(T) bool) []T {
	filtered := make([]T, 0, len(ts))
	for _, e := range ts {
		if f(e) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func GetenvInt(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Errorf("environment variable %s=%q cannot be converted to an int", key, value))
	}
	return intValue
}

func GetenvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Errorf("environment variable %s=%q cannot be converted to a bool", key, value))
	}
	return boolValue
}

func GetenvDuration(key string, defaultValue time.Duration) time.Duration {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	durationValue, err := time.ParseDuration(value)
	if err != nil {
		panic(fmt.Errorf("environment variable %s=%q cannot be converted to a time.Duration", key, value))
	}
	return durationValue
}

func GetenvStr(key string, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func MustGetenvInt(key string) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		panic(fmt.Errorf("environment variable %s must be set", key))
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Errorf("environment variable %s=%q cannot be converted to an int", key, value))
	}
	return intValue
}

func MustGetenvStr(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		panic(fmt.Errorf("environment variable %s must be set", key))
	}

	return value
}

func MustToJsonByte(obj any) []byte {
	jsonData, err := json.Marshal(obj)
	if err != nil {
		panic("ToJsonByte: cannot marshal")
	}
	return jsonData
}
