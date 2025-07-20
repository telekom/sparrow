// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package traceroute

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/ipv4"
	"golang.org/x/sys/unix"
)

var (
	_ net.Conn        = (*fakeConn)(nil)
	_ syscall.RawConn = (*fakeRawConn)(nil)
)

// fakeConn implements [net.Conn] with no-op methods.
type fakeConn struct {
	setReadDeadlineFunc func(t time.Time) error
}

func (f *fakeConn) Read(b []byte) (int, error)    { return 0, nil }
func (f *fakeConn) Write(b []byte) (int, error)   { return len(b), nil }
func (f *fakeConn) Close() error                  { return nil }
func (f *fakeConn) LocalAddr() net.Addr           { return &net.UDPAddr{} }
func (f *fakeConn) RemoteAddr() net.Addr          { return &net.UDPAddr{} }
func (f *fakeConn) SetDeadline(t time.Time) error { return nil }
func (f *fakeConn) SetReadDeadline(t time.Time) error {
	if f.setReadDeadlineFunc != nil {
		return f.setReadDeadlineFunc(t)
	}
	return nil
}
func (f *fakeConn) SetWriteDeadline(t time.Time) error { return nil }

// fakeRawConn implements [syscall.RawConn] for testing.
type fakeRawConn struct {
	readFunc func(func(fd uintptr) bool) error
}

func (f *fakeRawConn) Read(fn func(fd uintptr) bool) error  { return f.readFunc(fn) }
func (f *fakeRawConn) Control(fn func(fd uintptr)) error    { return nil }
func (f *fakeRawConn) Write(fn func(fd uintptr) bool) error { return nil }

func TestErrQueueListener_Read(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) *errQueueListener
		wantErr     bool
		wantTimeout bool
		wantPacket  icmpPacket
	}{
		{
			name: "successful time exceeded",
			setup: func(_ *testing.T) *errQueueListener {
				l := &errQueueListener{
					conn: &fakeConn{},
					rawConn: &fakeRawConn{
						readFunc: func(fn func(fd uintptr) bool) error { fn(0); return nil },
					},
					recvPort: 1234,
					oobBuf:   make([]byte, oobBufSize),
				}
				recvMsg = func(fd uintptr, oob []byte, flags int) (*socketMsg, error) {
					return &socketMsg{
						from: &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 1234},
						port: 1234,
						oob:  []byte{},
					}, nil
				}
				parseExtendedErr = func(ctx context.Context, msg *socketMsg) (*icmpPacket, error) {
					return &icmpPacket{remoteAddr: msg.from, port: msg.port, reached: false}, nil
				}
				return l
			},
			wantPacket:  icmpPacket{remoteAddr: &net.UDPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 1234}, port: 1234, reached: false},
			wantErr:     false,
			wantTimeout: false,
		},
		{
			name: "deadline exceeded on empty queue",
			setup: func(_ *testing.T) *errQueueListener {
				l := &errQueueListener{
					conn: &fakeConn{},
					rawConn: &fakeRawConn{
						readFunc: func(fn func(fd uintptr) bool) error { fn(0); return nil },
					},
					recvPort: 4321,
					oobBuf:   make([]byte, oobBufSize),
				}
				recvMsg = func(fd uintptr, oob []byte, flags int) (*socketMsg, error) {
					return nil, unix.EAGAIN
				}
				return l
			},
			wantPacket:  icmpPacket{},
			wantErr:     true,
			wantTimeout: true,
		},
		{
			name: "error while receiving socket message",
			setup: func(_ *testing.T) *errQueueListener {
				l := &errQueueListener{
					conn: &fakeConn{},
					rawConn: &fakeRawConn{
						readFunc: func(fn func(fd uintptr) bool) error { fn(0); return nil },
					},
					recvPort: 4321,
					oobBuf:   make([]byte, oobBufSize),
				}
				recvMsg = func(fd uintptr, oob []byte, flags int) (*socketMsg, error) {
					return nil, errors.New("failed to receive message")
				}
				return l
			},
			wantPacket:  icmpPacket{},
			wantErr:     true,
			wantTimeout: true,
		},
		{
			name: "error setting read deadline",
			setup: func(_ *testing.T) *errQueueListener {
				l := &errQueueListener{
					conn: &fakeConn{
						setReadDeadlineFunc: func(_ time.Time) error {
							return errors.New("failed to set read deadline")
						},
					},
					rawConn: &fakeRawConn{
						readFunc: func(fn func(fd uintptr) bool) error { return nil },
					},
					recvPort: 1234,
					oobBuf:   make([]byte, oobBufSize),
				}
				return l
			},
			wantPacket:  icmpPacket{},
			wantErr:     true,
			wantTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origRecv := recvMsg
			origParse := parseExtendedErr
			defer func() { recvMsg = origRecv; parseExtendedErr = origParse }()

			l := tt.setup(t)

			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()

			pkt, err := l.Read(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.wantPacket, pkt)
			if tt.wantTimeout {
				assert.ErrorIs(t, err, context.DeadlineExceeded, "expected timeout error")
			} else {
				assert.NoError(t, err, "expected no error")
			}
		})
	}
}

