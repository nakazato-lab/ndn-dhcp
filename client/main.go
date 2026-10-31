package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const dhcpServerURL = "http://ndn-dhcp:8080"

type routerInfo struct {
	RouterIP   string `json:"routerIP"`
	Port       uint32 `json:"port"`
	RouterName string `json:"routerName"`
}

const (
	addressFile    = "/etc/ndn-config/ADDRESS"
	routerNameFile = "/etc/ndn-config/ROUTER_NAME"
)

func requestRouterInfo(nodeName string) (routerInfo, error) {
	serverURL := dhcpServerURL
	endpoint, err := url.Parse(serverURL)
	if err != nil {
		return routerInfo{}, err
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/v1/router"
	query := endpoint.Query()
	query.Set("nodeName", nodeName)
	endpoint.RawQuery = query.Encode()

	response, err := (&http.Client{Timeout: 5 * time.Second}).Get(endpoint.String())
	if err != nil {
		return routerInfo{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return routerInfo{}, fmt.Errorf("DHCP Server returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var result routerInfo
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return routerInfo{}, err
	}
	if result.RouterIP == "" || result.Port == 0 || result.RouterName == "" {
		return routerInfo{}, fmt.Errorf("DHCP Server returned an invalid Router endpoint")
	}
	return result, nil
}

func registerRouterInfo(router routerInfo) error {
	if err := os.MkdirAll(filepath.Dir(addressFile), 0755); err != nil {
		return err
	}
	address := net.JoinHostPort(router.RouterIP, fmt.Sprint(router.Port))
	if err := os.WriteFile(addressFile, []byte(address+"\n"), 0644); err != nil {
		return err
	}
	return os.WriteFile(routerNameFile, []byte(router.RouterName+"\n"), 0644)
}

func main() {
	nodeName := flag.String("node-name", os.Getenv("NODE_NAME"), "requesting Kubernetes Node name")
	flag.Parse()

	if *nodeName == "" {
		fmt.Fprintln(os.Stderr, "node name is required: set --node-name or NODE_NAME")
		os.Exit(1)
	}

	router, err := requestRouterInfo(*nodeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request DHCP Server: %v\n", err)
		os.Exit(1)
	}
	if err := registerRouterInfo(router); err != nil {
		fmt.Fprintf(os.Stderr, "write router info: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(net.JoinHostPort(router.RouterIP, fmt.Sprint(router.Port)))
}
