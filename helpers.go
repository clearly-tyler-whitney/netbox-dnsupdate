// helpers.go

package main

import (
	"fmt"
	"net"
	"strings"
)

// extractHost extracts the host from the BindServerAddress.
func extractHost(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		// Assume no port specified, return the whole address
		return address
	}
	return host
}

// extractPort extracts the port from the BindServerAddress or returns default "53".
func extractPort(address string) string {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		// Assume no port specified, return default "53"
		return "53"
	}
	return port
}

// reverseDNSName computes the reverse DNS name for an IP address.
func reverseDNSName(ip string) string {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return ""
	}

	if parsedIP.To4() != nil {
		// IPv4
		octets := strings.Split(parsedIP.String(), ".")
		reverseOctets := []string{octets[3], octets[2], octets[1], octets[0]}
		return strings.Join(reverseOctets, ".") + ".in-addr.arpa."
	} else {
		// IPv6
		expandedIP := parsedIP.To16()
		if expandedIP == nil {
			return ""
		}
		var nibbles []string
		for i := len(expandedIP) - 1; i >= 0; i-- {
			nibbles = append(nibbles, fmt.Sprintf("%x", expandedIP[i]&0x0F))
			nibbles = append(nibbles, fmt.Sprintf("%x", (expandedIP[i]>>4)&0x0F))
		}
		return strings.Join(nibbles, ".") + ".ip6.arpa."
	}
}

// isValidIP checks if the provided string is a valid IP address.
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// snapshotToRecordData converts a *Snapshot to a *RecordData.
func snapshotToRecordData(s *Snapshot) *RecordData {
	if s == nil {
		return nil
	}
	return &RecordData{
		ID:         s.ID,
		FQDN:       s.FQDN,
		Type:       s.Type,
		Value:      s.Value,
		TTL:        s.TTL,
		DisablePTR: s.DisablePTR,
		Zone:       ZoneData{ID: s.Zone},
	}
}

// getFQDN retrieves the FQDN from preData or postData.
func getFQDN(preData, postData *RecordData) string {
	if postData != nil {
		return postData.FQDN
	}
	if preData != nil {
		return preData.FQDN
	}
	return ""
}
