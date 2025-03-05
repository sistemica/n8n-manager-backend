package main

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "github.com/sistemica/n8n-manager-backend/migrations"
	"github.com/sistemica/n8n-manager-backend/n8n"
)

func initLogger() *zap.Logger {
	// Get log level from environment variable (default to "info")
	logLevel := os.Getenv("LOG_LEVEL")
	var level zapcore.Level

	switch strings.ToLower(logLevel) {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn", "warning":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel // Default level
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	// Log the current level for confirmation
	logger.Info("Logger initialized", zap.String("level", level.String()))

	return logger
}

func main() {
	logger := initLogger()
	defer logger.Sync()

	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	app := pocketbase.New()

	isGoRun := strings.HasPrefix(os.Args[0], os.TempDir())
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: isGoRun,
	})

	n8n.InitCronJobs(app, logger)

	app.RootCmd.PersistentFlags().String("http", "0.0.0.0:"+port, "the HTTP server address")

	logger.Info("Starting PocketBase server",
		zap.String("port", port),
	)

	if err := app.Start(); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
