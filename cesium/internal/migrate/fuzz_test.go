// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate_test

import (
	jsonv2 "encoding/json/v2"
	"reflect"
	"testing"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/migrate"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/telem"
)

// FuzzMigrate checks that Migrate takes any channel that a meta file decodes to, and
// that a second Migrate changes nothing.
func FuzzMigrate(f *testing.F) {
	codec := json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true))
	for _, ch := range []channel.Channel{
		{Key: 1, Name: "clock", DataType: telem.TimestampT, IsIndex: true},
		{Key: 2, DataType: telem.Float64T},
		{Key: 3, DataType: telem.StringT, Virtual: true, IsIndex: true, Version: 2},
	} {
		data, err := codec.Encode(ch)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		var ch channel.Channel
		if codec.Decode(data, &ch) != nil {
			return
		}
		once := migrate.Migrate(migrate.DBState{Channel: ch})
		if twice := migrate.Migrate(once); !reflect.DeepEqual(twice, once) {
			t.Fatalf("a second migrate gave %+v, want %+v", twice, once)
		}
	})
}