func Test_parseExtendedErr(t *testing.T) {
	testAddr := &net.UDPAddr{IP: net.IPv4(192, 168, 1, 1), Port: 33434}

	tests := []struct {
		name     string
		icmpType uint8
		icmpCode uint8
		reached  bool
		wantErr  bool
	}{
		{
			name:     "time exceeded",
			icmpType: uint8(ipv4.ICMPTypeTimeExceeded),
			icmpCode: 0,
			reached:  false,
		},
		{
			name:     "destination unreachable - port unreachable",
			icmpType: uint8(ipv4.ICMPTypeDestinationUnreachable),
			icmpCode: icmpUnreachablePort,
			reached:  true,
		},
		{
			name:     "destination unreachable - host unreachable",
			icmpType: uint8(ipv4.ICMPTypeDestinationUnreachable),
			icmpCode: icmpUnreachableHost,
			reached:  false,
		},
		{
			name:     "unexpected ICMP type",
			icmpType: 99,
			icmpCode: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &socketMsg{
				from: testAddr,
				port: 33434,
				oob:  newExtendedErrOOB(tt.icmpType, tt.icmpCode),
			}

			got, err := parseExtendedErr(context.Background(), msg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, &icmpPacket{
					remoteAddr: testAddr,
					port:       33434,
					reached:    tt.reached,
				}, got)
			}
		})
	}
}

func Test_parseExtendedErr_Errors(t *testing.T) {
	testAddr := &net.UDPAddr{IP: net.IPv4(192, 168, 1, 1), Port: 33434}

	t.Run("short extended error data", func(t *testing.T) {
		msg := &socketMsg{
			from: testAddr,
			port: 33434,
			oob:  newControlMessage(unix.SOL_IP, unix.IP_RECVERR, []byte{0x01, 0x02, 0x03}),
		}

		_, err := parseExtendedErr(context.Background(), msg)
		assert.Error(t, err)
	})

	t.Run("no IP_RECVERR message", func(t *testing.T) {
		msg := &socketMsg{
			from: testAddr,
			port: 33434,
			oob:  newControlMessage(unix.SOL_SOCKET, unix.SO_TIMESTAMP, make([]byte, 16)),
		}

		_, err := parseExtendedErr(context.Background(), msg)
		assert.Error(t, err)
	})

	t.Run("empty OOB data", func(t *testing.T) {
		msg := &socketMsg{from: testAddr, port: 33434, oob: []byte{}}

		_, err := parseExtendedErr(context.Background(), msg)
		assert.Error(t, err)
	})
}

func Test_newSockExtendedErr(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		data := []byte{
			0x01, 0x00, 0x00, 0x00, // Errno: 1
			0x02,                   // Origin: 2
			0x0b,                   // Type: 11
			0x03,                   // Code: 3
			0x00,                   // Pad
			0x34, 0x12, 0x00, 0x00, // Info: 0x1234
			0x78, 0x56, 0x00, 0x00, // Data: 0x5678
		}

		got, err := newSockExtendedErr(data)

		assert.NoError(t, err)
		assert.Equal(t, unix.SockExtendedErr{
			Errno:  1,
			Origin: 2,
			Type:   11,
			Code:   3,
			Info:   0x1234,
			Data:   0x5678,
		}, got)
	})

	t.Run("data too short (only 3 bytes)", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03}
		_, err := newSockExtendedErr(data)

		assert.Error(t, err)
	})

	t.Run("minimum size with all zeros", func(t *testing.T) {
		data := make([]byte, minExtendedErrSize)

		got, err := newSockExtendedErr(data)

		assert.NoError(t, err)
		assert.Equal(t, unix.SockExtendedErr{}, got)
	})
}

// newExtendedErrOOB creates OOB data with IP_RECVERR control message containing extended error
func newExtendedErrOOB(icmpType, icmpCode uint8) []byte {
	extErrData := make([]byte, minExtendedErrSize)
	extErrData[5] = icmpType
	extErrData[6] = icmpCode
	return newControlMessage(unix.SOL_IP, unix.IP_RECVERR, extErrData)
}

