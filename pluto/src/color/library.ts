// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type group } from "@synnaxlabs/client";
import type { Color } from "@synnaxlabs/lyra/color";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color as xcolor, compare } from "@synnaxlabs/x";

import { useResultList } from "@/color/queries";
import { Group } from "@/group";

/** The key of the palette that holds the named colors in no palette. */
export const NO_PALETTE = "";

const useAll = (): color.Named[] => {
  const list = useResultList({});
  return list.variant === "success" ? list.data : [];
};

/**
 * Returns every named color as the palettes of a color picker: the palettes by name,
 * then the colors in no palette under "No palette". Each palette lists its colors in
 * palette order.
 */
export const usePalettes = (): Color.Palette[] => {
  const all = useAll();
  const groups = Group.useResultList({
    where: { scope: { eq: color.TYPE_RESOURCE_ID.type } },
  });
  if (groups.variant !== "success") return [];
  const members = Map.groupBy(
    [...all].sort(color.compare),
    (c): group.Key => c.group ?? NO_PALETTE,
  );
  const palettes = [...groups.data]
    .sort(compare.by((g) => g.name))
    .map(({ key, name }) => ({ key, name, colors: members.get(key) ?? [] }));
  const loose = members.get(NO_PALETTE);
  if (loose != null)
    palettes.push({ key: NO_PALETTE, name: "No palette", colors: loose });
  return palettes;
};

const useSeriesNamed = (): color.Named[] =>
  useAll()
    .filter((c) => c.group === color.SERIES_PALETTE)
    .sort(color.compare);

/** Returns the keys of the series colors in palette order. */
export const useSeries = (): color.Key[] => useSeriesNamed().map((c) => c.key);

/** Returns the series colors in palette order, resolved for the theme. */
export const useSeriesValues = (): xcolor.Color[] => {
  const t = Theming.use();
  return useSeriesNamed().map((c) => Theming.resolve(c, t));
};

/**
 * Returns a function that resolves a color reference to its value for the theme. The
 * function returns undefined for an absent reference or a named color that is not
 * loaded.
 */
export const useResolve = (): ((ref?: color.Ref) => xcolor.Color | undefined) => {
  const t = Theming.use();
  const named = new Map(useAll().map((c) => [c.key, c]));
  return (ref) => {
    if (ref == null) return undefined;
    if (ref.type === "fixed") return xcolor.construct(ref.value);
    const n = named.get(ref.key);
    return n == null ? undefined : Theming.resolve(n, t);
  };
};
