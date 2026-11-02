// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package time

import (
	"fmt"
	stdtime "time"

	"github.com/synnaxlabs/x/telem"
)

// civilTime holds the UTC calendar and clock fields of a timestamp.
type civilTime struct {
	year, month, day, hour, minute, second, nanosecond int
	// yearDay counts from 1 on January 1.
	yearDay int
	// weekday counts from 0 on Sunday.
	weekday          int
	isoYear, isoWeek int
}

func breakDown(ts telem.TimeStamp) civilTime {
	t := ts.Time().UTC()
	isoYear, isoWeek := t.ISOWeek()
	return civilTime{
		year:       t.Year(),
		month:      int(t.Month()),
		day:        t.Day(),
		hour:       t.Hour(),
		minute:     t.Minute(),
		second:     t.Second(),
		nanosecond: t.Nanosecond(),
		yearDay:    t.YearDay(),
		weekday:    int(t.Weekday()),
		isoYear:    isoYear,
		isoWeek:    isoWeek,
	}
}

// expansion appends one format code's text for c to b.
type expansion func(b []byte, c civilTime) []byte

func pad(b []byte, v, width int) []byte { return fmt.Appendf(b, "%0*d", width, v) }

func year(b []byte, c civilTime) []byte   { return pad(b, c.year, 4) }
func month(b []byte, c civilTime) []byte  { return pad(b, c.month, 2) }
func day(b []byte, c civilTime) []byte    { return pad(b, c.day, 2) }
func hour(b []byte, c civilTime) []byte   { return pad(b, c.hour, 2) }
func minute(b []byte, c civilTime) []byte { return pad(b, c.minute, 2) }
func second(b []byte, c civilTime) []byte { return pad(b, c.second, 2) }

func spaceDay(b []byte, c civilTime) []byte { return fmt.Appendf(b, "%2d", c.day) }

func hour12(b []byte, c civilTime) []byte { return pad(b, (c.hour+11)%12+1, 2) }

func weekdayName(b []byte, c civilTime) []byte {
	return append(b, stdtime.Weekday(c.weekday).String()...)
}

func shortWeekday(b []byte, c civilTime) []byte {
	return append(b, stdtime.Weekday(c.weekday).String()[:3]...)
}

func monthName(b []byte, c civilTime) []byte {
	return append(b, stdtime.Month(c.month).String()...)
}

func shortMonth(b []byte, c civilTime) []byte {
	return append(b, stdtime.Month(c.month).String()[:3]...)
}

func meridiem(b []byte, c civilTime) []byte {
	if c.hour < 12 {
		return append(b, "AM"...)
	}
	return append(b, "PM"...)
}

func clock(b []byte, c civilTime) []byte {
	b = hour(b, c)
	b = append(b, ':')
	b = minute(b, c)
	b = append(b, ':')
	return second(b, c)
}

func slashDate(b []byte, c civilTime) []byte {
	b = month(b, c)
	b = append(b, '/')
	b = day(b, c)
	b = append(b, '/')
	return pad(b, c.year%100, 2)
}

// codes maps each supported format code to its expansion.
var codes = map[byte]expansion{
	'Y': year,
	'y': func(b []byte, c civilTime) []byte { return pad(b, c.year%100, 2) },
	'C': func(b []byte, c civilTime) []byte { return pad(b, c.year/100, 2) },
	'm': month,
	'd': day,
	'e': spaceDay,
	'H': hour,
	'I': hour12,
	'M': minute,
	'S': second,
	'j': func(b []byte, c civilTime) []byte { return pad(b, c.yearDay, 3) },
	'f': func(b []byte, c civilTime) []byte { return pad(b, c.nanosecond/1000, 6) },
	'N': func(b []byte, c civilTime) []byte { return pad(b, c.nanosecond, 9) },
	'u': func(b []byte, c civilTime) []byte { return pad(b, (c.weekday+6)%7+1, 1) },
	'w': func(b []byte, c civilTime) []byte { return pad(b, c.weekday, 1) },
	// Days before the first Sunday of the year fall in week 0.
	'U': func(b []byte, c civilTime) []byte {
		return pad(b, (c.yearDay-1-c.weekday+7)/7, 2)
	},
	// Days before the first Monday of the year fall in week 0.
	'W': func(b []byte, c civilTime) []byte {
		return pad(b, (c.yearDay-1-(c.weekday+6)%7+7)/7, 2)
	},
	'G': func(b []byte, c civilTime) []byte { return pad(b, c.isoYear, 4) },
	'g': func(b []byte, c civilTime) []byte { return pad(b, c.isoYear%100, 2) },
	'V': func(b []byte, c civilTime) []byte { return pad(b, c.isoWeek, 2) },
	'z': func(b []byte, _ civilTime) []byte { return append(b, "+0000"...) },
	'Z': func(b []byte, _ civilTime) []byte { return append(b, "UTC"...) },
	'F': func(b []byte, c civilTime) []byte {
		b = year(b, c)
		b = append(b, '-')
		b = month(b, c)
		b = append(b, '-')
		return day(b, c)
	},
	'T': clock,
	'D': slashDate,
	'R': func(b []byte, c civilTime) []byte {
		b = hour(b, c)
		b = append(b, ':')
		return minute(b, c)
	},
	'a': shortWeekday,
	'A': weekdayName,
	'b': shortMonth,
	'h': shortMonth,
	'B': monthName,
	'p': meridiem,
	'c': func(b []byte, c civilTime) []byte {
		b = shortWeekday(b, c)
		b = append(b, ' ')
		b = shortMonth(b, c)
		b = append(b, ' ')
		b = spaceDay(b, c)
		b = append(b, ' ')
		b = clock(b, c)
		b = append(b, ' ')
		return year(b, c)
	},
	'x': slashDate,
	'X': clock,
	'r': func(b []byte, c civilTime) []byte {
		b = hour12(b, c)
		b = append(b, ':')
		b = minute(b, c)
		b = append(b, ':')
		b = second(b, c)
		b = append(b, ' ')
		return meridiem(b, c)
	},
	'n': func(b []byte, _ civilTime) []byte { return append(b, '\n') },
	't': func(b []byte, _ civilTime) []byte { return append(b, '\t') },
	'%': func(b []byte, _ civilTime) []byte { return append(b, '%') },
}

// strftime renders ts in UTC with format. An unsupported code, and a '%' at the end
// of format, are copied through unchanged.
func strftime(ts telem.TimeStamp, format string) string {
	c := breakDown(ts)
	b := make([]byte, 0, 2*len(format))
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 == len(format) {
			b = append(b, format[i])
			continue
		}
		i++
		if expand, ok := codes[format[i]]; ok {
			b = expand(b, c)
		} else {
			b = append(b, '%', format[i])
		}
	}
	return string(b)
}
