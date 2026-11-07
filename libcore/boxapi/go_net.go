package boxapi

import (
	"context"
	"net"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

func DialContext(ctx context.Context, box *box.Box, tracker adapter.ConnectionTracker, network, addr string) (net.Conn, error) {
	defOutboundTag := box.Outbound().Default().Tag()
	conn, err := dialer.NewDetour(box.Outbound(), defOutboundTag, true).DialContext(ctx, network, metadata.ParseSocksaddr(addr))
	if err != nil {
		return nil, err
	}
	if tracker != nil {
		flow := tracker.RoutedFlow(ctx, adapter.InboundContext{}, nil, box.Outbound().Default())
		if flow != nil {
			// This is an outgoing socket: writes upload and reads download.
			conn = bufio.NewCounterConn(conn,
				[]N.CountFunc{func(n int64) { flow.CountReverse(int(n)) }},
				[]N.CountFunc{func(n int64) { flow.CountForward(int(n)) }},
			)
		}
	}
	return conn, nil
}
