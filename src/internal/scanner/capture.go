package scanner

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcapgo"
)

type packetReader interface {
	ReadPacketData() ([]byte, gopacket.CaptureInfo, error)
	LinkType() layers.LinkType
}

func ReadCapture(input io.Reader, source string) (*Scan, error) {
	data, e := io.ReadAll(io.LimitReader(input, (64<<20)+1))
	if e != nil {
		return nil, e
	}
	if len(data) > 64<<20 {
		return nil, fmt.Errorf("capture exceeds 64 MB")
	}
	if e = validateContainer(data); e != nil {
		return nil, e
	}
	b := bufio.NewReader(bytes.NewReader(data))
	magic, e := b.Peek(4)
	if e != nil {
		return nil, fmt.Errorf("capture header: %w", e)
	}
	var r packetReader
	if bytes.Equal(magic, []byte{10, 13, 13, 10}) {
		r, e = pcapgo.NewNgReader(b, pcapgo.NgReaderOptions{ErrorOnMismatchingLinkType: true})
	} else {
		r, e = pcapgo.NewReader(b)
	}
	if e != nil {
		return nil, fmt.Errorf("invalid capture: %w", e)
	}
	if r.LinkType() != layers.LinkTypeIEEE802_11 && r.LinkType() != layers.LinkTypeIEEE80211Radio {
		return nil, fmt.Errorf("capture requires raw 802.11 or radiotap; got %s (Ethernet captures cannot reveal Wi-Fi security)", r.LinkType())
	}
	s := &Scan{Source: source, CreatedAt: time.Now().UTC(), APs: []AP{}, Warnings: []string{}}
	aps := map[string]AP{}
	malformed := 0
	for count := 0; ; count++ {
		if count >= 250000 {
			s.Warnings = append(s.Warnings, "Stopped at 250,000 packets; results may be incomplete.")
			break
		}
		raw, ci, err := r.ReadPacketData()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("capture packet: %w", err)
		}
		a, err := parseFrame(raw, r.LinkType(), ci.Timestamp)
		if err != nil {
			malformed++
			continue
		}
		if a == nil {
			continue
		}
		if old, ok := aps[a.BSSID]; ok {
			a.Observations = old.Observations + 1
			a.FirstSeen = old.FirstSeen
			if a.SSID == "" && old.SSID != "" {
				a.SSID = old.SSID
			}
			if a.LastSeen.Before(old.LastSeen) {
				old.Observations = a.Observations
				aps[a.BSSID] = old
				continue
			}
		}
		aps[a.BSSID] = *a
	}
	for _, a := range aps {
		s.APs = append(s.APs, a)
	}
	slices.SortFunc(s.APs, func(a, b AP) int { return strings.Compare(a.SSID+a.BSSID, b.SSID+b.BSSID) })
	if malformed > 0 {
		s.Warnings = append(s.Warnings, fmt.Sprintf("Skipped %d malformed management frames or radiotap headers.", malformed))
	}
	if len(s.APs) == 0 {
		s.Warnings = append(s.Warnings, "No beacon or probe-response frames found. Check the monitor interface, channel, and capture duration.")
	}
	return s, nil
}

