package database

import "database/sql"
import _ "github.com/go-sql-driver/mysql"

type MysqlConnectInfo struct {
	Host     string
	Port     string
	Username string
	Password string
	Dbname   string
}

type mysqlConnectionInterface interface {
	open() error
	close() error
}

type MysqlConnection struct {
	dbInfo *MysqlConnectInfo
	db *sql.DB
}

func NewMysqlConnection(dbInfo *MysqlConnectInfo) *MysqlConnection {
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