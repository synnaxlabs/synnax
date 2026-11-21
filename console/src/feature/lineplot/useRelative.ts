// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlot } from "@synnaxlabs/pluto";

import { FLAGS } from "@/flags";
import { Session } from "@/session";

/**
 * @returns whether the plot draws its x axis in relative time. A plot set to relative
 * time with no range that has a zero draws in absolute time and keeps its stored mode.
 */
export const useRelative = (): boolean => {
  const x1 = LinePlot.useXAxis({ axisKey: "x1" });
  const refs = LinePlot.useRanges();
  return FLAGS.compare && LinePlot.isRelative(x1) && refs.some(Session.Range.hasZero);
};
