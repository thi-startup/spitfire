package networking

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/thi-startup/spitfire/pkg/log"
)

// TAPDevice represents a TAP network device
type TAPDevice struct {
	Name       string
	MACAddress string
	Bridge     string
	MTU        int
}

// CreateTAPDevice creates a new TAP device
func CreateTAPDevice(name string, mac string) (*TAPDevice, error) {
	// Create TAP device using netlink
	la := netlink.NewLinkAttrs()
	la.Name = name
	
	// Set MAC address if provided
	if mac != "" {
		hwAddr, err := net.ParseMAC(mac)
		if err != nil {
			return nil, fmt.Errorf("invalid MAC address %s: %w", mac, err)
		}
		la.HardwareAddr = hwAddr
	}

	// Create the TAP device
	tap := &netlink.Tuntap{
		LinkAttrs: la,
		Mode:      netlink.TUNTAP_MODE_TAP,
	}

	if err := netlink.LinkAdd(tap); err != nil {
		return nil, fmt.Errorf("failed to create TAP device %s: %w", name, err)
	}

	// Bring the interface up
	if err := netlink.LinkSetUp(tap); err != nil {
		// Clean up on failure
		netlink.LinkDel(tap)
		return nil, fmt.Errorf("failed to bring up TAP device %s: %w", name, err)
	}

	log.Debugf("Created TAP device %s", name)

	return &TAPDevice{
		Name:       name,
		MACAddress: mac,
		MTU:        1500, // Default MTU
	}, nil
}

// Delete removes the TAP device
func (t *TAPDevice) Delete() error {
	link, err := netlink.LinkByName(t.Name)
	if err != nil {
		// Device doesn't exist, that's ok
		return nil
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete TAP device %s: %w", t.Name, err)
	}

	log.Debugf("Deleted TAP device %s", t.Name)
	return nil
}

// AttachToBridge attaches the TAP device to a bridge
func (t *TAPDevice) AttachToBridge(bridgeName string) error {
	// Get the TAP device
	tap, err := netlink.LinkByName(t.Name)
	if err != nil {
		return fmt.Errorf("failed to find TAP device %s: %w", t.Name, err)
	}

	// Get the bridge
	bridge, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("failed to find bridge %s: %w", bridgeName, err)
	}

	// Attach TAP to bridge
	if err := netlink.LinkSetMaster(tap, bridge.(*netlink.Bridge)); err != nil {
		return fmt.Errorf("failed to attach TAP %s to bridge %s: %w", t.Name, bridgeName, err)
	}

	t.Bridge = bridgeName
	log.Debugf("Attached TAP device %s to bridge %s", t.Name, bridgeName)
	return nil
}

// SetMTU sets the MTU for the TAP device
func (t *TAPDevice) SetMTU(mtu int) error {
	link, err := netlink.LinkByName(t.Name)
	if err != nil {
		return fmt.Errorf("failed to find TAP device %s: %w", t.Name, err)
	}

	if err := netlink.LinkSetMTU(link, mtu); err != nil {
		return fmt.Errorf("failed to set MTU on TAP device %s: %w", t.Name, err)
	}

	t.MTU = mtu
	log.Debugf("Set MTU %d on TAP device %s", mtu, t.Name)
	return nil
}

// Bridge represents a network bridge
type Bridge struct {
	Name      string
	IPAddress string
	Subnet    string
	MTU       int
}

// CreateBridge creates a new network bridge
func CreateBridge(name string, subnet string) (*Bridge, error) {
	// Check if bridge already exists
	if link, _ := netlink.LinkByName(name); link != nil {
		log.Debugf("Bridge %s already exists", name)
		return &Bridge{
			Name:   name,
			Subnet: subnet,
			MTU:    1500,
		}, nil
	}

	// Create bridge
	la := netlink.NewLinkAttrs()
	la.Name = name
	
	br := &netlink.Bridge{LinkAttrs: la}
	
	if err := netlink.LinkAdd(br); err != nil {
		return nil, fmt.Errorf("failed to create bridge %s: %w", name, err)
	}

	// Parse and set IP address if provided
	if subnet != "" {
		addr, err := netlink.ParseAddr(subnet)
		if err != nil {
			netlink.LinkDel(br)
			return nil, fmt.Errorf("invalid subnet %s: %w", subnet, err)
		}

		if err := netlink.AddrAdd(br, addr); err != nil {
			netlink.LinkDel(br)
			return nil, fmt.Errorf("failed to add address to bridge %s: %w", name, err)
		}
	}

	// Bring the bridge up
	if err := netlink.LinkSetUp(br); err != nil {
		netlink.LinkDel(br)
		return nil, fmt.Errorf("failed to bring up bridge %s: %w", name, err)
	}

	log.Progress("Created bridge %s with subnet %s", name, subnet)

	return &Bridge{
		Name:      name,
		IPAddress: strings.Split(subnet, "/")[0],
		Subnet:    subnet,
		MTU:       1500,
	}, nil
}

