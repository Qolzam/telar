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

	// PostgreSQL specific
	PostgresConfig *PostgreSQLConfig
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