// newControlMessage creates a control message with given level, type and data
func newControlMessage(level, msgType int, data []byte) []byte {
	cmsgLen := unix.CmsgLen(len(data))
	buf := make([]byte, cmsgLen)

	hdr := (*unix.Cmsghdr)(unsafe.Pointer(&buf[0]))
	hdr.Len = uint64(cmsgLen)
	hdr.Level = int32(level)
	hdr.Type = int32(msgType)

	copy(buf[unix.CmsgSpace(0):], data)
	return buf
}

func Test_recvMsg(t *testing.T) {
	// Store the original function to restore after tests
	origUnixRecvMsg := unixRecvMsg
	defer func() { unixRecvMsg = origUnixRecvMsg }()

	t.Run("successful message reception with valid UDP packet", func(t *testing.T) {
		// Create a mock IP packet with UDP header
		// IP header (20 bytes) + UDP header (8 bytes)
		// IP header starts with version (4) and header length (5 * 4 = 20 bytes)
		mockData := make([]byte, dataBufSize)
		mockData[0] = 0x45 // Version 4, header length 5 (20 bytes)
		// Skip to UDP header at offset 20
		// UDP destination port at offset 22-23 (big endian)
		binary.BigEndian.PutUint16(mockData[22:24], 33434) // destination port 33434

		mockOobData := []byte{0x01, 0x02, 0x03, 0x04}
		mockFrom := &unix.SockaddrInet4{
			Port: 12345,
			Addr: [4]byte{192, 168, 1, 1},
		}

		unixRecvMsg = func(fd int, p, oob []byte, flags int) (n, oobn, recvflags int, from unix.Sockaddr, err error) {
			copy(p, mockData)
			copy(oob, mockOobData)
			return len(mockData), len(mockOobData), 0, mockFrom, nil
		}

		result, err := recvMsg(123, make([]byte, oobBufSize), unix.MSG_ERRQUEUE)

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 33434, result.port)
		assert.Equal(t, len(mockOobData), len(result.oob))
		assert.Equal(t, mockOobData, result.oob)

		// Check that the address conversion worked
		expectedAddr := &net.IPAddr{IP: net.IPv4(192, 168, 1, 1)}
		assert.Equal(t, expectedAddr, result.from)
	})

	t.Run("unix.Recvmsg returns error", func(t *testing.T) {
		unixRecvMsg = func(fd int, p, oob []byte, flags int) (n, oobn, recvflags int, from unix.Sockaddr, err error) {
			return 0, 0, 0, nil, errors.New("socket error")
		}

		result, err := recvMsg(456, make([]byte, oobBufSize), unix.MSG_ERRQUEUE)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("received packet too small - less than minimum header size", func(t *testing.T) {
		// Create data that's too small (less than 24 bytes for IP header + UDP header)
		mockData := make([]byte, 10) // Only 10 bytes
		mockData[0] = 0x45           // Version 4, header length 5 (20 bytes)

		unixRecvMsg = func(fd int, p, oob []byte, flags int) (n, oobn, recvflags int, from unix.Sockaddr, err error) {
			copy(p, mockData)
			return 10, 0, 0, &unix.SockaddrInet4{}, nil
		}

		result, err := recvMsg(789, make([]byte, oobBufSize), unix.MSG_ERRQUEUE)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("received packet exactly minimum size", func(t *testing.T) {
		// Create minimum valid packet: 20-byte IP header + 4 bytes for UDP dest port
		mockData := make([]byte, 24)
		mockData[0] = 0x45                                 // Version 4, header length 5 (20 bytes)
		binary.BigEndian.PutUint16(mockData[22:24], 12345) // UDP destination port

		unixRecvMsg = func(fd int, p, oob []byte, flags int) (n, oobn, recvflags int, from unix.Sockaddr, err error) {
			copy(p, mockData)
			return 24, 0, 0, &unix.SockaddrInet4{Port: 80}, nil
		}

		result, err := recvMsg(999, make([]byte, oobBufSize), unix.MSG_ERRQUEUE)

		assert.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 12345, result.port)
	})

	t.Run("different IP header lengths", func(t *testing.T) {
		tests := []struct {
			name       string
			headerByte byte
			headerLen  int
			expectPort uint16
		}{
			{
				name:       "minimum IP header (20 bytes)",
				headerByte: 0x45, // Version 4, IHL 5 (5*4=20 bytes)
				headerLen:  20,
				expectPort: 8080,
			},
			{
				name:       "IP header with options (24 bytes)",
				headerByte: 0x46, // Version 4, IHL 6 (6*4=24 bytes)
				headerLen:  24,
				expectPort: 9090,
			},
			{
				name:       "maximum IP header (60 bytes)",
				headerByte: 0x4F, // Version 4, IHL 15 (15*4=60 bytes)
				headerLen:  60,
				expectPort: 1234,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockData := make([]byte, dataBufSize)
				mockData[0] = tt.headerByte
				// Place UDP destination port at correct offset
				binary.BigEndian.PutUint16(mockData[tt.headerLen+2:tt.headerLen+4], tt.expectPort)

				unixRecvMsg = func(fd int, p, oob []byte, flags int) (n, oobn, recvflags int, from unix.Sockaddr, err error) {
					copy(p, mockData)
					return tt.headerLen + 8, 0, 0, &unix.SockaddrInet4{}, nil // IP header + UDP header (8 bytes)
				}

				result, err := recvMsg(111, make([]byte, oobBufSize), unix.MSG_ERRQUEUE)

				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, int(tt.expectPort), result.port)
			})
		}
	})

	t.Run("out-of-band data handling", func(t *testing.T) {
		tests := []struct {
			name     string
			oobData  []byte
			oobLen   int
			expected []byte
		}{
			{
				name:     "no oob data",
				oobData:  []byte{},
				oobLen:   0,
				expected: []byte{},
			},
			{
				name:     "partial oob data",
				oobData:  []byte{0x01, 0x02, 0x03, 0x04, 0x05},
				oobLen:   3,
				expected: []byte{0x01, 0x02, 0x03},
			},
			{
				name:     "full oob data",
				oobData:  []byte{0xAA, 0xBB, 0xCC, 0xDD},
				oobLen:   4,
				expected: []byte{0xAA, 0xBB, 0xCC, 0xDD},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockData := make([]byte, dataBufSize)
				mockData[0] = 0x45                                 // Standard IP header
				binary.BigEndian.PutUint16(mockData[22:24], 54321) // UDP port

				unixRecvMsg = func(fd int, p, oob []byte, flags int) (n, oobn, recvflags int, from unix.Sockaddr, err error) {
					copy(p, mockData)
					copy(oob, tt.oobData)
					return 28, tt.oobLen, 0, &unix.SockaddrInet4{}, nil
				}

				result, err := recvMsg(222, make([]byte, oobBufSize), unix.MSG_ERRQUEUE)

				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expected, result.oob)
			})
		}
	})

	t.Run("different socket address types", func(t *testing.T) {
		tests := []struct {
			name         string
			sockAddr     unix.Sockaddr
			expectedAddr net.Addr
		}{
			{
				name: "IPv4 address",
				sockAddr: &unix.SockaddrInet4{
					Port: 8080,
					Addr: [4]byte{10, 0, 0, 1},
				},
				expectedAddr: &net.IPAddr{IP: net.IPv4(10, 0, 0, 1)},
			},
			{
				name: "IPv6 address",
				sockAddr: &unix.SockaddrInet6{
					Port: 9090,
					Addr: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, // ::1
				},
				expectedAddr: &net.IPAddr{IP: net.IPv6loopback},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockData := make([]byte, dataBufSize)
				mockData[0] = 0x45                                // Standard IP header
				binary.BigEndian.PutUint16(mockData[22:24], 7777) // UDP port

				unixRecvMsg = func(fd int, p, oob []byte, flags int) (n, oobn, recvflags int, from unix.Sockaddr, err error) {
					copy(p, mockData)
					return 28, 0, 0, tt.sockAddr, nil
				}

				result, err := recvMsg(333, make([]byte, oobBufSize), unix.MSG_ERRQUEUE)

				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedAddr, result.from)
			})
		}
	})

	t.Run("edge case - header length mask extraction", func(t *testing.T) {
		// Test that the header length mask (0x0F) correctly extracts the IHL field
		tests := []struct {
			name       string
			firstByte  byte
			expectedHL int // expected header length in bytes
		}{
			{"IHL=5, other bits set", 0xF5, 20},  // 11110101 -> IHL=5
			{"IHL=6, other bits set", 0xE6, 24},  // 11100110 -> IHL=6
			{"IHL=15, other bits set", 0xAF, 60}, // 10101111 -> IHL=15
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mockData := make([]byte, dataBufSize)
				mockData[0] = tt.firstByte
				// Place UDP destination port at correct offset
				binary.BigEndian.PutUint16(mockData[tt.expectedHL+2:tt.expectedHL+4], 4444)

				unixRecvMsg = func(fd int, p, oob []byte, flags int) (n, oobn, recvflags int, from unix.Sockaddr, err error) {
					copy(p, mockData)
					return tt.expectedHL + 8, 0, 0, &unix.SockaddrInet4{}, nil
				}

				result, err := recvMsg(444, make([]byte, oobBufSize), unix.MSG_ERRQUEUE)

				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 4444, result.port)
			})
		}
	})
}
