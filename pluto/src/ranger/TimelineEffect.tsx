// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type NumericTimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Icon } from "@/icon";
import { Input } from "@/input";
import { UNSET } from "@/ranger/move";
import { getStage, type Stage, STAGE_ICONS, STAGE_NAMES } from "@/ranger/stage";
import { Text } from "@/text";

/** One thing a commit would change about a range. */
export type Change =
  | { kind: "stage"; from: Stage; to: Stage }
  | {
      kind: "set" | "moved" | "cleared";
      bound: Input.Bound;
      before: number;
      after: number;
    };

const MICROSECOND = Number(TimeSpan.MICROSECOND.valueOf());

/**
 * Lists what committing `next` would change about `range` besides the end being
 * edited: the stage when the commit crosses now, then each other end it sets, moves,
 * or clears.
 */
export const describeChanges = (
  range: NumericTimeRange,
  next: NumericTimeRange,
  editing?: Input.Bound,
): Change[] => {
  const out: Change[] = [];
  // A candidate of now can round up to the next float64, a fraction of a microsecond
  // ahead, so the stages are read a microsecond late.
  const now = TimeStamp.now().add(TimeSpan.MICROSECOND);
  const from = getStage(range, now);
  const to = getStage(next, now);
  if (from !== to) out.push({ kind: "stage", from, to });
  for (const bound of ["start", "end"] as const) {
    if (bound === editing) continue;
    const before = range[bound];
    const after = next[bound];
    // Float64 nanoseconds carry noise below the microsecond.
    if (Math.abs(after - before) < MICROSECOND) continue;
    let kind: "set" | "moved" | "cleared" = "moved";
    if (after >= UNSET) kind = "cleared";
    else if (before >= UNSET) kind = "set";
    out.push({ kind, bound, before, after });
  }
  return out;
};

export interface TimelineEffectProps {
  changes: Change[];
  /** The finest unit the effect shows. */
  resolution?: TimeSpan;
}

const VERBS = { set: "Sets", moved: "Moves" } as const;

/**
 * Renders {@link describeChanges}: a stage change as its two stages, a cleared end in
 * the warning color with the value it loses, and a set or moved end quietly.
 */
export const TimelineEffect = ({
  changes,
  resolution,
}: TimelineEffectProps): ReactElement => {
  const now = TimeStamp.now();
  return (
    <>
      {changes.map((change) => {
        if (change.kind === "stage") {
          const From = STAGE_ICONS[change.from];
          const To = STAGE_ICONS[change.to];
          return (
            <Text.Text key="stage" level="small" gap="tiny">
              <Text.Text el="span" level="small" color={9} gap="tiny">
                <From />
                {STAGE_NAMES[change.from]}
              </Text.Text>
              <Icon.Arrow.Right />
              <To />
              {STAGE_NAMES[change.to]}
            </Text.Text>
          );
        }
        const { kind, bound, before, after } = change;
        if (kind === "cleared")
          return (
            <Text.Text key={bound} level="small" status="warning" gap="small">
              Clears {bound}
              <Text.Text el="span" level="small" color={9}>
                was {Input.formatInstant(before, now, resolution)}
              </Text.Text>
            </Text.Text>
          );
        return (
          <Text.Text key={bound} level="small" color={9}>
            {VERBS[kind]} {bound} to {Input.formatInstant(after, now, resolution)}
          </Text.Text>
        );
      })}
    </>
  );
};
