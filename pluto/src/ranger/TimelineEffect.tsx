// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type NumericTimeRange, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Icon } from "@/icon";
import { Input } from "@/input";
import { moveEnd, moveStart, UNSET } from "@/ranger/move";
import { getStage, STAGE_NAMES } from "@/ranger/stage";
import { Text } from "@/text";

export interface TimelineEffectProps {
  /** The range the cell holds one end of. */
  range: NumericTimeRange;
  /** Which end the cell holds. */
  bound: Input.Bound;
  /** The instant the cell would commit. */
  candidate: number;
}

/**
 * What committing a reading would change about the range besides the cell itself:
 * the other stamp when the edit drags it, and the stage when the edit crosses now.
 * Belongs in the `effect` of a {@link Input.DateTime} holding one end of `range`.
 */
export const TimelineEffect = ({
  range,
  bound,
  candidate,
}: TimelineEffectProps): ReactElement | null => {
  const start = bound === "start";
  const next = start ? moveStart(range, candidate) : moveEnd(range, candidate);
  const other = start ? "end" : "start";
  const before = start ? range.end : range.start;
  const after = start ? next.end : next.start;
  const stage = getStage(range);
  const nextStage = getStage(next);

  let moves: string | null = null;
  if (after !== before)
    moves =
      after >= UNSET
        ? `Clears ${other}`
        : `Moves ${other} to ${Input.formatInstant(after, TimeStamp.now())}`;
  if (moves == null && nextStage === stage) return null;

  return (
    <>
      {moves != null && (
        <Text.Text level="small" status="warning">
          {moves}
        </Text.Text>
      )}
      {nextStage !== stage && (
        <Text.Text level="small" color={9} gap="tiny">
          <Icon.Arrow.Right />
          {STAGE_NAMES[nextStage]}
        </Text.Text>
      )}
    </>
  );
};
