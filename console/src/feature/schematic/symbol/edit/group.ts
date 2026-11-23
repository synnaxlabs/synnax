// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Schematic } from "@synnaxlabs/pluto";

const holdsAll = (group: Element, shapes: Element[]): boolean =>
  shapes.every((s) => s !== group && group.contains(s));

/**
 * @returns the drawing markup with a name on the nearest group that holds every shape
 * of selectors, and the selector of that group. Null when no group holds them all.
 */
export const parentGroup = (
  markup: string,
  selectors: string[],
): Schematic.Node.Region.Named | null => {
  const root = Schematic.Node.Region.parseSVG(markup);
  if (root == null) throw new Error("the markup of the drawing does not parse");
  const shapes = [...new Set(selectors.flatMap((s) => [...root.querySelectorAll(s)]))];
  let el: Element | null = shapes[0]?.parentElement ?? null;
  for (; el != null && el !== root; el = el.parentElement)
    if (el instanceof SVGGElement && holdsAll(el, shapes))
      return Schematic.Node.Region.name(markup, root, el);
  return null;
};
