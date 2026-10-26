// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type HastPluginDefinition } from "satteri";

// Single letters other than s and V are left out, because "step 1 A" is not a unit.
const UNITS = [
  "Hz",
  "kHz",
  "MHz",
  "GHz",
  "ns",
  "µs",
  "us",
  "ms",
  "s",
  "min",
  "V",
  "mV",
  "mA",
  "Ω",
  "°C",
  "°F",
  "KB",
  "MB",
  "GB",
  "TB",
  "bit",
  "bits",
  "byte",
  "bytes",
  "nanoseconds",
  "microseconds",
  "milliseconds",
  "seconds",
  "minutes",
  "hours",
  "days",
];

const NUMBER_UNIT = new RegExp(`(\\d) (?=(?:${UNITS.join("|")})(?!\\w))`, "g");

const CODE = new Set(["code", "pre"]);

/** Keeps a number and its unit, such as 50,000 Hz, on one line. Code is unchanged. */
export const quantities = (): HastPluginDefinition => ({
  name: "quantities",
  text: (node, ctx) => {
    const parent = ctx.parent(node);
    if (parent.type === "element" && CODE.has(parent.tagName)) return;
    const value = node.value.replace(NUMBER_UNIT, "$1\u00a0");
    if (value !== node.value) ctx.replaceNode(node, { type: "text", value });
  },
});
