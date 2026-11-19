// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/set"
)

// seriesPalette is the key of the "Series" palette, which holds the series colors.
var seriesPalette = uuid.MustParse("4d2d0872-30f9-4573-92da-2b72db8f8c64")

// shipped is a built-in named color with its light and dark values as 0xRRGGBBAA.
type shipped struct {
	key         Key
	light, dark uint32
}

// series holds the series colors in order. Each value has 3:1 contrast on the canvas of
// its theme. Under each color vision deficiency, the first eight stay at least 0.05
// apart in OKLab.
var series = [...]shipped{
	{uuid.MustParse("3f00d0f0-26ed-418f-8bf4-1d264a0a55f1"), 0xDC136CFF, 0xDC136CFF},
	{uuid.MustParse("c9972ff4-924f-44a0-80d8-c3997017fa17"), 0x099BE9FF, 0x20A4F3FF},
	{uuid.MustParse("bb72471c-0d8a-421f-aead-152479a46c27"), 0x45A73AFF, 0x96F98AFF},
	{uuid.MustParse("719d47f6-2592-46c1-9d19-c4b0bc181897"), 0xD67E00FF, 0xF18F01FF},
	{uuid.MustParse("fc817476-d2ed-41c3-b5ba-b628daeb5125"), 0xFC5B6EFF, 0xFD5C6FFF},
	{uuid.MustParse("c5cfae40-b07b-4aaa-9183-28a7d4d6c04d"), 0x8075FFFF, 0x8075FFFF},
	{uuid.MustParse("c34781b5-8a00-4453-b30d-17b769e99d04"), 0xC17BC2FF, 0xF7AEF8FF},
	{uuid.MustParse("a892a83a-bca4-4144-a80e-9e69a0883860"), 0x791E94FF, 0x8B33A7FF},
	{uuid.MustParse("8e187613-dbe6-427c-868d-575649a77c08"), 0x107550FF, 0x5BB38AFF},
	{uuid.MustParse("6c21cb3b-66dc-4338-954e-10b1803adb1c"), 0x0D9299FF, 0x64D1D8FF},
	{uuid.MustParse("f54d65d9-cd2e-4364-9142-04b050b67b2f"), 0x908203FF, 0xCEC15AFF},
	{uuid.MustParse("916ba674-29ea-4193-9b04-d0ca7fc37cbd"), 0x8059BBFF, 0xBD96FEFF},
	{uuid.MustParse("1af8578c-f914-48ad-8ef5-b4cd02910090"), 0xAC0184FF, 0xF25BC3FF},
	{uuid.MustParse("129ea6b8-5285-41cd-8520-e71faddbf781"), 0xBB6452FF, 0xFFA28EFF},
	{uuid.MustParse("10368988-b260-41a3-898d-3dbd9fba2c58"), 0x657EBCFF, 0xA1BCFFFF},
	{uuid.MustParse("7e10b078-93f1-4369-b1ba-daaa126301a6"), 0x7C5F06FF, 0xB99A51FF},
}

// Series returns the keys of the named colors in the "Series" palette, sorted by
// position and then by key. A color with no position comes last.
func (s *Service) Series(ctx context.Context, snap gorp.Snapshot) ([]Key, error) {
	var named []Named
	if err := s.NewRetrieve().
		Where(MatchGroups(seriesPalette)).
		Entries(&named).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	slices.SortFunc(named, func(a, b Named) int {
		return cmp.Or(comparePositions(a.Position, b.Position), a.Key.Compare(b.Key))
	})
	return lo.Map(named, func(n Named, _ int) Key { return n.Key }), nil
}

func comparePositions(a, b *string) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	}
	return strings.Compare(*a, *b)
}

func isBuiltin(key Key) bool {
	return lo.ContainsBy(series[:], func(s shipped) bool { return s.key == key })
}

// Builtins returns the named colors that the Core ships, with their shipped values. A
// stored built-in can differ from its shipped values, because users can edit it.
func Builtins() []Named {
	named := make([]Named, len(series))
	for i, s := range series {
		dark := xcolor.FromUint32(s.dark)
		named[i] = Named{
			Key:   s.key,
			Name:  fmt.Sprintf("Series %d", i+1),
			Light: xcolor.FromUint32(s.light),
			Dark:  &dark,
			Group: new(seriesPalette),
			// "a0" to "a9", then "aA" to "aF".
			Position: new("a" + string(positionDigits[i])),
		}
	}
	return named
}

// createBuiltins creates each built-in named color that is missing, with its shipped
// values, and never changes one that exists. It creates the "Series" palette when a
// missing color belongs in it and the palette is missing too.
func (s *Service) createBuiltins(
	ctx context.Context,
	db *gorp.DB,
	groups *group.Service,
) error {
	keys := lo.Map(series[:], func(s shipped, _ int) Key { return s.key })
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		var existing []Named
		if err := s.NewRetrieve().
			Where(MatchKeys(keys...)).
			SkipMissingKeys().
			Entries(&existing).
			Exec(ctx, tx); err != nil {
			return err
		}
		stored := set.New(
			lo.Map(existing, func(n Named, _ int) Key { return n.Key })...,
		)
		missing := lo.Reject(Builtins(), func(n Named, _ int) bool {
			return stored.Contains(n.Key)
		})
		if len(missing) == 0 {
			return nil
		}
		hasPalette, err := groups.NewRetrieve().
			Where(group.MatchKeys(seriesPalette)).
			Exists(ctx, tx)
		if err != nil {
			return err
		}
		if !hasPalette {
			palette := group.Group{
				Key:   seriesPalette,
				Name:  "Series",
				Scope: resource.TypeColor,
			}
			if err = groups.NewWriter(tx).Create(ctx, &palette); err != nil {
				return err
			}
		}
		return s.NewWriter(tx).CreateMany(ctx, missing)
	})
}
