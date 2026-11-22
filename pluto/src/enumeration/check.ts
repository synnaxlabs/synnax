// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration } from "@synnaxlabs/client";
import { theme } from "@synnaxlabs/lyra/theme";
import { color } from "@synnaxlabs/x";

/** The least contrast that an entry color needs on the canvas of each theme. */
const MIN_CONTRAST = 3;
/** The least OKLab distance between two entry colors, in each kind of vision. */
const MIN_DISTANCE = 0.05;
/** The OKLab distance under which an entry color reads as an alarm color. */
const ALARM_DISTANCE = 0.1;

const THEMES = [
  { name: "light", theme: theme.themeZ.parse(theme.SYNNAX_LIGHT) },
  { name: "dark", theme: theme.themeZ.parse(theme.SYNNAX_DARK) },
] as const;

/** The color of an entry in each theme, or undefined when the symbol picks it. */
export type ResolveColor = (entry: enumeration.Entry) => theme.Pair | undefined;

const SIMULATIONS: ((c: color.Color) => color.Color)[] = color.DEFICIENCIES.map(
  (d) => (c) => color.simulate(c, d),
);

/** @returns whether the colors are too close in some theme, under the given vision. */
const close = (
  a: theme.Pair,
  b: theme.Pair,
  see: (c: color.Color) => color.Color,
): boolean =>
  THEMES.some(
    ({ theme: t }) =>
      color.distance(see(theme.resolve(a, t)), see(theme.resolve(b, t))) < MIN_DISTANCE,
  );

const isAlarm = (pair: theme.Pair): boolean =>
  THEMES.some(({ theme: t }) => {
    const c = theme.resolve(pair, t);
    return [t.colors.error.z, t.colors.warning.z].some(
      (alarm) => color.distance(c, alarm) < ALARM_DISTANCE,
    );
  });

/**
 * Returns the warnings for one entry of an enumeration, or none. The checks warn and
 * never block a save.
 */
export const check = (
  entry: enumeration.Entry,
  entries: enumeration.Entry[],
  resolve: ResolveColor,
): string[] => {
  const warnings: string[] = [];
  const others = entries.filter((e) => e.key !== entry.key);
  const name = entry.name.trim();
  if (name === "") warnings.push("It has no name, so only its color shows it.");
  else if (others.some((e) => e.name.trim() === name))
    warnings.push("Another entry has its name, so only color tells them apart.");
  const pair = resolve(entry);
  if (pair == null) return warnings;
  for (const { name: mode, theme: t } of THEMES)
    if (color.contrast(theme.resolve(pair, t), t.colors.gray.l0) < MIN_CONTRAST)
      warnings.push(`Its color is hard to see in the ${mode} theme.`);
  const colored = others.flatMap((e) => {
    const p = resolve(e);
    return p == null ? [] : [{ name: e.name, pair: p }];
  });
  const near = (see: (c: color.Color) => color.Color): string | undefined =>
    colored.find((o) => close(pair, o.pair, see))?.name;
  const twin = near((c) => c);
  if (twin != null) warnings.push(`Its color is almost the color of ${twin}.`);
  else
    for (const see of SIMULATIONS) {
      const lookalike = near(see);
      if (lookalike == null) continue;
      warnings.push(`With color blindness, its color looks like ${lookalike}.`);
      break;
    }
  if (!entry.abnormal && isAlarm(pair))
    warnings.push("It has an alarm color, but it is not abnormal.");
  return warnings;
};
