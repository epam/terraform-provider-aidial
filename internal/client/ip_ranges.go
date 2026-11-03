package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
)

// AllowedIPAddressRanges is a key's client IP allow-list.
//
// DIAL Core reads it from a JSON array of CIDR strings but serves it back as
// the IpAddressRanges object ({"ranges":[{"mask":..,"maskedBaseIp":..}]}), so
// marshaling and unmarshaling are asymmetric. A nil pointer means "any
// client"; an empty list means "no client".
type AllowedIPAddressRanges struct {
	// CIDRs holds the ranges, normalized to their masked base address.
	CIDRs []string
	// Undecodable reports that Core returned ranges in an encoding this client
	// could not turn back into CIDRs; callers should keep their prior value.
	Undecodable bool
}

func (r AllowedIPAddressRanges) MarshalJSON() ([]byte, error) {
	if r.CIDRs == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(r.CIDRs)
}

func (r *AllowedIPAddressRanges) UnmarshalJSON(data []byte) error {
	*r = AllowedIPAddressRanges{CIDRs: []string{}}

	var cidrs []string
	if err := json.Unmarshal(data, &cidrs); err == nil {
		for _, cidr := range cidrs {
			normalized, err := NormalizeCIDR(cidr)
			if err != nil {
				r.Undecodable = true
				return nil
			}
			r.CIDRs = append(r.CIDRs, normalized)
		}
		return nil
	}

	var object struct {
		Ranges []struct {
			Mask         json.RawMessage `json:"mask"`
			MaskedBaseIP json.RawMessage `json:"maskedBaseIp"`
		} `json:"ranges"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		r.Undecodable = true
		return nil
	}
	for _, item := range object.Ranges {
		mask, errMask := decodeIPBytes(item.Mask)
		base, errBase := decodeIPBytes(item.MaskedBaseIP)
		if errMask != nil || errBase != nil || len(mask) != len(base) || (len(mask) != 4 && len(mask) != 16) {
			r.Undecodable = true
			return nil
		}
		ones, bits := net.IPMask(mask).Size()
		if bits == 0 {
			r.Undecodable = true
			return nil
		}
		r.CIDRs = append(r.CIDRs, (&net.IPNet{IP: net.IP(base), Mask: net.CIDRMask(ones, bits)}).String())
	}
	return nil
}

// NormalizeCIDR returns cidr with its host bits cleared, the form DIAL Core
// stores, e.g. 198.51.100.14/24 becomes 198.51.100.0/24.
func NormalizeCIDR(cidr string) (string, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}
	return network.String(), nil
}

// decodeIPBytes decodes a Java byte[] as serialized by Jackson: either a
// base64 string or an array of (signed) byte values.
func decodeIPBytes(raw json.RawMessage) ([]byte, error) {
	raw = bytes.TrimSpace(raw)
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err == nil {
		return base64.StdEncoding.DecodeString(encoded)
	}
	var items []json.Number
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	result := make([]byte, len(items))
	for i, item := range items {
		value, err := strconv.ParseInt(item.String(), 10, 16)
		if err != nil || value < -128 || value > 255 {
			return nil, fmt.Errorf("invalid byte %q", item)
		}
		result[i] = byte(value)
	}
	return result, nil
}
