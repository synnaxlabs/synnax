// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/synnaxlabs/x/zyn"
)

// --- Realistic schemas matching production usage ---

// Channel: 9 fields, the largest real schema
var channelSchema = zyn.Object(map[string]zyn.Schema{
	"key":         zyn.Uint32().Coerce(),
	"name":        zyn.String(),
	"leaseholder": zyn.Uint16().Coerce(),
	"is_index":    zyn.Bool(),
	"index":       zyn.Uint32().Coerce(),
	"data_type":   zyn.String(),
	"internal":    zyn.Bool(),
	"virtual":     zyn.Bool(),
	"expression":  zyn.String(),
})

// Workspace: 2 fields, the most common pattern (UUID + String)
var workspaceSchema = zyn.Object(map[string]zyn.Schema{
	"key":  zyn.UUID(),
	"name": zyn.String(),
})

// User: 5 fields with UUID + Strings + Bool
var userSchema = zyn.Object(map[string]zyn.Schema{
	"key":        zyn.UUID(),
	"username":   zyn.String(),
	"first_name": zyn.String(),
	"last_name":  zyn.String(),
	"root_user":  zyn.Bool(),
})

// Device: 7 fields, mix of String + Uint32 + Bool
var deviceSchema = zyn.Object(map[string]zyn.Schema{
	"key":        zyn.String(),
	"rack":       zyn.Uint32().Coerce(),
	"location":   zyn.String(),
	"name":       zyn.String(),
	"make":       zyn.String(),
	"model":      zyn.String(),
	"configured": zyn.Bool(),
})

// Range: 4 fields with nested color + time_range objects (depth 2)
var colorSchema = zyn.Object(map[string]zyn.Schema{
	"r": zyn.Number().Uint8().Coerce(),
	"g": zyn.Number().Uint8().Coerce(),
	"b": zyn.Number().Uint8().Coerce(),
	"a": zyn.Number().Float64().Coerce(),
})

var timeRangeSchema = zyn.Object(map[string]zyn.Schema{
	"start": zyn.Int64().Coerce(),
	"end":   zyn.Int64().Coerce(),
})

var rangeSchema = zyn.Object(map[string]zyn.Schema{
	"key":        zyn.UUID(),
	"name":       zyn.String(),
	"color":      colorSchema,
	"time_range": timeRangeSchema,
})

// --- Realistic test data ---

func channelPayload() map[string]any {
	return map[string]any{
		"key":         uint32(42),
		"name":        "temperature_sensor_01",
		"leaseholder": uint16(1),
		"is_index":    false,
		"index":       uint32(41),
		"data_type":   "float32",
		"internal":    false,
		"virtual":     false,
		"expression":  "",
	}
}

func workspacePayload() map[string]any {
	return map[string]any{
		"key":  uuid.New().String(),
		"name": "Telemetry Dashboard",
	}
}

func userPayload() map[string]any {
	return map[string]any{
		"key":        uuid.New().String(),
		"username":   "jdoe",
		"first_name": "John",
		"last_name":  "Doe",
		"root_user":  false,
	}
}

func devicePayload() map[string]any {
	return map[string]any{
		"key":        "LJM-470031234",
		"rack":       uint32(1),
		"location":   "Lab A",
		"name":       "LabJack T7",
		"make":       "LabJack",
		"model":      "T7-Pro",
		"configured": true,
	}
}

func rangePayload() map[string]any {
	return map[string]any{
		"key":  uuid.New().String(),
		"name": "Test Run 42",
		"color": map[string]any{
			"r": uint8(255),
			"g": uint8(0),
			"b": uint8(128),
			"a": 1.0,
		},
		"time_range": map[string]any{
			"start": int64(1700000000000000000),
			"end":   int64(1700000060000000000),
		},
	}
}

// --- Primitive benchmarks ---

func BenchmarkStringParse(b *testing.B) {
	schema := zyn.String()
	var dest string
	b.ReportAllocs()
	for b.Loop() {
		_ = schema.Parse("temperature_sensor_01", &dest)
	}
}

func BenchmarkStringDump(b *testing.B) {
	schema := zyn.String()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = schema.Dump("temperature_sensor_01")
	}
}

func BenchmarkUUIDParse(b *testing.B) {
	schema := zyn.UUID()
	id := uuid.New().String()
	var dest uuid.UUID
	b.ReportAllocs()
	for b.Loop() {
		_ = schema.Parse(id, &dest)
	}
}

func BenchmarkUUIDDump(b *testing.B) {
	schema := zyn.UUID()
	id := uuid.New()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = schema.Dump(id)
	}
}

func BenchmarkNumberParse(b *testing.B) {
	b.Run("float64", func(b *testing.B) {
		schema := zyn.Number()
		var dest float64
		b.ReportAllocs()
		for b.Loop() {
			_ = schema.Parse(42.5, &dest)
		}
	})
	b.Run("uint32_coerce", func(b *testing.B) {
		schema := zyn.Uint32().Coerce()
		var dest uint32
		b.ReportAllocs()
		for b.Loop() {
			_ = schema.Parse(uint32(42), &dest)
		}
	})
	b.Run("int64_coerce", func(b *testing.B) {
		schema := zyn.Int64().Coerce()
		var dest int64
		b.ReportAllocs()
		for b.Loop() {
			_ = schema.Parse(int64(1700000000), &dest)
		}
	})
}

func BenchmarkNumberDump(b *testing.B) {
	b.Run("uint32", func(b *testing.B) {
		schema := zyn.Uint32().Coerce()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = schema.Dump(uint32(42))
		}
	})
	b.Run("int64", func(b *testing.B) {
		schema := zyn.Int64().Coerce()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = schema.Dump(int64(1700000000))
		}
	})
}

