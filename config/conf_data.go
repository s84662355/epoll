package config

type confData struct {
	TcpListenerAddress  []string /// [":2423",":5467"]
	GrpcListenerAddress string
	LogDir              string
	LocalIp             string
	ProcessName         string
	Redis               *redis_config
	Rabbitmq            *rabbitmq_config
	Nacos               *nacos_config
}
