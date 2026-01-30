package cli

import (
	"errors"
	"net"
)

func DetectPrimaryIPv4() (net.IP, error) {
	if ip := detectViaUDP(); ip != nil {
		return ip, nil
	}

	if ip := detectViaInterfaces(); ip != nil {
		return ip, nil
	}

	return nil, errors.New("no non-loopback IPv4 address found")
}

func detectViaUDP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil
	}

	ip := localAddr.IP.To4()
	if ip == nil || ip.IsLoopback() {
		return nil
	}

	return ip
}

func detectViaInterfaces() net.IP {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ip := extractIPv4(addr)
			if ip != nil && isPrivateIPv4(ip) {
				return ip
			}
		}
	}

	return nil
}

func extractIPv4(addr net.Addr) net.IP {
	switch value := addr.(type) {
	case *net.IPNet:
		return value.IP.To4()
	case *net.IPAddr:
		return value.IP.To4()
	default:
		return nil
	}
}

func isPrivateIPv4(ip net.IP) bool {
	if ip == nil {
		return false
	}

	switch {
	case ip[0] == 10:
		return true
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		return true
	case ip[0] == 192 && ip[1] == 168:
		return true
	default:
		return false
	}
}
