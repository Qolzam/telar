// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package interfaces

// RepositoryConfig represents the configuration for repository creation
type RepositoryConfig struct {
	DatabaseType     string
	ConnectionString string
	DatabaseName     string

	// MongoDB specific
	MongoConfig *MongoDBConfig

	// PostgreSQL specific
	PostgresConfig *PostgreSQLConfig
}

// MongoDBConfig represents MongoDB specific configuration
type MongoDBConfig struct {
	Host                   string
	Port                   int
	Username               string
	Password               string
	AuthDatabase           string
	ReplicaSet             string
	SSL                    bool
	ConnectTimeout         int
	SocketTimeout          int
	MaxPoolSize            int
	MinPoolSize            int
	MaxIdleTime            int
	ServerSelectionTimeout int
}

// PostgreSQLConfig represents PostgreSQL specific configuration
type PostgreSQLConfig struct {
	Host               string
	Port               int
	Username           string
	Password           string
	Database           string
	SSLMode            string
	SSL                bool
	ConnectTimeout     int
	MaxOpenConnections int
	MaxIdleConnections int
	MaxLifetime        int
	Schema             string
}
