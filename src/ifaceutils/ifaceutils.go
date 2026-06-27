package ifaceutils

import (
	"errors"
	"fmt"
	"log"
	"net"
	"slices"
	"strings"

	"github.com/blokadainfo/wireguard-multipath/src/glob"
)

var (
	_disallowedCIDRs = [...]net.IPNet{
		ExtractCIDR(MustParseCIDR("127.0.0.0/8")),
		ExtractCIDR(MustParseCIDR("169.254.0.0/16")),
	}

	ErrNoAddressFound    = errors.New("no address was found")
	ErrAddressNotAllowed = errors.New("address is not allowed")
)

func KnownError(err error) bool {
	if errors.Is(err, ErrNoAddressFound) {
		return true
	}

	if errors.Is(err, ErrAddressNotAllowed) {
		return true
	}

	return false
}

func ListInterfaces() {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Failed to get interfaces for listing")
	}

	for _, iface := range interfaces {
		ifname := iface.Name
		print("\r\n" + ifname + "\r\n")

		ifaddr, _ := GetAddressByInterface(iface)

		print("  Address: " + ifaddr + "\r\n")
	}
}

func GetAddressByInterface(iface net.Interface) (string, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", ErrNoAddressFound
	}

	for _, addr := range addrs {
		splAddr, _, _ := strings.Cut(addr.String(), "/")
		if isAddressAllowed(splAddr) {
			return splAddr, nil
		}
	}

	return "", ErrAddressNotAllowed
}

func isAddressAllowed(addr string) bool {
	if strings.ContainsRune(addr, ':') { // TODO: IPv6 support
		return false
	}

	ip := net.ParseIP(addr) // TODO: Handle if ip is nil (failed to parse)?
	for _, cidr := range _disallowedCIDRs {
		if cidr.Contains(ip) {
			return false
		}
	}

	return true
}

func IsInterfaceExcluded(excludedInterfaces []glob.Glob, ifname string) bool {
	return slices.ContainsFunc(excludedInterfaces, func(e glob.Glob) bool {
		return e.MatchString(ifname)
	})
}

func IsCIDRExcluded(excludedCIDRs []net.IPNet, ifaddr string) bool {
	ip := net.ParseIP(ifaddr)
	if ip == nil {
		panic(fmt.Sprintf("failed to parse ip '%v'", ifaddr))
	}

	return slices.ContainsFunc(excludedCIDRs, func(e net.IPNet) bool {
		return e.Contains(ip)
	})
}
