package rtnetlink

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rgwohlbold/rtnetlink/internal/unix"
	"github.com/mdlayher/netlink/nlenc"
)

func TestAddressMessageMarshalBinary(t *testing.T) {
	skipBigEndian(t)

	tests := []struct {
		name string
		m    Message
		b    []byte
		err  error
	}{
		{
			name: "empty",
			m:    &AddressMessage{},
			b: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "no attributes",
			m: &AddressMessage{
				Family:       unix.AF_INET,
				PrefixLength: 8,
				Scope:        0,
				Index:        1,
				Flags:        0,
			},
			b: []byte{
				0x02, 0x08, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "attributes",
			m: &AddressMessage{
				Attributes: &AddressAttributes{
					Address:   net.IPv4(192, 0, 2, 1),
					Broadcast: net.IPv4(255, 255, 255, 255),
					Label:     "lo",
				},
			},
			b: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x08, 0x00, 0x01, 0x00, 0xc0, 0x00, 0x02, 0x01,
				0x08, 0x00, 0x04, 0x00, 0xff, 0xff, 0xff, 0xff,
				0x07, 0x00, 0x03, 0x00, 0x6c, 0x6f, 0x00, 0x00,
				0x08, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "no broadcast",
			m: &AddressMessage{
				Attributes: &AddressAttributes{
					Address: net.ParseIP("2001:db8::"),
					Label:   "lo",
				},
			},
			b: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x14, 0x00, 0x01, 0x00, 0x20, 0x01, 0x0d, 0xb8,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x07, 0x00, 0x03, 0x00,
				0x6c, 0x6f, 0x00, 0x00, 0x08, 0x00, 0x08, 0x00,
				0x00, 0x00, 0x00, 0x00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.m.MarshalBinary()

			if want, got := tt.err, err; want != got {
				t.Fatalf("unexpected error:\n- want: %v\n-  got: %v", want, got)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tt.b, b); diff != "" {
				t.Fatalf("unexpected bytes (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAddressMessageUnmarshalBinary(t *testing.T) {
	skipBigEndian(t)

	tests := []struct {
		name string
		b    []byte
		m    Message
		ok   bool
	}{
		{
			name: "empty",
		},
		{
			name: "short",
			b:    make([]byte, 3),
		},
		{
			name: "invalid attr",
			b: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x08, 0x00, 0x01, 0x00, 0xc0, 0x00, 0x02, 0x01,
				0x08, 0x00, 0x06, 0x00, 0xff, 0xff, 0xff, 0xff,
				0x08, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			name: "data",
			b: []byte{
				0x02, 0x08, 0xfe, 0x01, 0x01, 0x00, 0x00, 0x00,
				0x08, 0x00, 0x01, 0x00, 0x7f, 0x00, 0x00, 0x01,
				0x08, 0x00, 0x02, 0x00, 0x7f, 0x00, 0x00, 0x01,
				0x07, 0x00, 0x03, 0x00, 0x6c, 0x6f, 0x00, 0x00,
				0x08, 0x00, 0x08, 0x00, 0x80, 0x00, 0x00, 0x00,
				0x14, 0x00, 0x06, 0x00, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0x44, 0x01, 0x00,
				0x00, 0x44, 0x01, 0x00, 0x00,
			},
			m: &AddressMessage{
				Family:       2,
				PrefixLength: 8,
				Flags:        0xfe,
				Scope:        1,
				Index:        1,
				Attributes: &AddressAttributes{
					Address:   net.IP{0x7f, 0x0, 0x0, 0x1},
					Local:     net.IP{0x7f, 0x0, 0x0, 0x1},
					Label:     "lo",
					Broadcast: net.IP(nil),
					Anycast:   net.IP(nil),
					CacheInfo: CacheInfo{
						Prefered: 0xffffffff,
						Valid:    0xffffffff,
						Created:  0x144,
						Updated:  0x144,
					},
					Multicast: net.IP(nil),
					Flags:     0x80,
				},
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m AddressMessage
			err := m.UnmarshalBinary(tt.b)

			if tt.ok && err != nil {
				t.Fatalf("failed to unmarshal binary: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("expected an error, but none occurred")
			}
			if err != nil {
				t.Logf("err: %v", err)
				return
			}

			if diff := cmp.Diff(tt.m, &m); diff != "" {
				t.Fatalf("unexpected AddressMessage (-want +got):\n%s", diff)
			}
		})
	}
}

func skipBigEndian(t *testing.T) {
	if nlenc.NativeEndian() == binary.BigEndian {
		t.Skip("skipping test on big-endian system")
	}
}
