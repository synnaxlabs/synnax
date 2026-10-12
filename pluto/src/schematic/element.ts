// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { type theme } from "@synnaxlabs/lyra/theme";
import { type color } from "@synnaxlabs/x";

import { Edge } from "@/schematic/edge";
import { Node } from "@/schematic/node";

export const elementConfigZ = schematic.elementConfigZ;
export type ElementConfig = schematic.ElementConfig;

export const ELEMENT_REGISTRY = { ...Node.REGISTRY, ...Edge.REGISTRY };

const DEFAULT_COLOR_FALLBACKS: Required<Node.ColorFallbacks> = {
  fillColor: Node.Form.noFillFallback,
  strokeColor: Node.Form.defaultFallback,
  textColor: Node.Form.defaultFallback,
};

/**
 * @returns the color an element of the variant paints while its color at key is
 * absent.
 */
export const colorFallback = (
  key: Node.ColorKey,
  variant: schematic.ElementConfigType,
  theme: theme.Theme,
): color.Color =>
  (ELEMENT_REGISTRY[variant].colorFallbacks?.[key] ?? DEFAULT_COLOR_FALLBACKS[key])(
    theme,
  );
