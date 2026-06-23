package ifaceutils

import (
	"fmt"
	"net"
)

func MustParseCIDR(s string) (net.IP, *net.IPNet) {
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse CIDR (%v): %v", s, err))
	}

	return ip, ipnet
}

func ExtractIP(ip net.IP, cidr *net.IPNet) net.IP {
	return ip
}

func ExtractCIDR(ip net.IP, cidr *net.IPNet) net.IPNet {
	return *cidr
}
