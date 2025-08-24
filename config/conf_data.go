package config

type confData struct {
	TcpListenerAddress []string /// [":2423",":5467"]
	ProxyType          string
	LogDir             string
}
