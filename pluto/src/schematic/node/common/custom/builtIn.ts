// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { Primitive } from "@/schematic/node/common/primitive";

export interface BuiltInParams extends Pick<schematic.symbol.Spec, "svg" | "handles"> {}

/**
 * @returns the spec of a static built-in symbol. Its scale is the shrink Primitive.SVG
 * gives a built-in drawing, so Drawing draws it at the size of that drawing.
 */
export const createBuiltIn = ({
  svg,
  handles,
}: BuiltInParams): schematic.symbol.Spec => ({
  svg,
  handles,
  states: [],
  variant: "static",
  scale: Primitive.BASE_SCALE,
  strokeScaled: false,
});
