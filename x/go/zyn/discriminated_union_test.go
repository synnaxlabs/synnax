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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("DiscriminatedUnion", func() {
	var schema zyn.DiscriminatedUnionZ
	BeforeEach(func() {
		schema = zyn.DiscriminatedUnion("Type",
			zyn.Object(map[string]zyn.Schema{
				"Type":       zyn.Literal("read"),
				"SampleRate": zyn.Number(),
			}),
			zyn.Object(map[string]zyn.Schema{
				"Type":     zyn.Literal("write"),
				"Endpoint": zyn.String(),
			}),
		)
	})
	Describe("Basic Parsing", func() {
		Specify("first variant", func() {
			type ReadConfig struct {
				Type       string
				SampleRate float64
			}
			var dest ReadConfig
			Expect(schema.Parse(map[string]any{"Type": "read", "SampleRate": 1000.0}, &dest)).To(Succeed())
			Expect(dest.Type).To(Equal("read"))
			Expect(dest.SampleRate).To(Equal(1000.0))
		})
		Specify("second variant", func() {
			type WriteConfig struct {
				Type     string
				Endpoint string
			}
			var dest WriteConfig
			Expect(schema.Parse(map[string]any{"Type": "write", "Endpoint": "opc.tcp://localhost"}, &dest)).To(Succeed())
			Expect(dest.Type).To(Equal("write"))
			Expect(dest.Endpoint).To(Equal("opc.tcp://localhost"))
		})
		Specify("snake_case input data", func() {
			type ReadConfig struct {
				Type       string
				SampleRate float64
			}
			var dest ReadConfig
			Expect(schema.Parse(map[string]any{"type": "read", "sample_rate": 1000.0}, &dest)).To(Succeed())
			Expect(dest.Type).To(Equal("read"))
			Expect(dest.SampleRate).To(Equal(1000.0))
		})
	})
	Describe("Validate", func() {
		It("Should succeed for a valid variant", func() {
			Expect(schema.Validate(map[string]any{"Type": "read", "SampleRate": 42.0})).To(Succeed())
		})
		It("Should fail for an unknown discriminator value", func() {
			Expect(schema.Validate(map[string]any{"Type": "delete"})).
				To(MatchError(ContainSubstring("unknown discriminator value")))
		})
		It("Should fail for non-map data", func() {
			Expect(schema.Validate("not a map")).To(HaveOccurred())
		})
		It("Should succeed for optional nil", func() {
			Expect(schema.Optional().Validate(nil)).To(Succeed())
		})
		It("Should fail for required nil", func() {
			Expect(schema.Validate(nil)).To(MatchError(validate.ErrRequired))
		})
		It("Should fail when discriminator field is missing", func() {
			Expect(schema.Validate(map[string]any{"SampleRate": 42.0})).
				To(MatchError(ContainSubstring("discriminator field")))
		})
	})
	Describe("Invalid Inputs", func() {
		Specify("discriminator field missing from data", func() {
			type Cfg struct{ SampleRate float64 }
			var dest Cfg
			Expect(schema.Parse(map[string]any{"SampleRate": 1000.0}, &dest)).
				To(MatchError(ContainSubstring("discriminator field")))
		})
		Specify("unknown discriminator value", func() {
			type Cfg struct{ Type string }
			var dest Cfg
			err := schema.Parse(map[string]any{"Type": "delete"}, &dest)
			Expect(err).To(MatchError(ContainSubstring("unknown discriminator value")))
			Expect(err).To(MatchError(ContainSubstring("read")))
			Expect(err).To(MatchError(ContainSubstring("write")))
		})
		Specify("non-map data", func() {
			type Cfg struct{ Type string }
			var dest Cfg
			Expect(schema.Parse("not a map", &dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("non-pointer destination", func() {
			type Cfg struct{ Type string }
			var dest Cfg
			Expect(schema.Parse(map[string]any{"Type": "read"}, dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("variant field validation error", func() {
			type Cfg struct {
				Type       string
				SampleRate float64
			}
			var dest Cfg
			Expect(schema.Parse(map[string]any{"Type": "read", "SampleRate": "bad"}, &dest)).
				To(HaveOccurred())
		})
	})
	Describe("Optional Fields", func() {
		Specify("optional with nil value", func() {
			type Cfg struct{ Type string }
			var dest Cfg
			Expect(schema.Optional().Parse(nil, &dest)).To(Succeed())
		})
		Specify("required with nil value", func() {
			type Cfg struct{ Type string }
			var dest Cfg
			Expect(schema.Parse(nil, &dest)).To(MatchError(validate.ErrRequired))
		})
	})
	Describe("Dump", func() {
		Specify("struct first variant", func() {
			type ReadConfig struct {
				Type       string
				SampleRate float64
			}
			result := MustSucceed(schema.Dump(ReadConfig{Type: "read", SampleRate: 1000}))
			Expect(result).To(Equal(map[string]any{
				"type":        "read",
				"sample_rate": 1000.0,
			}))
		})
		Specify("struct second variant", func() {
			type WriteConfig struct {
				Type     string
				Endpoint string
			}
			result := MustSucceed(schema.Dump(WriteConfig{Type: "write", Endpoint: "opc.tcp://localhost"}))
			Expect(result).To(Equal(map[string]any{
				"type":     "write",
				"endpoint": "opc.tcp://localhost",
			}))
		})
		Specify("map input", func() {
			result := MustSucceed(schema.Dump(map[string]any{"type": "read", "sample_rate": 1000.0}))
			Expect(result).To(Equal(map[string]any{
				"type":        "read",
				"sample_rate": 1000.0,
			}))
		})
		Specify("nil required", func() {
			Expect(schema.Dump(nil)).Error().To(MatchError(validate.ErrRequired))
		})
		Specify("nil pointer", func() {
			type Cfg struct{ Type string }
			var c *Cfg
			Expect(schema.Dump(c)).Error().To(MatchError(validate.ErrRequired))
		})
		Specify("optional nil value", func() {
			result := MustSucceed(schema.Optional().Dump(nil))
			Expect(result).To(BeNil())
		})
		Specify("optional nil pointer", func() {
			type Cfg struct{ Type string }
			var c *Cfg
			result := MustSucceed(schema.Optional().Dump(c))
			Expect(result).To(BeNil())
		})
		Specify("non-struct non-map value", func() {
			Expect(schema.Dump("not a struct")).Error().
				To(MatchError(ContainSubstring("expected struct or map[string]any")))
		})
		Specify("unknown discriminator in struct", func() {
			type BadConfig struct{ Type string }
			Expect(schema.Dump(BadConfig{Type: "delete"})).Error().
				To(MatchError(ContainSubstring("unknown discriminator value")))
		})
		Specify("non-nil pointer to struct is dereferenced", func() {
			type ReadConfig struct {
				Type       string
				SampleRate float64
			}
			cfg := &ReadConfig{Type: "read", SampleRate: 1000}
			result := MustSucceed(schema.Dump(cfg))
			Expect(result).To(Equal(map[string]any{
				"type":        "read",
				"sample_rate": 1000.0,
			}))
		})
		Specify("map input with missing discriminator", func() {
			Expect(schema.Dump(map[string]any{"SampleRate": 1000.0})).Error().
				To(MatchError(ContainSubstring("discriminator field")))
		})
		Specify("map input with unknown discriminator value", func() {
			Expect(schema.Dump(map[string]any{"Type": "delete"})).Error().
				To(MatchError(ContainSubstring("unknown discriminator value")))
		})
		Specify("struct missing discriminator field", func() {
			type NoDisc struct{ SampleRate float64 }
			Expect(schema.Dump(NoDisc{SampleRate: 100})).Error().
				To(MatchError(ContainSubstring("discriminator field")))
		})
		Specify("round-trip parse then dump", func() {
			type ReadConfig struct {
				Type       string
				SampleRate float64
			}
			var dest ReadConfig
			Expect(schema.Parse(map[string]any{"type": "read", "sample_rate": 1000.0}, &dest)).To(Succeed())
			result := MustSucceed(schema.Dump(dest))
			Expect(result).To(Equal(map[string]any{
				"type":        "read",
				"sample_rate": 1000.0,
			}))
		})
	})
	Describe("Discriminator Case Forms", func() {
		It("Should match discriminator via snake_case data key", func() {
			s := zyn.DiscriminatedUnion("Type",
				zyn.Object(map[string]zyn.Schema{
					"Type": zyn.Literal("a"),
					"X":    zyn.Number(),
				}),
				zyn.Object(map[string]zyn.Schema{
					"Type": zyn.Literal("b"),
					"Y":    zyn.String(),
				}),
			)
			Expect(s.Validate(map[string]any{"type": "a", "X": 1.0})).To(Succeed())
		})
		It("Should match discriminator via raw data key when neither pascal nor snake match", func() {
			s := zyn.DiscriminatedUnion("TYPE",
				zyn.Object(map[string]zyn.Schema{
					"TYPE": zyn.Literal("a"),
				}),
				zyn.Object(map[string]zyn.Schema{
					"TYPE": zyn.Literal("b"),
				}),
			)
			Expect(s.Validate(map[string]any{"TYPE": "a"})).To(Succeed())
		})
		It("Should resolve findFieldSchema via snake_case fallback", func() {
			Expect(func() {
				zyn.DiscriminatedUnion("type",
					zyn.Object(map[string]zyn.Schema{
						"Type": zyn.Literal("a"),
					}),
					zyn.Object(map[string]zyn.Schema{
						"Type": zyn.Literal("b"),
					}),
				)
			}).NotTo(Panic())
		})
		It("Should resolve findFieldSchema via PascalCase fallback", func() {
			Expect(func() {
				zyn.DiscriminatedUnion("Type",
					zyn.Object(map[string]zyn.Schema{
						"type": zyn.Literal("a"),
					}),
					zyn.Object(map[string]zyn.Schema{
						"type": zyn.Literal("b"),
					}),
				)
			}).NotTo(Panic())
		})
	})
	Describe("Snake Case Schema Definition", func() {
		It("Should work with snake_case field names in schema", func() {
			snakeSchema := zyn.DiscriminatedUnion("task_type",
				zyn.Object(map[string]zyn.Schema{
					"task_type": zyn.Literal("read"),
					"rate":      zyn.Number(),
				}),
				zyn.Object(map[string]zyn.Schema{
					"task_type": zyn.Literal("write"),
					"target":    zyn.String(),
				}),
			)
			type ReadTask struct {
				TaskType string
				Rate     float64
			}
			var dest ReadTask
			Expect(snakeSchema.Parse(map[string]any{"task_type": "read", "rate": 500.0}, &dest)).To(Succeed())
			Expect(dest.TaskType).To(Equal("read"))
			Expect(dest.Rate).To(Equal(500.0))
		})
	})
	Describe("Nested Objects", func() {
		It("Should work with nested object variants", func() {
			nestedSchema := zyn.DiscriminatedUnion("Kind",
				zyn.Object(map[string]zyn.Schema{
					"Kind": zyn.Literal("simple"),
					"Config": zyn.Object(map[string]zyn.Schema{
						"Value": zyn.String(),
					}),
				}),
				zyn.Object(map[string]zyn.Schema{
					"Kind": zyn.Literal("complex"),
					"Config": zyn.Object(map[string]zyn.Schema{
						"Values": zyn.Array(zyn.String()),
					}),
				}),
			)
			type SimpleConfig struct{ Value string }
			type SimpleVariant struct {
				Kind   string
				Config SimpleConfig
			}
			var dest SimpleVariant
			Expect(nestedSchema.Parse(map[string]any{
				"Kind":   "simple",
				"Config": map[string]any{"Value": "hello"},
			}, &dest)).To(Succeed())
			Expect(dest.Kind).To(Equal("simple"))
			Expect(dest.Config.Value).To(Equal("hello"))
		})
	})
	Describe("Shape", func() {
		It("Should return a DiscriminatedUnionShape", func() {
			shape := schema.Shape()
			Expect(shape.DataType()).To(Equal(zyn.DiscriminatedUnionT))
			Expect(shape.Optional()).To(BeFalse())
			Expect(shape.Fields()).To(BeNil())
			duShape, ok := shape.(zyn.DiscriminatedUnionShape)
			Expect(ok).To(BeTrue())
			Expect(duShape.Discriminator()).To(Equal("Type"))
			Expect(duShape.Variants()).To(HaveLen(2))
			Expect(duShape.Variants()).To(HaveKey("read"))
			Expect(duShape.Variants()).To(HaveKey("write"))
		})
		It("Should reflect optional", func() {
			Expect(schema.Optional().Shape().Optional()).To(BeTrue())
		})
	})
	Describe("Constructor Panics", func() {
		It("Should panic with fewer than 2 schemas", func() {
			Expect(func() {
				zyn.DiscriminatedUnion("type",
					zyn.Object(map[string]zyn.Schema{"type": zyn.Literal("a")}),
				)
			}).To(Panic())
		})
		It("Should panic when discriminator field is missing", func() {
			Expect(func() {
				zyn.DiscriminatedUnion("type",
					zyn.Object(map[string]zyn.Schema{"kind": zyn.Literal("a")}),
					zyn.Object(map[string]zyn.Schema{"kind": zyn.Literal("b")}),
				)
			}).To(Panic())
		})
		It("Should panic when discriminator field is not a Literal", func() {
			Expect(func() {
				zyn.DiscriminatedUnion("type",
					zyn.Object(map[string]zyn.Schema{"type": zyn.String()}),
					zyn.Object(map[string]zyn.Schema{"type": zyn.Literal("b")}),
				)
			}).To(Panic())
		})
		It("Should panic when discriminator field is an Enum with multiple values", func() {
			Expect(func() {
				zyn.DiscriminatedUnion("type",
					zyn.Object(map[string]zyn.Schema{"type": zyn.Enum("a", "b")}),
					zyn.Object(map[string]zyn.Schema{"type": zyn.Literal("c")}),
				)
			}).To(Panic())
		})
		It("Should panic on duplicate discriminator values", func() {
			Expect(func() {
				zyn.DiscriminatedUnion("type",
					zyn.Object(map[string]zyn.Schema{"type": zyn.Literal("a")}),
					zyn.Object(map[string]zyn.Schema{"type": zyn.Literal("a")}),
				)
			}).To(Panic())
		})
	})
})
