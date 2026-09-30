// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { Telem } from "@/schematic/node/common/telem";
import { type telem } from "@/telem/aether";
import { type Scale as VisScale } from "@/vis/scale";

/** Stored shape of a live scale indicator, shared by every symbol that renders one. */
export type Config = schematic.ScaleIndicatorConfig;

/** source builds the smoothed read pipeline the indicator's value is drawn from. */
export const source = ({
  channel,
  rollingAverage,
}: Pick<Config, "channel" | "rollingAverage">): telem.NumberSourceSpec =>
  Telem.smoothedNumberSource({ channel, rollingAverage });

/** The indicator fields the vis scale draws from. */
export type VisConfig = Pick<
  Config,
  | "bounds"
  | "levelColor"
  | "strokeColor"
  | "textColor"
  | "units"
  | "stalenessColor"
  | "stalenessTimeout"
  | "notation"
  | "precision"
  | "side"
  | "caretSide"
  | "level"
  | "levelHidden"
  | "caretHidden"
  | "scaleHidden"
>;

/** visProps translates the stored indicator fields into the vis scale's props. */
export const visProps = ({
  levelColor,
  strokeColor,
  levelHidden,
  caretHidden,
  scaleHidden,
  bounds,
  textColor,
  units,
  stalenessColor,
  stalenessTimeout,
  notation,
  precision,
  side,
  caretSide,
  level,
}: VisConfig): Omit<VisScale.UseProps, "aetherKey" | "box"> & {
  showScale: boolean;
} => ({
  bounds,
  color: levelColor,
  axisColor: strokeColor,
  textColor,
  units,
  stalenessColor,
  stalenessTimeout,
  notation,
  precision,
  side,
  caretSide,
  level,
  showFill: !levelHidden,
  showCaret: !caretHidden,
  showScale: !scaleHidden,
});
