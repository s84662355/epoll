package server

import (
	"context"
	"net"
)

func DialContext(
	ctx context.Context,
	network, address string,
) (net.Conn, error) {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, network, address)
	return conn, err
}
