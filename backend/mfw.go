package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// decodeLatin1 converts Windows-1252 / Latin-1 bytes (used by Delphi Mimosa .mfw files) to UTF-8.
func decodeLatin1(b []byte) string {
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = rune(c)
	}
	return string(runes)
}

func readShortString(b []byte, off int, maxLen int) string {
	if off < 0 || off >= len(b) {
		return ""
	}
	l := int(b[off])
	if l > maxLen {
		l = maxLen
	}
	if off+1+l > len(b) {
		l = len(b) - off - 1
	}
	if l <= 0 {
		return ""
	}
	return strings.TrimSpace(decodeLatin1(b[off+1 : off+1+l]))
}

// ImportMFW parses a native Mimosa 5.x binary .mfw file based on our reverse-engineered format specification.
func ImportMFW(data []byte, fileName string) (*Project, error) {
	if len(data) < 0xE00 {
		return nil, errors.New("archivo .mfw demasiado pequeño o formato inválido")
	}

	p := NewEmptyProject("es")
	if fileName != "" {
		p.Name = strings.TrimSuffix(fileName, ".mfw")
	}

	numCodes := int(binary.LittleEndian.Uint16(data[0x2E:0x30]))
	totalSched := int(binary.LittleEndian.Uint16(data[0x22:0x24]))

	firstWeek := int(data[0x66])
	maxWeeks := int(data[0x67])
	maxDays := int(data[0x68])
	maxPeriods := int(data[0x69])
	if firstWeek < 1 {
		firstWeek = 1
	}
	if maxWeeks < 1 || maxWeeks > 255 {
		maxWeeks = 1
	}
	if maxDays < 1 || maxDays > 7 {
		maxDays = 5
	}
	if maxPeriods < 1 || maxPeriods > 30 {
		maxPeriods = 10
	}

	p.Time.FirstWeek = firstWeek
	p.Time.MaxWeeks = maxWeeks
	p.Time.MaxDays = maxDays
	p.Time.MaxPeriods = maxPeriods

	// Days string at 0x11E
	daysRaw := readShortString(data, 0x11E, 60)
	if daysRaw != "" {
		parts := strings.Split(daysRaw, "$")
		days := []string{}
		for _, d := range parts {
			if strings.TrimSpace(d) != "" {
				days = append(days, strings.TrimSpace(d))
			}
		}
		if len(days) >= 5 {
			p.Time.Days = days
		}
	}

	// Bells (30 x 12 bytes starting at 0x15C)
	bells := make([]string, 0, maxPeriods)
	for i := 0; i < 30; i++ {
		off := 0x15C + i*12
		s := readShortString(data, off, 11)
		if s != "" {
			bells = append(bells, s)
		} else {
			h := (8 + i) % 24
			bells = append(bells, fmt.Sprintf("%02d:00-%02d:45", h, h))
		}
	}
	p.Time.Bells = bells

	// Categories at 0x2C8 (courses), 0x3C8 (resources), 0x4C8 (bookings)
	if ccRaw := readShortString(data, 0x2C8, 255); ccRaw != "" {
		p.CourseCategories = parseCategoryString(ccRaw, false)
	}
	if rcRaw := readShortString(data, 0x3C8, 255); rcRaw != "" {
		p.ResourceCategories = parseCategoryString(rcRaw, true)
	}
	if bkRaw := readShortString(data, 0x4C8, 255); bkRaw != "" {
		bks := []string{}
		for _, tok := range strings.Split(bkRaw, "$") {
			if strings.TrimSpace(tok) != "" {
				bks = append(bks, strings.TrimSpace(tok))
			}
		}
		if len(bks) > 0 {
			p.BookingTypes = bks
		}
	}

	// Week names starting at 0x5D3 (stride 8 bytes)
	p.Time.Weeks = make([]WeekDef, maxWeeks)
	for i := 0; i < maxWeeks; i++ {
		wName := readShortString(data, 0x5D3+i*8, 7)
		if wName == "" {
			wName = fmt.Sprintf("S%d", i+1)
		}
		p.Time.Weeks[i] = WeekDef{Index: i + 1, Name: wName, Length: 1}
	}

	// Locate Chunk 10 header: uint16(10), uint16(16) around 0xD80..0xE40 (handles both v5128 and v5129)
	pos := 0xDE9
	for off := 0xD80; off <= 0xE40 && off+4 < len(data); off++ {
		if binary.LittleEndian.Uint16(data[off:off+2]) == 10 && binary.LittleEndian.Uint16(data[off+2:off+4]) == 16 {
			pos = off + 4
			break
		}
	}
	if pos+numCodes*16 > len(data) {
		return nil, errors.New("tabla de códigos truncada en .mfw")
	}
	rawCodes := make([]string, numCodes+1)
	for i := 1; i <= numCodes; i++ {
		rawCodes[i] = readShortString(data, pos, 15)
		pos += 16
	}

	// Chunk 11 header (4 bytes: tag=11, recSize=14) followed by numCodes x 14 bytes
	if pos+4+numCodes*14 > len(data) {
		return nil, errors.New("metadatos de códigos truncados en .mfw")
	}
	pos += 4

	type codeMeta struct {
		isCourse bool
		cat      int
		planned  int
		id       string
	}
	metas := make([]codeMeta, numCodes+1)
	cNum, rNum := 0, 0
	for i := 1; i <= numCodes; i++ {
		flags := binary.LittleEndian.Uint16(data[pos+4 : pos+6])
		maxP := int(binary.LittleEndian.Uint16(data[pos+10 : pos+12]))
		pos += 14

		isCourse := (flags & 0x4000) != 0
		cat := 1
		for bit := 0; bit < 7; bit++ {
			if (flags & (uint16(0x0080) << bit)) != 0 {
				cat = bit + 1
				break
			}
		}
		if isCourse {
			cNum++
			id := fmt.Sprintf("c_%d", cNum)
			metas[i] = codeMeta{isCourse: true, cat: cat, planned: maxP, id: id}
			p.Courses = append(p.Courses, Course{
				ID: id, Code: rawCodes[i], Name: rawCodes[i], Category: cat, Planned: maxP, Resources: []string{},
			})
		} else {
			rNum++
			id := fmt.Sprintf("r_%d", rNum)
			metas[i] = codeMeta{isCourse: false, cat: cat, planned: maxP, id: id}
			p.Resources = append(p.Resources, Resource{
				ID: id, Code: rawCodes[i], Name: rawCodes[i], Category: cat, Planned: maxP,
			})
		}
	}

	// Scan forward for chunks 21 (owners), 22 (start slots), 23 (end slots), 24 (collections)
	var owners, starts, ends, colls []uint16
	readChunkU16 := func(tag uint16) []uint16 {
		// Search from pos for uint16(tag) followed by a plausible even byte length
		for off := pos; off+4 <= len(data); off += 1 {
			t := binary.LittleEndian.Uint16(data[off : off+2])
			sz := int(binary.LittleEndian.Uint16(data[off+2 : off+4]))
			if t == tag && sz >= 2 && sz%2 == 0 && off+4+sz <= len(data) {
				if tag >= 21 && tag <= 23 && totalSched > 0 && sz != (totalSched+1)*2 {
					continue
				}
				// For chunk 21..24, element 0 is always 0
				if binary.LittleEndian.Uint16(data[off+4:off+6]) != 0 {
					continue
				}
				arr := make([]uint16, sz/2)
				for k := 0; k < len(arr); k++ {
					arr[k] = binary.LittleEndian.Uint16(data[off+4+k*2 : off+6+k*2])
				}
				pos = off + 4 + sz
				return arr
			}
		}
		return nil
	}

	if totalSched > 0 {
		owners = readChunkU16(21)
		starts = readChunkU16(22)
		ends = readChunkU16(23)
	}
	colls = readChunkU16(24)

	// Parse Collections from chunk 24
	if len(colls) > 1 {
		curCourseID := ""
		for i := 1; i < len(colls); i++ {
			v := colls[i]
			if v == 0 {
				continue
			}
			if (v & 0x4000) != 0 {
				cIdx := int(v & 0x1FFF)
				if cIdx >= 1 && cIdx <= numCodes && metas[cIdx].isCourse {
					curCourseID = metas[cIdx].id
				} else {
					curCourseID = ""
				}
			} else if curCourseID != "" {
				rIdx := int(v & 0x1FFF)
				if rIdx >= 1 && rIdx <= numCodes && !metas[rIdx].isCourse {
					if c := p.CourseByID(curCourseID); c != nil {
						c.Resources = append(c.Resources, metas[rIdx].id)
					}
				}
			}
		}
	}

	// Parse Week Allocations & Timetable Slots from chunks 21/22/23
	slotOrder := 0
	bookingOrder := 0
	if len(owners) > 1 && len(starts) == len(owners) && len(ends) == len(owners) {
		for i := 1; i < len(owners); i++ {
			ov := owners[i]
			sv := starts[i]
			ev := ends[i]
			if ov == 0 {
				continue
			}
			codeIdx := int(ov & 0x1FFF)
			if codeIdx < 1 || codeIdx > numCodes {
				continue
			}
			isBooking := (ov & 0x8000) != 0
			w1 := int(sv >> 8)
			d1 := int((sv >> 5) & 0x07)
			h1 := int(sv & 0x1F)
			w2 := int(ev >> 8)
			d2 := int((ev >> 5) & 0x07)
			h2 := int(ev & 0x1F)

			if w1 < 1 {
				w1 = 1
			}
			if w2 < w1 {
				w2 = w1
			}
			if w2 > maxWeeks {
				w2 = maxWeeks
			}

			if isBooking || !metas[codeIdx].isCourse {
				// Booking on resource or course
				tType := "resource"
				if metas[codeIdx].isCourse {
					tType = "course"
				}
				if d1 == 0 {
					for w := w1; w <= w2; w++ {
						bookingOrder++
						p.Bookings = append(p.Bookings, Booking{
							ID: fmt.Sprintf("b_%d", bookingOrder), TargetType: tType, TargetID: metas[codeIdx].id,
							Scope: "week", Week: w, Kind: 1,
						})
					}
				} else {
					for w := w1; w <= w2; w++ {
						for d := d1; d <= d2 && d <= maxDays; d++ {
							for h := h1; h <= h2 && h <= maxPeriods; h++ {
								bookingOrder++
								p.Bookings = append(p.Bookings, Booking{
									ID: fmt.Sprintf("b_%d", bookingOrder), TargetType: tType, TargetID: metas[codeIdx].id,
									Scope: "timetable", Week: w, Day: d, Period: h, Kind: 1,
								})
							}
						}
					}
				}
				continue
			}

			// Course week allocation or timetable slot
			cid := metas[codeIdx].id
			if d1 == 0 {
				// Week allocation: low byte of ev is weekly periods
				per := int(ev & 0xFF)
				if per > 0 {
					for w := w1; w <= w2; w++ {
						p.SetWeekAlloc(cid, w, p.WeekAlloc(cid, w)+per)
					}
				}
			} else {
				for w := w1; w <= w2; w++ {
					for d := d1; d <= d2 && d <= maxDays; d++ {
						for h := h1; h <= h2 && h <= maxPeriods; h++ {
							slotOrder++
							p.Slots = append(p.Slots, SlotAssignment{
								ID: fmt.Sprintf("s_%d", slotOrder), CourseID: cid, Week: w, Day: d, Period: h,
							})
						}
					}
				}
			}
		}
	}

	// Parse tail string table for full Course/Resource names & Institution/Department
	// Each entry: uint16 id, uint16 lenPlusOne, uint8 len, bytes[len]
	for off := pos; off+5 < len(data); off++ {
		id := binary.LittleEndian.Uint16(data[off : off+2])
		sz := int(binary.LittleEndian.Uint16(data[off+2 : off+4]))
		slen := int(data[off+4])
		if sz == slen+1 && slen >= 1 && slen <= 120 && off+5+slen <= len(data) {
			txt := decodeLatin1(data[off+5 : off+5+slen])
			// Verify printable
			printable := true
			for _, r := range txt {
				if r < 32 {
					printable = false
					break
				}
			}
			if !printable {
				continue
			}
			if (id & 0x4000) != 0 {
				cIdx := int(id & 0x1FFF)
				if cIdx >= 1 && cIdx <= numCodes {
					if metas[cIdx].isCourse {
						if c := p.CourseByID(metas[cIdx].id); c != nil {
							c.Name = txt
						}
					} else {
						if r := p.ResourceByID(metas[cIdx].id); r != nil {
							r.Name = txt
						}
					}
				}
				off += 4 + slen
			} else if id == 0x80 || id == 0x0080 {
				p.Institution = txt
				off += 4 + slen
			} else if id == 0x81 || id == 0x0081 {
				p.Department = txt
				off += 4 + slen
			}
		}
	}

	p.RecomputeDerived()
	return p, nil
}
