package main

import (
//        "bytes"
	"net"
        "context"
	"time"
        "os"
        "os/exec"


        "github.com/golang/protobuf/ptypes/empty"
        "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
        "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
        "github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
        "github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
        "github.com/sirupsen/logrus"
)

// LoadBalancerEndpoint a comment
type LoadBalancerEndpoint struct {
	SelfIp	string
}

func (lbe *LoadBalancerEndpoint) Request(
        ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	newConnection := request.GetConnection()

//	logrus.Infof("LB Request: newConnection.Labels :v", newConnection.Labels)

        serverIp, _, err := net.ParseCIDR(newConnection.Context.IpContext.SrcIpAddr)

	if err != nil {
		logrus.Errorf("LB: request, Error from parseCidr %v", err)
                return nil, err
	}
	
	cmd := exec.Command("vppctl", "lb", "as", os.Getenv("VIP_NETWORK"), serverIp.String())
	err = cmd.Run()
	if err != nil {
		logrus.Errorf("Error from lb as add %v", err)
		return nil, err
	}

	//if newConnection.Context.IpContext.Ipflags == nil {
	//	newConnection.Context.IpContext.Ipflags = make(map[string]string)
	//}

	//newConnection.Context.IpContext.Ipflags["GRE_VIP"]=os.Getenv("VIP_NETWORK")

	newConnection.Labels["GRE_VIP"]=os.Getenv("VIP_NETWORK");

	err = newConnection.IsComplete()
	if err != nil {
		logrus.Errorf("LB: New connection is not complete: %v", err)
		return nil, err
	}

	logrus.Infof("lb completed on connection: %v", newConnection)

	if endpoint.Next(ctx) != nil {
		return endpoint.Next(ctx).Request(ctx, request)
	}

	return newConnection, nil
}

// Close implements the close handler
func (lbe *LoadBalancerEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {

	serverIp, _, err := net.ParseCIDR(connection.Context.IpContext.SrcIpAddr)
        if err != nil {
		return &empty.Empty{}, nil
        }

	cmd := exec.Command("vppctl", "lb", "as", os.Getenv("VIP_NETWORK"), serverIp.String(), "del")
        err = cmd.Run()
        if err != nil {
                logrus.Errorf("Error from lb as remove %v", err)
                return nil, err
        }

        if endpoint.Next(ctx) != nil {
                if _, err := endpoint.Next(ctx).Close(ctx, connection); err != nil {
                        return &empty.Empty{}, nil
                }
        }
	return &empty.Empty{}, nil
}



// Name returns the composite name
func (lbe *LoadBalancerEndpoint) Name() string {
        return "LB"
}

// newLoadBalancerEndpoint creates a LoadBalancerEndpoint
func newLoadBalancerEndpoint(configuration *common.NSConfiguration, ipam *IpamEndpoint) *LoadBalancerEndpoint {

	SelfIp := ipam.getSelfIp()
	logrus.Infof("lbe: got ip %v", SelfIp)
        self := &LoadBalancerEndpoint{
		SelfIp:	SelfIp,
        }

        ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
        defer cancel()

        logrus.Infof("lb Wait for portavail")
        if err := tools.WaitForPortAvailable(ctx, "tcp", defaultVPPAgentEndpoint, 100*time.Millisecond); err != nil {
                return nil
        }
        logrus.Infof("lb Wait for portavail: got it")


	cmd := exec.Command("vppctl", "lb",  "conf", "ip4-src-address", SelfIp)
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("Error from lb conf %v", err)
	}

	cmd = exec.Command("vppctl", "lb", "vip", os.Getenv("VIP_NETWORK"))
	err = cmd.Run()
	if err != nil {
		logrus.Errorf("Error from lb vip %v", err)
	}
	
	cmd = exec.Command("vppctl", "create", "host-interface", "name", "tap0")
	err = cmd.Run()
	if err != nil {
		logrus.Errorf("Error from adding vpn itf %v", err)
		return nil
	}
	
	cmd = exec.Command("vppctl", "set", "interface", "ip", "address", "host-tap0", "10.8.0.1/24" )
	err = cmd.Run()
	if err != nil {
		logrus.Errorf("Error from set address vpn itf %v", err)
		return nil
	}
	
	cmd = exec.Command("vppctl", "set", "interface", "state", "host-tap0", "up" )
	err = cmd.Run()
	if err != nil {
		logrus.Errorf("Error from set state vpn itf %v", err)
		return nil
	}

	cmd = exec.Command("ifconfig", "tap0", "0.0.0.0")
	err = cmd.Run()
	if err != nil {
		logrus.Errorf("Error from reset address host itf %v", err)
		return nil
	}

        return self
}
