// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Schematic } from "@synnaxlabs/pluto";

/**
 * @returns the drawing markup with a name on the group that holds every shape of
 * selectors, and the selector of that group. Null when the shapes sit in no one group.
 */
export const parentGroup = (
  markup: string,
  selectors: string[],
): Schematic.Node.Region.Named | null => {
  const root = Schematic.Node.Region.parseSVG(markup);
  if (root == null) throw new Error("the markup of the drawing does not parse");
  const shapes = new Set(selectors.flatMap((s) => [...root.querySelectorAll(s)]));
  const parents = new Set([...shapes].map((s) => s.parentElement));
  const [parent] = parents;
  if (parents.size !== 1 || !(parent instanceof SVGGElement)) return null;
  return Schematic.Node.Region.name(markup, root, parent);
};
