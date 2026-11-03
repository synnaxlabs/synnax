// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package tz_test

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/tz"
)

// winter is before daylight time starts, and summer is in it.
var (
	winter = time.Date(2026, time.March, 7, 15, 4, 5, 0, time.UTC)
	summer = time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
)

func zoneAt(loc *time.Location, t time.Time) (string, int) { return t.In(loc).Zone() }

// ruleZone returns the TZif data gen.go builds for testdata/rules.tsv.
func ruleZone(rule string) []byte {
	var b bytes.Buffer
	header := func(types, chars uint32) {
		b.WriteString("TZif2")
		b.Write(make([]byte, 15))
		for _, n := range []uint32{0, 0, 0, 0, types, chars} {
			Expect(binary.Write(&b, binary.BigEndian, n)).To(Succeed())
		}
	}
	header(0, 0)
	header(1, 4)
	Expect(binary.Write(&b, binary.BigEndian, int32(3600))).To(Succeed())
	b.Write([]byte{0, 0})
	b.WriteString("FAL\x00")
	b.WriteString("\n" + rule + "\n")
	return b.Bytes()
}

// eachVector calls check with the fields of every data line in the vector file.
func eachVector(path string, check func(fields []string)) int {
	f := MustSucceed(os.Open(path))
	DeferCleanup(f.Close)
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		Expect(fields).To(HaveLen(4), line)
		check(fields)
		count++
	}
	Expect(scanner.Err()).ToNot(HaveOccurred())
	return count
}

var _ = Describe("Database", func() {
	var db *tz.Database
	BeforeEach(func() { db = MustSucceed(tz.Open()) })

	Describe("Load", func() {
		It("Should load a zone by its exact name", func() {
			loc := MustSucceed(db.Load("America/Denver"))
			Expect(loc.String()).To(Equal("America/Denver"))
			name, offset := zoneAt(loc, winter)
			Expect(name).To(Equal("MST"))
			Expect(offset).To(Equal(-7 * 3600))
			name, offset = zoneAt(loc, summer)
			Expect(name).To(Equal("MDT"))
			Expect(offset).To(Equal(-6 * 3600))
		})

		DescribeTable("Should match a name without regard to case",
			func(name string) {
				Expect(MustSucceed(db.Load(name)).String()).To(Equal("America/Denver"))
			},
			Entry("lower case", "america/denver"),
			Entry("upper case", "AMERICA/DENVER"),
			Entry("mixed case", "aMeRiCa/DeNvEr"),
		)

		It("Should load a short alias with the same rules as its zone", func() {
			alias := MustSucceed(db.Load("us/mountain"))
			Expect(alias.String()).To(Equal("US/Mountain"))
			denver := MustSucceed(db.Load("America/Denver"))
			for _, t := range []time.Time{winter, summer} {
				aliasName, aliasOffset := zoneAt(alias, t)
				denverName, denverOffset := zoneAt(denver, t)
				Expect(aliasName).To(Equal(denverName))
				Expect(aliasOffset).To(Equal(denverOffset))
			}
		})

		It("Should load a fixed zone that has no daylight time", func() {
			loc := MustSucceed(db.Load("EST"))
			name, offset := zoneAt(loc, summer)
			Expect(name).To(Equal("EST"))
			Expect(offset).To(Equal(-5 * 3600))
		})

		It("Should return the same location for a name loaded twice", func() {
			Expect(db.Load("Europe/Berlin")).
				To(BeIdenticalTo(MustSucceed(db.Load("europe/berlin"))))
		})

		DescribeTable("Should reject a name that is not a zone",
			func(name string) {
				Expect(db.Load(name)).Error().To(SatisfyAll(
					MatchError(query.ErrNotFound),
					MatchError(ContainSubstring(strconv.Quote(name))),
				))
			},
			Entry("an unknown zone", "America/Denvr"),
			Entry("an empty name", ""),
			Entry("an abbreviation the database does not define", "PST"),
		)
	})

	Describe("Names", func() {
		It("Should list every zone in sorted order", func() {
			names := db.Names()
			vectorZones := set.New[string]()
			eachVector("testdata/vectors.tsv", func(fields []string) {
				vectorZones.Add(fields[0])
			})
			Expect(names).To(HaveLen(len(vectorZones)))
			Expect(slices.IsSorted(names)).To(BeTrue())
			Expect(names).To(ContainElements(
				"America/Denver",
				"America/Edmonton",
				"US/Mountain",
				"Canada/Mountain",
				"EST",
				"UTC",
			))
		})

		It("Should load every listed zone", func() {
			for _, name := range db.Names() {
				Expect(db.Load(name)).Error().ToNot(HaveOccurred(), name)
			}
		})
	})

	It("Should agree with the zone vectors the C++ reader checks", func() {
		count := eachVector("testdata/vectors.tsv", func(fields []string) {
			sec := MustSucceed(strconv.ParseInt(fields[1], 10, 64))
			name, offset := zoneAt(MustSucceed(db.Load(fields[0])), time.Unix(sec, 0))
			Expect([]string{strconv.Itoa(offset), name}).
				To(Equal(fields[2:]), "%v, run go generate in x/go/tz", fields)
		})
		Expect(count).To(BeNumerically(">", 15_000))
	})

	It("Should agree with the rule string vectors the C++ reader checks", func() {
		count := eachVector("testdata/rules.tsv", func(fields []string) {
			loc := MustSucceed(time.LoadLocationFromTZData("rule", ruleZone(fields[0])))
			sec := MustSucceed(strconv.ParseInt(fields[1], 10, 64))
			name, offset := zoneAt(loc, time.Unix(sec, 0))
			Expect([]string{strconv.Itoa(offset), name}).
				To(Equal(fields[2:]), "%v, run go generate in x/go/tz", fields)
		})
		Expect(count).To(BeNumerically(">", 1_500))
	})
})
