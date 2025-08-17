package config

type rabbitmq_config struct {
	Host                     string
	Port                     int
	User                     string
	Password                 string
	VirtualHost              string
	BlacklistExchange        string ///黑名单交换机
	BlacklistAccesslogQueue  string ///黑名单上报队列
	AccesslogToInfluxDBQueue string
}
