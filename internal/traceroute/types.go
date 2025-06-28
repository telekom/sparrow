// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"time"

	"github.com/telekom/sparrow/internal/helper"
)

// Result represents the result of a traceroute, mapping each target to its hops.
// Each target can have multiple hops, which are represented by the Hop struct.
type Result map[Target][]Hop

// Protocol represents the protocol used for the traceroute.
type Protocol string

// Protocol constants for the traceroute.
const (
	ProtocolTCP Protocol = "tcp"
)

func (p Protocol) String() string {
	switch p {
	case ProtocolTCP:
		return string(p)
	default:
		return "unknown"
	}
}

func (p Protocol) IsValid() bool {
	valid := []Protocol{ProtocolTCP}
	return slices.Contains(valid, p)
}

// Options contains the optional configuration for the traceroute.
type Options struct {
	// Retry is the retry configuration for the traceroute.
	Retry helper.RetryConfig `json:"retry" yaml:"retry" mapstructure:"retry"`
	// MaxTTL is the maximum TTL to use for the traceroute.
	MaxTTL int `json:"maxHops" yaml:"maxHops" mapstructure:"maxHops"`
	// Timeout is the timeout for each hop in the traceroute.
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

// Target represents a target for the traceroute.
type Target struct {
	// Protocol is the protocol to use for the traceroute.
	Protocol Protocol `json:"protocol" yaml:"protocol" mapstructure:"protocol"`
	// Address is the target address to trace to.
	Address string `json:"address" yaml:"address" mapstructure:"address"`
	// Port is the port to use for the traceroute.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// hopTTL is the TTL to start the traceroute with.
	hopTTL int
	// hopChan is the channel to send hops to.
	hopChan chan<- Hop
}

// withHopTTL returns a new Target with the specified hop TTL.
func (t Target) withHopTTL(ttl int) Target {
	return Target{
		Protocol: t.Protocol,
		Address:  t.Address,
		Port:     t.Port,
		hopChan:  t.hopChan,
		hopTTL:   ttl,
	}
}

func (t Target) String() string {
	if t.Port != 0 {
		return net.JoinHostPort(t.Address, strconv.Itoa(t.Port))
	}
	return t.Address
}

func (t Target) Validate() error {
	if t.Address == "" {
		return errors.New("target address cannot be empty")
	}
	if !t.Protocol.IsValid() {
		return fmt.Errorf("invalid target protocol: %s", t.Protocol)
	}
	if t.Port < 0 || t.Port > 65535 {
		return fmt.Errorf("invalid target port: %d, must be between 0 and 65535", t.Port)
	}
	return nil
}

func (t Target) ToAddr() (net.Addr, error) {
	switch t.Protocol {
	case ProtocolTCP:
		return net.ResolveTCPAddr("tcp", t.String())
	default:
		return nil, net.InvalidAddrError("invalid target protocol")
	}
}

type Hop struct {
	Latency time.Duration `json:"-" yaml:"-"`
	Addr    HopAddress    `json:"addr" yaml:"addr"`
	Name    string        `json:"name" yaml:"name"`
	TTL     int           `json:"ttl" yaml:"ttl"`
	Reached bool          `json:"reached" yaml:"reached"`
}

func (h Hop) MarshalJSON() ([]byte, error) {
	type alias Hop
	return json.Marshal(&struct {
		Latency string `json:"latency"`
		alias
	}{
		Latency: h.Latency.String(),
		alias:   alias(h),
	})
}

func (h Hop) String() string {
	reached := ""
	if h.Reached {
		reached = "  (reached)"
	}

	const maxNameLength = 45
	name := h.Name
	if name == "" || len(name) > maxNameLength {
		name = h.Addr.String()
	}

	return fmt.Sprintf("%-2d  %-45.45s  %s%s",
		h.TTL, name, h.Latency.String(), reached)
}

type HopAddress struct {
	IP   string `json:"ip" yaml:"ip"`
	Port int    `json:"port,omitempty" yaml:"port,omitempty"`
}

func newHopAddress(addr net.Addr) HopAddress {
	switch a := addr.(type) {
	case *net.UDPAddr:
		return HopAddress{IP: a.IP.String(), Port: a.Port}
	case *net.TCPAddr:
		return HopAddress{IP: a.IP.String(), Port: a.Port}
	case *net.IPAddr:
		return HopAddress{IP: a.IP.String()}
	default:
		return HopAddress{}
	}
}

func (a HopAddress) String() string {
	if a.Port != 0 {
		return fmt.Sprintf("%s:%d", a.IP, a.Port)
	}
	return a.IP
}
