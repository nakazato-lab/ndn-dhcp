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

type routerResponse struct {
	RouterIP string `json:"routerIP"`
	Port     uint32 `json:"port"`
}

const addressFile = "/etc/ndn-config/ADDRESS"

func requestRouterAddress(nodeName string) (string, error) {
	serverURL := dhcpServerURL
	endpoint, err := url.Parse(serverURL)
	if err != nil {
		return "", err
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/v1/router"
	query := endpoint.Query()
	query.Set("nodeName", nodeName)
	endpoint.RawQuery = query.Encode()

	response, err := (&http.Client{Timeout: 5 * time.Second}).Get(endpoint.String())
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return "", fmt.Errorf("DHCP Server returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var result routerResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.RouterIP == "" || result.Port == 0 {
		return "", fmt.Errorf("DHCP Server returned an invalid Router endpoint")
	}
	return net.JoinHostPort(result.RouterIP, fmt.Sprint(result.Port)), nil
}

func registerRouterAddress(path, address string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(address+"\n"), 0644)
}

func main() {
	nodeName := flag.String("node-name", os.Getenv("NODE_NAME"), "requesting Kubernetes Node name")
	flag.Parse()

	if *nodeName == "" {
		fmt.Fprintln(os.Stderr, "node name is required: set --node-name or NODE_NAME")
		os.Exit(1)
	}

	address, err := requestRouterAddress(*nodeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "request DHCP Server: %v\n", err)
		os.Exit(1)
	}
	if err := registerRouterAddress(addressFile, address); err != nil {
		fmt.Fprintf(os.Stderr, "write transport: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(address)
}
