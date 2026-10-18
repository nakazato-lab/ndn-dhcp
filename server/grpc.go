package main

import (
	"context"
	"net"

	dhcppb "github.com/nakazato-lab/ndn-dhcp/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const nfdPort = 6363

type routerResolver struct {
	dhcppb.UnimplementedRouterResolverServer
	routerMap *nodeRouterMap
}

func (r *routerResolver) GetRouterAddress(_ context.Context, request *dhcppb.RouterRequest) (*dhcppb.RouterResponse, error) {
	if request.GetNodeName() == "" {
		return nil, status.Error(codes.InvalidArgument, "node_name is required")
	}
	router, found := r.routerMap.selectRouter(request.GetNodeName())
	if !found {
		return nil, status.Error(codes.Unavailable, "no NDN Router is available")
	}
	return &dhcppb.RouterResponse{
		RouterIp: router,
		Port:     nfdPort,
	}, nil
}

func newGRPCServer(address string, routerMap *nodeRouterMap) (*grpc.Server, net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}
	server := grpc.NewServer()
	dhcppb.RegisterRouterResolverServer(server, &routerResolver{routerMap: routerMap})
	return server, listener, nil
}
