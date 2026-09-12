package nfc

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type Record struct {
	Type       string `json:"type"`
	TNF        int    `json:"tnf"`
	ID         string `json:"id,omitempty"`
	Text       string `json:"text,omitempty"`
	Language   string `json:"language,omitempty"`
	URI        string `json:"uri,omitempty"`
	PayloadHex string `json:"payload_hex"`
}

var uriPrefixes = []string{"", "http://www.", "https://www.", "http://", "https://", "tel:", "mailto:", "ftp://anonymous:anonymous@", "ftp://ftp.", "ftps://", "sftp://", "smb://", "nfs://", "ftp://", "dav://", "news:", "telnet://", "imap:", "rtsp://", "urn:", "pop:", "sip:", "sips:", "tftp:", "btspp://", "btl2cap://", "btgoep://", "tcpobex://", "irdaobex://", "file://", "urn:epc:id:", "urn:epc:tag:", "urn:epc:pat:", "urn:epc:raw:", "urn:epc:", "urn:nfc:"}

// ParseNDEF validates record boundaries. Chunked records are deliberately unsupported.
func ParseNDEF(data []byte) ([]Record, error) {
	if len(data) > 4096 {
		return nil, fmt.Errorf("NDEF exceeds 4096 bytes")
	}
	records := []Record{}
	first := true
	for len(data) > 0 {
		if len(records) >= 64 {
			return nil, fmt.Errorf("too many NDEF records")
		}
		if len(data) < 3 {
			return nil, fmt.Errorf("truncated NDEF record header")
		}
		header, tl := data[0], int(data[1])
		data = data[2:]
		if (header&0x80 != 0) != first {
			return nil, fmt.Errorf("invalid NDEF message-begin flag")
		}
		first = false
		if header&0x20 != 0 {
			return nil, fmt.Errorf("chunked NDEF records are unsupported")
		}
		var pl uint32
		if header&0x10 != 0 {
			pl = uint32(data[0])
			data = data[1:]
		} else {
			if len(data) < 4 {
				return nil, fmt.Errorf("truncated payload length")
			}
			pl = binary.BigEndian.Uint32(data[:4])
			data = data[4:]
		}
		il := 0
		if header&8 != 0 {
			if len(data) < 1 {
				return nil, fmt.Errorf("truncated ID length")
			}
			il = int(data[0])
			data = data[1:]
		}
		if uint64(tl)+uint64(il)+uint64(pl) > uint64(len(data)) {
			return nil, fmt.Errorf("NDEF payload exceeds message boundary")
		}
		kind := string(data[:tl])
		identifier := hex.EncodeToString(data[tl : tl+il])
		payload := data[tl+il : tl+il+int(pl)]
		data = data[tl+il+int(pl):]
		tnf := int(header & 7)
		if tnf == 6 || tnf == 7 {
			return nil, fmt.Errorf("unsupported NDEF TNF")
		}
		if tnf == 0 && (tl != 0 || il != 0 || pl != 0) {
			return nil, fmt.Errorf("nonempty empty-type record")
		}
		if tnf == 5 && tl != 0 {
			return nil, fmt.Errorf("unknown TNF must have empty type")
		}
		r := Record{Type: strings.ToValidUTF8(kind, "�"), TNF: tnf, ID: identifier, PayloadHex: hex.EncodeToString(payload)}
		if tnf == 1 && kind == "T" {
			if len(payload) == 0 {
				return nil, fmt.Errorf("empty text record")
			}
			lang := int(payload[0] & 63)
			if lang+1 > len(payload) {
				return nil, fmt.Errorf("text language length exceeds payload")
			}
			r.Language = string(payload[1 : 1+lang])
			body := payload[1+lang:]
			if payload[0]&0x80 == 0 {
				if !utf8.Valid(body) {
					return nil, fmt.Errorf("invalid UTF-8 NDEF text")
				}
				r.Text = string(body)
			} else {
				if len(body)%2 != 0 {
					return nil, fmt.Errorf("invalid UTF-16 NDEF text")
				}
				var order binary.ByteOrder = binary.BigEndian
				if len(body) >= 2 {
					if body[0] == 255 && body[1] == 254 {
						order = binary.LittleEndian
						body = body[2:]
					} else if body[0] == 254 && body[1] == 255 {
						body = body[2:]
					}
				}
				units := []uint16{}
				for len(body) > 0 {
					units = append(units, order.Uint16(body[:2]))
					body = body[2:]
				}
				r.Text = string(utf16.Decode(units))
			}
		}
		if tnf == 1 && kind == "U" {
			if len(payload) < 1 || int(payload[0]) >= len(uriPrefixes) {
				return nil, fmt.Errorf("invalid URI prefix")
			}
			if !utf8.Valid(payload[1:]) {
				return nil, fmt.Errorf("invalid URI encoding")
			}
			r.URI = uriPrefixes[int(payload[0])] + string(payload[1:])
		}
		records = append(records, r)
		if header&0x40 != 0 {
			if len(data) != 0 {
				return nil, fmt.Errorf("trailing bytes after NDEF message-end")
			}
			return records, nil
		}
	}
	if len(records) > 0 {
		return nil, fmt.Errorf("missing NDEF message-end")
	}
	return records, nil
}
