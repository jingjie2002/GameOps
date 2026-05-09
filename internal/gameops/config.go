package gameops

import (
	"os"
	"strings"
)

type Config struct {
	Addr             string
	AdminUser        string
	AdminPassword    string
	OperatorUser     string
	OperatorPassword string
	AuditorUser      string
	AuditorPassword  string
	TokenSigningKey  string
	CoreRankHTTP     string
	MySQLDSN         string
}

func ConfigFromEnv() Config {
	return Config{
		Addr:             getenv("GAMEOPS_ADDR", "127.0.0.1:18090"),
		AdminUser:        getenv("GAMEOPS_ADMIN_USER", "admin"),
		AdminPassword:    getenv("GAMEOPS_ADMIN_PASSWORD", "admin_demo"),
		OperatorUser:     getenv("GAMEOPS_OPERATOR_USER", "operator"),
		OperatorPassword: getenv("GAMEOPS_OPERATOR_PASSWORD", "operator_demo"),
		AuditorUser:      getenv("GAMEOPS_AUDITOR_USER", "auditor"),
		AuditorPassword:  getenv("GAMEOPS_AUDITOR_PASSWORD", "auditor_demo"),
		TokenSigningKey:  getenv("GAMEOPS_TOKEN_SIGNING_KEY", "gameops-dev-signing-key"),
		CoreRankHTTP:     trimRightSlash(getenv("CORE_RANK_HTTP", "http://127.0.0.1:8081")),
		MySQLDSN:         getenv("GAMEOPS_MYSQL_DSN", ""),
	}
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func trimRightSlash(value string) string {
	return strings.TrimRight(value, "/")
}
