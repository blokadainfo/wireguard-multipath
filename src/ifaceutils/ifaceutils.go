package ifaceutils

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
)

var ErrAddressNotAllowed = errors.New("address is not allowed")

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
		return "", fmt.Errorf("failed to get address of the interface: %v", iface.Name)
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

	ip := net.ParseIP(addr)
	disallowedNetworks := [...]string{
		"169.254.0.0/16",
		"127.0.0.0/8",
	}

	for _, disallowedNetwork := range disallowedNetworks {
		_, subnet, _ := net.ParseCIDR(disallowedNetwork)
		if subnet.Contains(ip) {
			return false
		}
	}

	return true
}
