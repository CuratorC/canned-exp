package bootstrap

import (
	"canned-exp/internal/enum"

	"github.com/CuratorC/gocanned/cerr"
	"github.com/CuratorC/gocanned/config"
	"github.com/CuratorC/gocanned/database"
)

// SetupDatabase initializes a database connection by name and returns it.
func SetupDatabase(name enum.DatabaseName) (db *database.DB, err error) {
	nameStr := string(name)
	cfg := database.Config{
		Driver:          config.GetString("database." + nameStr + ".driver"),
		Host:            config.GetString("database." + nameStr + ".host"),
		Port:            config.GetInt("database." + nameStr + ".port"),
		User:            config.GetString("database." + nameStr + ".user"),
		Password:        config.GetString("database." + nameStr + ".password"),
		DBName:          config.GetString("database." + nameStr + ".name"),
		DSN:             config.GetString("database." + nameStr + ".dsn"),
		Charset:         config.GetString("database." + nameStr + ".charset"),
		MaxOpenConns:    config.GetInt("database." + nameStr + ".max_open_conns"),
		MaxIdleConns:    config.GetInt("database." + nameStr + ".max_idle_conns"),
		ConnMaxLifetime: config.GetInt("database." + nameStr + ".conn_max_lifetime"),
	}

	db, err = database.Connect(cfg)
	if err != nil {
		return nil, cerr.Wrapf(err, "database %s connection failed", nameStr)
	}

	return db, nil
}