func BenchmarkBoolParse(b *testing.B) {
	schema := zyn.Bool()
	var dest bool
	b.ReportAllocs()
	for b.Loop() {
		_ = schema.Parse(true, &dest)
	}
}

func BenchmarkBoolDump(b *testing.B) {
	schema := zyn.Bool()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = schema.Dump(true)
	}
}

// --- Object benchmarks (real schemas) ---

func BenchmarkObjectDump(b *testing.B) {
	b.Run("workspace_2f", func(b *testing.B) {
		data := workspacePayload()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = workspaceSchema.Dump(data)
		}
	})
	b.Run("user_5f", func(b *testing.B) {
		data := userPayload()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = userSchema.Dump(data)
		}
	})
	b.Run("device_7f", func(b *testing.B) {
		data := devicePayload()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = deviceSchema.Dump(data)
		}
	})
	b.Run("channel_9f", func(b *testing.B) {
		data := channelPayload()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = channelSchema.Dump(data)
		}
	})
	b.Run("range_4f_nested", func(b *testing.B) {
		data := rangePayload()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = rangeSchema.Dump(data)
		}
	})
}

func BenchmarkObjectParse(b *testing.B) {
	type Workspace struct {
		Key  uuid.UUID
		Name string
	}
	type User struct {
		Key       uuid.UUID
		Username  string
		FirstName string
		LastName  string
		RootUser  bool
	}
	type Channel struct {
		Key         uint32
		Name        string
		Leaseholder uint16
		IsIndex     bool
		Index       uint32
		DataType    string
		Internal    bool
		Virtual     bool
		Expression  string
	}
	b.Run("workspace_2f", func(b *testing.B) {
		data := workspacePayload()
		var dest Workspace
		b.ReportAllocs()
		for b.Loop() {
			_ = workspaceSchema.Parse(data, &dest)
		}
	})
	b.Run("user_5f", func(b *testing.B) {
		data := userPayload()
		var dest User
		b.ReportAllocs()
		for b.Loop() {
			_ = userSchema.Parse(data, &dest)
		}
	})
	b.Run("channel_9f", func(b *testing.B) {
		data := channelPayload()
		var dest Channel
		b.ReportAllocs()
		for b.Loop() {
			_ = channelSchema.Parse(data, &dest)
		}
	})
}

func BenchmarkObjectValidate(b *testing.B) {
	b.Run("workspace_2f", func(b *testing.B) {
		data := workspacePayload()
		b.ReportAllocs()
		for b.Loop() {
			_ = workspaceSchema.Validate(data)
		}
	})
	b.Run("channel_9f", func(b *testing.B) {
		data := channelPayload()
		b.ReportAllocs()
		for b.Loop() {
			_ = channelSchema.Validate(data)
		}
	})
}

// --- Array benchmarks (realistic sizes) ---

func BenchmarkArrayParse(b *testing.B) {
	schema := zyn.Array(zyn.String())
	for _, size := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("strings_%d", size), func(b *testing.B) {
			data := make([]any, size)
			for i := range data {
				data[i] = fmt.Sprintf("item-%d", i)
			}
			dest := make([]string, 0)
			b.ReportAllocs()
			for b.Loop() {
				_ = schema.Parse(data, &dest)
			}
		})
	}
}

func BenchmarkArrayOfChannelsDump(b *testing.B) {
	schema := zyn.Array(channelSchema)
	for _, size := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			data := make([]any, size)
			for i := range data {
				data[i] = channelPayload()
			}
			b.ReportAllocs()
			for b.Loop() {
				_, _ = schema.Dump(data)
			}
		})
	}
}

// --- Map benchmarks ---

func BenchmarkMapParse(b *testing.B) {
	schema := zyn.Map(zyn.String(), zyn.String())
	for _, size := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			data := make(map[string]any, size)
			for i := range size {
				data[strconv.Itoa(i)] = fmt.Sprintf("value-%d", i)
			}
			var dest map[string]string
			b.ReportAllocs()
			for b.Loop() {
				_ = schema.Parse(data, &dest)
			}
		})
	}
}

func BenchmarkMapDump(b *testing.B) {
	schema := zyn.Map(zyn.String(), zyn.String())
	for _, size := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			data := make(map[string]string, size)
			for i := range size {
				data[strconv.Itoa(i)] = fmt.Sprintf("value-%d", i)
			}
			b.ReportAllocs()
			for b.Loop() {
				_, _ = schema.Dump(data)
			}
		})
	}
}

// --- Union vs DiscriminatedUnion ---

func BenchmarkUnionVsDiscriminated(b *testing.B) {
	readSchema := zyn.Object(map[string]zyn.Schema{
		"type":        zyn.Literal("read"),
		"sample_rate": zyn.Number(),
	})
	writeSchema := zyn.Object(map[string]zyn.Schema{
		"type":     zyn.Literal("write"),
		"endpoint": zyn.String(),
	})
	union := zyn.Union(readSchema, writeSchema)
	discriminated := zyn.DiscriminatedUnion("type", readSchema, writeSchema)
	data := map[string]any{"type": "write", "endpoint": "opc.tcp://localhost"}

	b.Run("union_validate", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = union.Validate(data)
		}
	})
	b.Run("discriminated_validate", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = discriminated.Validate(data)
		}
	})
}

// --- End-to-end: NewResource simulation (Dump from map) ---

func BenchmarkNewResourceSimulation(b *testing.B) {
	b.Run("channel", func(b *testing.B) {
		data := channelPayload()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = channelSchema.Dump(data)
		}
	})
	b.Run("range_nested", func(b *testing.B) {
		data := rangePayload()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = rangeSchema.Dump(data)
		}
	})
}
