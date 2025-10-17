package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/rsingh25/tukashi-lib/util"

	_ "github.com/lib/pq"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	//Querries returns a pointer to the Queries struct, which contains methods for executing SQL queries.
	Queries() *Queries

	//starts a new transaction
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, *Queries, error)
}

type service struct {
	db *sql.DB
}

var (
	dbInstance *service
	appLog     *slog.Logger
)

func init() {
	appLog = util.Logger.With("package", "database")
}

func NewService() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}
	var db *sql.DB

	dbHost := util.MustGetenvStr("DB_HOST")
	dbPort := util.MustGetenvStr("DB_PORT")
	dbName := util.MustGetenvStr("DB_NAME")
	schema := util.MustGetenvStr("DB_SCHEMA")
	dbUser := util.MustGetenvStr("DB_USERNAME")
	password := util.MustGetenvStr("DB_PASSWORD")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?search_path=%s", dbUser, password, dbHost, dbPort, dbName, schema)
	db = util.Must(sql.Open("pgx", connStr))

	dbInstance = &service{
		db: db,
	}

	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		appLog.Error("db down:", "err", err) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	appLog.Debug("Disconnected from database")
	return s.db.Close()
}

func (s *service) Queries() *Queries {
	return New(s.db)
}

func (s *service) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, *Queries, error) {
	tx, err := s.db.BeginTx(ctx, opts)

	if err != nil {
		return nil, nil, err
	} else {
		return tx, New(s.db).WithTx(tx), nil
	}
}