func parseFrame(raw []byte, link layers.LinkType, ts time.Time) (*AP, error) {
	freq := 0
	var rssi *int
	if link == layers.LinkTypeIEEE80211Radio {
		payload, f, signal, err := readRadiotap(raw)
		if err != nil {
			return nil, err
		}
		raw, freq, rssi = payload, f, signal
	}
	if len(raw) < 2 {
		return nil, errors.New("short frame")
	}
	if raw[0]&0xfc != 0x80 && raw[0]&0xfc != 0x50 {
		return nil, nil
	}
	if len(raw) < 36 {
		return nil, errors.New("short management frame")
	}
	if raw[16]&1 != 0 {
		return nil, errors.New("invalid BSSID")
	}
	a := &AP{BSSID: net.HardwareAddr(raw[16:22]).String(), Vendor: "Unknown", Frequency: freq, RSSI: rssi, Generation: "Unknown", BeaconInterval: int(binary.LittleEndian.Uint16(raw[32:34])), FirstSeen: ts, LastSeen: ts, Observations: 1, Security: Security{Protocol: "Open", AKM: []string{}, Ciphers: []string{}, PMF: "unknown"}}
	privacy := binary.LittleEndian.Uint16(raw[34:36])&0x10 != 0
	if privacy {
		a.Security.Protocol = "WEP / unknown legacy"
	}
	ies := raw[36:]
	hasRSN, hasWPA := false, false
	for len(ies) > 0 {
		if len(ies) < 2 || int(ies[1])+2 > len(ies) {
			return nil, errors.New("truncated information element")
		}
		id, d := ies[0], ies[2:2+int(ies[1])]
		ies = ies[2+int(ies[1]):]
		switch id {
		case 0:
			if len(d) > 32 {
				return nil, errors.New("SSID too long")
			}
			a.SSID = strings.ToValidUTF8(string(d), "�")
			a.Hidden = len(d) == 0 || bytes.Equal(d, make([]byte, len(d)))
			if a.Hidden {
				a.SSID = ""
			}
		case 3:
			if len(d) == 1 {
				a.Channel = int(d[0])
			}
		case 45:
			a.Generation = "4"
			a.Width = 20
			if len(d) >= 2 && binary.LittleEndian.Uint16(d)&2 != 0 {
				a.Width = 40
			}
		case 61:
			if len(d) >= 2 {
				a.Channel = int(d[0])
				a.Width = 20
				if d[1]&4 != 0 && d[1]&3 != 0 {
					a.Width = 40
				}
			}
		case 191:
			a.Generation = "5"
		case 192:
			if len(d) >= 3 {
				switch d[0] {
				case 1:
					a.Width = 80
					if d[2] != 0 {
						a.Width = 160
					}
				case 2, 3:
					a.Width = 160
				}
			}
		case 255:
			if len(d) > 0 {
				if d[0] == 35 && a.Generation != "7" {
					a.Generation = "6"
				}
				if d[0] == 108 {
					a.Generation = "7"
				}
			}
		case 48:
			hasRSN = true
			sec, err := parseRSN(d, false)
			if err != nil {
				a.Security = Security{Protocol: "Unknown", PMF: "unknown", AKM: []string{}, Ciphers: []string{}}
			} else {
				a.Security = sec
			}
		case 221:
			if len(d) >= 4 && bytes.Equal(d[:4], []byte{0, 80, 242, 1}) {
				hasWPA = true
				if !hasRSN {
					sec, err := parseRSN(d[4:], true)
					if err == nil {
						a.Security = sec
					} else {
						a.Security.Protocol = "Unknown"
					}
				}
			}
		}
	}
	if hasRSN && hasWPA {
		a.Security.Protocol = "WPA1/" + a.Security.Protocol
	}
	if freq > 0 {
		a.Band, a.Channel = frequencyChannel(freq)
	} else if a.Channel > 0 && a.Channel <= 14 {
		a.Band = "2.4"
		a.Frequency = 2407 + 5*a.Channel
		if a.Channel == 14 {
			a.Frequency = 2484
		}
	} else {
		a.Band = "unknown"
	}
	return a, nil
}

func frequencyChannel(f int) (string, int) {
	switch {
	case f == 2484:
		return "2.4", 14
	case f >= 2412 && f <= 2472:
		return "2.4", (f - 2407) / 5
	case f == 5935:
		return "6", 2
	case f >= 5955 && f <= 7115:
		return "6", (f - 5950) / 5
	case f >= 5000 && f < 5925:
		return "5", (f - 5000) / 5
	}
	return "unknown", 0
}

