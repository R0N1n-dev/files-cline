package fs

import "encoding/binary"

type ifdEntry struct {
	tag    uint16
	typ    uint16
	count  uint32
	offset uint32
	raw    []byte
}

// parseTIFFOrientation parses IFD0 (+ EXIF SubIFD) of a raw TIFF/Exif payload.
func parseTIFFOrientation(b []byte) *EXIFMetadata {
	if len(b) < 8 {
		return nil
	}
	var order binary.ByteOrder
	switch {
	case b[0] == 'I' && b[1] == 'I':
		order = binary.LittleEndian
	case b[0] == 'M' && b[1] == 'M':
		order = binary.BigEndian
	default:
		return nil
	}
	if order.Uint16(b[2:4]) != 42 {
		return nil
	}

	meta := &EXIFMetadata{Orientation: 1, HasEXIF: true}

	readIFD := func(ifdOffset uint32, allowSubIFD bool) {
		if int(ifdOffset)+2 > len(b) {
			return
		}
		count := int(order.Uint16(b[ifdOffset : ifdOffset+2]))
		entryBase := int(ifdOffset) + 2
		for i := 0; i < count; i++ {
			off := entryBase + i*12
			if off+12 > len(b) {
				return
			}
			e := ifdEntry{
				tag:    order.Uint16(b[off : off+2]),
				typ:    order.Uint16(b[off+2 : off+4]),
				count:  order.Uint32(b[off+4 : off+8]),
				offset: order.Uint32(b[off+8 : off+12]),
				raw:    b[off+8 : off+12],
			}
			setEXIFField(meta, e, b, order, allowSubIFD)
		}
	}

	readIFD(order.Uint32(b[4:8]), true)
	return meta
}

func setEXIFField(meta *EXIFMetadata, e ifdEntry, b []byte, order binary.ByteOrder, allowSubIFD bool) {
	typeSize := map[uint16]int{1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 7: 1, 9: 4, 10: 8}
	size, ok := typeSize[e.typ]
	if !ok || size == 0 {
		return
	}
	total := size * int(e.count)
	if total <= 0 {
		return
	}
	var data []byte
	if total <= 4 {
		data = e.raw[:total]
	} else {
		if int(e.offset)+total > len(b) || int(e.offset) < 0 {
			return
		}
		data = b[e.offset : int(e.offset)+total]
	}

	asString := func() string {
		out := make([]byte, 0, len(data))
		for _, c := range data {
			if c == 0 {
				break
			}
			out = append(out, c)
		}
		return string(out)
	}
	asUint := func() uint32 {
		switch e.typ {
		case 3:
			if len(data) >= 2 {
				return uint32(order.Uint16(data[:2]))
			}
		case 4:
			if len(data) >= 4 {
				return order.Uint32(data[:4])
			}
		}
		return 0
	}
	asRational := func() string {
		if e.typ != 5 || len(data) < 8 {
			return ""
		}
		return rationalString(order.Uint32(data[0:4]), order.Uint32(data[4:8]))
	}

	switch e.tag {
	case 0x0112:
		if v := asUint(); v >= 1 && v <= 8 {
			meta.Orientation = int(v)
		}
	case 0x010F:
		meta.Make = asString()
	case 0x0110:
		meta.Model = asString()
	case 0x0132:
		meta.DateTime = asString()
	case 0xA002:
		meta.PixelWidth = int(asUint())
	case 0xA003:
		meta.PixelHeight = int(asUint())
	case 0xA434:
		meta.LensModel = asString()
	case 0x829A:
		if s := asRational(); s != "" {
			meta.ExposureTime = s + "s"
		}
	case 0x829D:
		if s := asRational(); s != "" {
			meta.FNumber = "f/" + s
		}
	case 0x8827:
		meta.ISO = int(asUint())
	case 0x920A:
		if s := asRational(); s != "" {
			meta.FocalLength = s + "mm"
		}
	case 0x8769:
		if allowSubIFD {
			readSubIFD(meta, e.offset, b, order)
		}
	}
}

func readSubIFD(meta *EXIFMetadata, sub uint32, b []byte, order binary.ByteOrder) {
	if int(sub)+2 > len(b) {
		return
	}
	count := int(order.Uint16(b[sub : sub+2]))
	base := int(sub) + 2
	for i := 0; i < count; i++ {
		off := base + i*12
		if off+12 > len(b) {
			return
		}
		subEntry := ifdEntry{
			tag:    order.Uint16(b[off : off+2]),
			typ:    order.Uint16(b[off+2 : off+4]),
			count:  order.Uint32(b[off+4 : off+8]),
			offset: order.Uint32(b[off+8 : off+12]),
			raw:    b[off+8 : off+12],
		}
		setEXIFField(meta, subEntry, b, order, false)
	}
}

func rationalString(num, den uint32) string {
	if den == 0 {
		return ""
	}
	if den == 1 {
		return itoa(num)
	}
	return strconvFormatFloat(float64(num) / float64(den))
}

func itoa(v uint32) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
