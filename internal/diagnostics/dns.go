package diagnostics

import (
	"fmt"
	"net"
	"strings"
)

func (s *Service) DNSLookup(host string) (*DNSResult, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}

	result := &DNSResult{
		Host:        host,
		ARecords:    []string{},
		AAAARecords: []string{},
		NSRecords:   []string{},
		MXRecords:   []string{},
	}

	ips, err := net.LookupHost(host)
	if err == nil {
		for _, ip := range ips {
			parsed := net.ParseIP(ip)
			if parsed == nil {
				continue
			}
			if parsed.To4() != nil {
				result.ARecords = append(result.ARecords, ip)
			} else {
				result.AAAARecords = append(result.AAAARecords, ip)
			}
		}
	}

	cname, err := net.LookupCNAME(host)
	if err == nil && cname != "" {
		result.CNAME = strings.TrimSuffix(cname, ".")
	}

	nss, err := net.LookupNS(host)
	if err == nil {
		for _, ns := range nss {
			if ns.Host != "" {
				result.NSRecords = append(result.NSRecords, strings.TrimSuffix(ns.Host, "."))
			}
		}
	}

	mxs, err := net.LookupMX(host)
	if err == nil {
		for _, mx := range mxs {
			if mx.Host != "" {
				result.MXRecords = append(result.MXRecords, strings.TrimSuffix(mx.Host, "."))
			}
		}
	}

	return result, nil
}