func parseRSN(d []byte, wpa bool) (Security, error) {
	s := Security{Protocol: "WPA2", PMF: "disabled", AKM: []string{}, Ciphers: []string{}}
	if len(d) < 8 || binary.LittleEndian.Uint16(d) != 1 {
		return s, errors.New("invalid RSN version or length")
	}
	oui := []byte{0, 15, 172}
	if wpa {
		oui = []byte{0, 80, 242}
		s.Protocol = "WPA1"
		s.PMF = "unknown"
	}
	cipher := func(suite []byte) string {
		if !bytes.Equal(suite[:3], oui) {
			return "Unknown"
		}
		switch suite[3] {
		case 1:
			return "WEP-40"
		case 2:
			return "TKIP"
		case 4:
			return "CCMP"
		case 5:
			return "WEP-104"
		case 8:
			return "GCMP-128"
		case 9:
			return "GCMP-256"
		case 10:
			return "CCMP-256"
		}
		return fmt.Sprintf("Unknown (%d)", suite[3])
	}
	s.Ciphers = append(s.Ciphers, cipher(d[2:6]))
	d = d[6:]
	readList := func() ([][]byte, error) {
		if len(d) < 2 {
			return nil, errors.New("missing suite count")
		}
		n := int(binary.LittleEndian.Uint16(d))
		d = d[2:]
		if n < 1 || n > len(d)/4 {
			return nil, errors.New("invalid suite count")
		}
		out := [][]byte{}
		for k := 0; k < n; k++ {
			out = append(out, d[:4])
			d = d[4:]
		}
		return out, nil
	}
	c, e := readList()
	if e != nil {
		return s, e
	}
	for _, v := range c {
		s.Ciphers = append(s.Ciphers, cipher(v))
	}
	slices.Sort(s.Ciphers)
	s.Ciphers = slices.Compact(s.Ciphers)
	akms, e := readList()
	if e != nil {
		return s, e
	}
	for _, v := range akms {
		name := "Unknown"
		if bytes.Equal(v[:3], oui) {
			switch v[3] {
			case 1:
				name = "802.1X"
			case 2:
				name = "PSK"
			case 3:
				name = "FT-802.1X"
			case 4:
				name = "FT-PSK"
			case 5:
				name = "802.1X-SHA256"
			case 6:
				name = "PSK-SHA256"
			case 8:
				name = "SAE"
			case 9:
				name = "FT-SAE"
			case 11:
				name = "802.1X-Suite-B"
			case 12:
				name = "802.1X-Suite-B-192"
			case 18:
				name = "OWE"
			}
		}
		s.AKM = append(s.AKM, name)
	}
	if !wpa && len(d) == 1 {
		return s, errors.New("truncated RSN capabilities")
	}
	if !wpa && len(d) >= 2 {
		caps := binary.LittleEndian.Uint16(d)
		if caps&(1<<7) != 0 {
			s.PMF = "capable"
		}
		if caps&(1<<6) != 0 {
			if caps&(1<<7) == 0 {
				return s, errors.New("invalid PMF capability bits")
			}
			s.PMF = "required"
		}
	}
	if !wpa {
		sae := slices.Contains(s.AKM, "SAE") || slices.Contains(s.AKM, "FT-SAE")
		psk := slices.Contains(s.AKM, "PSK") || slices.Contains(s.AKM, "FT-PSK") || slices.Contains(s.AKM, "PSK-SHA256")
		switch {
		case slices.Contains(s.AKM, "Unknown"):
			s.Protocol = "Unknown"
		case sae && psk:
			s.Protocol = "WPA2/WPA3 transition"
		case sae:
			s.Protocol = "WPA3"
		case slices.Contains(s.AKM, "OWE"):
			s.Protocol = "OWE"
		case slices.Contains(s.AKM, "802.1X-Suite-B-192"):
			s.Protocol = "WPA3-Enterprise"
		case !psk:
			s.Protocol = "WPA2/3-Enterprise"
		}
	}
	for _, cipher := range s.Ciphers {
		if strings.HasPrefix(cipher, "Unknown") {
			s.Protocol = "Unknown"
		}
	}
	return s, nil
}

