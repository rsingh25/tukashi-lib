package web

import (
	"github.com/rsingh25/tukashi-lib/util"

	"log/slog"
)

var appLog *slog.Logger

func init() {
	appLog = util.Logger.With("package", "util/web")
}
