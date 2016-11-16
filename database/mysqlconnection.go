package database

import "database/sql"
import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/btcpool/config"
)

type mysqlConnectionInterface interface {
	open() error
	close() error
}

type MysqlConnection struct {
	dbInfo config.MysqlConnectConfig
	db *sql.DB
}

func NewMysqlConnection(dbInfo config.MysqlConnectConfig) *MysqlConnection {
	return &MysqlConnection{
		dbInfo: dbInfo,
	}
}

func (connection *MysqlConnection) open() error {
	return nil
}

func (connection *MysqlConnection) close() error {
	return nil
}