// Read only metadata needed for observations, with bounds checked before each field.
// Later radiotap fields and namespaces are skipped using the declared header length.
func readRadiotap(raw []byte) ([]byte, int, *int, error) {
	bad := func() ([]byte, int, *int, error) {
		return nil, 0, nil, errors.New("truncated or invalid radiotap header")
	}
	if len(raw) < 8 || raw[0] != 0 {
		return bad()
	}
	hlen := int(binary.LittleEndian.Uint16(raw[2:4]))
	if hlen < 8 || hlen > len(raw) {
		return bad()
	}
	present := binary.LittleEndian.Uint32(raw[4:8])
	next := present
	offset := 8
	for next&(1<<31) != 0 {
		if offset+4 > hlen {
			return bad()
		}
		next = binary.LittleEndian.Uint32(raw[offset : offset+4])
		offset += 4
	}
	sizes := []int{8, 1, 1, 4, 2, 1}
	aligns := []int{8, 1, 1, 2, 2, 1}
	frequency := 0
	var signal *int
	flags := byte(0)
	for bit := 0; bit <= 5; bit++ {
		if present&(1<<bit) == 0 {
			continue
		}
		align := aligns[bit]
		offset = (offset + align - 1) &^ (align - 1)
		if offset+sizes[bit] > hlen {
			return bad()
		}
		field := raw[offset : offset+sizes[bit]]
		switch bit {
		case 1:
			flags = field[0]
		case 3:
			frequency = int(binary.LittleEndian.Uint16(field))
		case 5:
			x := int(int8(field[0]))
			signal = &x
		}
		offset += sizes[bit]
	}
	payload := raw[hlen:]
	if flags&0x40 != 0 {
		return nil, 0, nil, errors.New("radiotap marks frame as bad FCS")
	}
	if flags&0x10 != 0 {
		if len(payload) < 4 {
			return bad()
		}
		payload = payload[:len(payload)-4]
	}
	return payload, frequency, signal, nil
}

// Reject impossible sizes before pcapgo allocates from untrusted length fields.
func validateContainer(data []byte) error {
	if len(data) < 4 {
		return errors.New("capture header is truncated")
	}
	if bytes.Equal(data[:4], []byte{10, 13, 13, 10}) {
		var order binary.ByteOrder = binary.LittleEndian
		for offset := 0; offset < len(data); {
			if len(data)-offset < 12 {
				return errors.New("truncated pcapng block")
			}
			if bytes.Equal(data[offset:offset+4], []byte{10, 13, 13, 10}) {
				switch {
				case bytes.Equal(data[offset+8:offset+12], []byte{77, 60, 43, 26}):
					order = binary.LittleEndian
				case bytes.Equal(data[offset+8:offset+12], []byte{26, 43, 60, 77}):
					order = binary.BigEndian
				default:
					return errors.New("invalid pcapng byte order")
				}
			}
			size := int(order.Uint32(data[offset+4 : offset+8]))
			if size < 12 || size%4 != 0 || size > 4<<20 || size > len(data)-offset {
				return errors.New("invalid or oversized pcapng block (maximum 4 MB)")
			}
			if int(order.Uint32(data[offset+size-4:offset+size])) != size {
				return errors.New("pcapng block length mismatch")
			}
			offset += size
		}
		return nil
	}
	if len(data) < 24 {
		return errors.New("pcap global header is truncated")
	}
	var order binary.ByteOrder
	switch {
	case bytes.Equal(data[:4], []byte{212, 195, 178, 161}), bytes.Equal(data[:4], []byte{77, 60, 178, 161}):
		order = binary.LittleEndian
	case bytes.Equal(data[:4], []byte{161, 178, 195, 212}), bytes.Equal(data[:4], []byte{161, 178, 60, 77}):
		order = binary.BigEndian
	default:
		return errors.New("unrecognized capture format; use PCAP or PCAPNG")
	}
	for offset := 24; offset < len(data); {
		if len(data)-offset < 16 {
			return errors.New("truncated pcap packet header")
		}
		size := int(order.Uint32(data[offset+8 : offset+12]))
		offset += 16
		if size > 65535 || size > len(data)-offset {
			return errors.New("invalid packet size (maximum 65535 bytes)")
		}
		offset += size
	}
	return nil
}
