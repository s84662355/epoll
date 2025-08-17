package config

type redis_config struct {
	Addr         string
	Password     string
	DB           int
	MinIdleConns int // 最小空闲连接数
	MaxIdleConns int ///最大空闲连接数
	PoolSize     int // 连接池大小
}
