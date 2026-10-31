package main

import (
	"encoding/json"
	"log"
	"net/http"

	dhcppb "github.com/nakazato-lab/ndn-dhcp/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type gateway struct {
	resolver dhcppb.RouterResolverClient
}

type routerResponse struct {
	RouterIP   string `json:"routerIP"`
	Port       uint32 `json:"port"`
	RouterName string `json:"routerName"`
}

func (g *gateway) handleRouterAddress(w http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method must be GET", http.StatusMethodNotAllowed)
		return
	}
	nodeName := request.URL.Query().Get("nodeName")
	if nodeName == "" {
		http.Error(w, "nodeName is required", http.StatusBadRequest)
		return
	}
	response, err := g.resolver.GetRouterAddress(request.Context(), &dhcppb.RouterRequest{NodeName: nodeName})
	if err != nil {
		code := http.StatusServiceUnavailable
		if status.Code(err) == codes.InvalidArgument {
			code = http.StatusBadRequest
		}
		http.Error(w, err.Error(), code)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routerResponse{
		RouterIP:   response.GetRouterIp(),
		Port:       response.GetPort(),
		RouterName: response.GetRouterName(),
	})
}

func main() {
	connection, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect to DHCP Server: %v", err)
	}
	defer connection.Close()

	handler := &gateway{resolver: dhcppb.NewRouterResolverClient(connection)}
	log.Fatal(http.ListenAndServe(":8080", http.HandlerFunc(handler.handleRouterAddress)))
}