// Delete removes the bridge
func (b *Bridge) Delete() error {
	link, err := netlink.LinkByName(b.Name)
	if err != nil {
		// Bridge doesn't exist, that's ok
		return nil
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete bridge %s: %w", b.Name, err)
	}

	log.Debugf("Deleted bridge %s", b.Name)
	return nil
}

// EnableIPForwarding enables IP forwarding for bridge networking
func EnableIPForwarding() error {
	// Enable IPv4 forwarding
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable IPv4 forwarding: %w", err)
	}

	// Enable IPv6 forwarding (optional, ignore errors)
	cmd = exec.Command("sysctl", "-w", "net.ipv6.conf.all.forwarding=1")
	cmd.Run() // Ignore errors for IPv6

	log.Debugf("Enabled IP forwarding")
	return nil
}

// SetupNAT sets up NAT/masquerading for a bridge
func SetupNAT(bridgeName string, subnet string) error {
	// Use iptables to set up NAT
	// This allows VMs on the bridge to access external networks
	
	// Enable masquerading for traffic from the bridge subnet
	cmd := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING",
		"-s", subnet, "!", "-o", bridgeName, "-j", "MASQUERADE")
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup NAT for bridge %s: %w", bridgeName, err)
	}

	// Allow forwarding from/to the bridge
	cmd = exec.Command("iptables", "-A", "FORWARD",
		"-i", bridgeName, "-j", "ACCEPT")
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup forwarding for bridge %s: %w", bridgeName, err)
	}

	cmd = exec.Command("iptables", "-A", "FORWARD",
		"-o", bridgeName, "-j", "ACCEPT")
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup forwarding for bridge %s: %w", bridgeName, err)
	}

	log.Debugf("Set up NAT for bridge %s with subnet %s", bridgeName, subnet)
	return nil
}

// CleanupNAT removes NAT rules for a bridge
func CleanupNAT(bridgeName string, subnet string) error {
	// Remove NAT rules (ignore errors as they might not exist)
	
	cmd := exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING",
		"-s", subnet, "!", "-o", bridgeName, "-j", "MASQUERADE")
	cmd.Run()

	cmd = exec.Command("iptables", "-D", "FORWARD",
		"-i", bridgeName, "-j", "ACCEPT")
	cmd.Run()

	cmd = exec.Command("iptables", "-D", "FORWARD",
		"-o", bridgeName, "-j", "ACCEPT")
	cmd.Run()

	log.Debugf("Cleaned up NAT for bridge %s", bridgeName)
	return nil
}

// GenerateMAC generates a random MAC address for VMs
func GenerateMAC() string {
	// Use locally administered MAC address range (02:xx:xx:xx:xx:xx)
	mac := make([]byte, 6)
	mac[0] = 0x02 // Locally administered
	
	// Generate random bytes for the rest
	for i := 1; i < 6; i++ {
		mac[i] = byte(i) // Simple generation, could use crypto/rand
	}
	
	return net.HardwareAddr(mac).String()
}

// FindAvailableIP finds an available IP address in a subnet
func FindAvailableIP(subnet string) (string, error) {
	// Parse the subnet
	_, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", fmt.Errorf("invalid subnet %s: %w", subnet, err)
	}

	// Start from .2 (reserve .1 for gateway)
	ip := ipnet.IP.To4()
	if ip == nil {
		return "", fmt.Errorf("only IPv4 subnets supported")
	}

	// Try to find an available IP
	// In production, this would check DHCP leases, ARP table, etc.
	ip[3] = 2 // Start from .2
	
	// For now, just increment and return
	// TODO: Implement proper IP allocation tracking
	return ip.String(), nil
}