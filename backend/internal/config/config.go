package config

type Config struct {
	ServerPort string
	Postgres  PostgresConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Load() Config {
	return Config{
		ServerPort: "8080",
		Postgres: PostgresConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "blog",
			Password: "blog123456",
			DBName:   "blog_db",
			SSLMode:  "disable",
		},
	}
}