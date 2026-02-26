package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	proto2 "cluster_manager/api/proto"
	"cluster_manager/internal/control_plane/autoscaling"
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
)

const (
	defaultControlPlane = "127.0.0.1:9090"
)

func deploy(i int) error {
	cpApi, err := common.NewControlPlaneConnection([]string{defaultControlPlane})
	if err != nil {
		return fmt.Errorf("failed to create control plane connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	autoscalingConfig := autoscaling.NewDefaultAutoscalingMetadata(false)
	autoscalingConfig.ScalingUpperBound = 1

	resp, err := cpApi.RegisterService(ctx, &proto2.ServiceInfo{
		Name:  fmt.Sprintf("fire-%d", i),
		Image: "docker.io/cvetkovic/empty_function_firecracker:latest",
		PortForwarding: &proto2.PortMapping{
			GuestPort: 80,
			Protocol:  proto2.L4Protocol_TCP,
		},
		AutoscalingConfig: autoscalingConfig,
	})

	if err != nil {
		return fmt.Errorf("register service failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("register service returned success=false")
	}

	fmt.Printf("service deployed: fire-%d\n", i)
	return nil
}

func callFunction(i int) error {
	logrus.Info("Invoking function via HTTP/2 h2c on http://localhost:8080")

	client := http.Client{
		Timeout: 300 * time.Second,
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			DisableCompression: true,
		},
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:8080/fire-%d", i), nil)
	if err != nil {
		return err
	}

	req.Host = fmt.Sprintf("fire-%d", i)
	req.Header.Set("workload", "empty")
	req.Header.Set("requested_cpu", strconv.Itoa(1))
	req.Header.Set("requested_memory", strconv.Itoa(1))
	req.Header.Set("multiplier", strconv.Itoa(0))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call function GET /: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("function GET returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read function body: %w", err)
	}

	fmt.Println("Function response:")
	fmt.Println(string(body))

	return nil
}

func usage() {
	fmt.Println("Usage: check <command>")
	fmt.Println("Commands:")
	fmt.Println("  deploy    Deploy the preset test service to control plane")
	fmt.Println("  call      Invoke the preset function (calls localhost:8080 then posts to 8082)")
	fmt.Println("  list      List services registered in control plane")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "deploy":
		if err := deploy(0); err != nil {
			logrus.Fatalf("deploy failed: %v", err)
		}
	case "call":
		if err := callFunction(0); err != nil {
			logrus.Fatalf("call failed: %v", err)
		}
	case "cold":
		// Deploy 100 times
		for i := 0; i < 100; i++ {
			if err := deploy(i); err != nil {
				logrus.Fatalf("deploy %d failed: %v", i, err)
			}
		}
		// Call 100 times
		for i := 0; i < 100; i++ {
			if err := callFunction(i); err != nil {
				logrus.Fatalf("call %d failed: %v", i, err)
			}
		}
	case "warm":
		for i := 0; i < 100; i++ {
			if err := callFunction(i); err != nil {
				logrus.Fatalf("call %d failed: %v", i, err)
			}
		}

	default:
		usage()
		os.Exit(2)
	}
}
