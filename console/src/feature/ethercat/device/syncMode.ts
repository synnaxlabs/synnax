// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";

import {
  type OpMode,
  type OpModeChoice,
  type SlaveProperties,
} from "@/feature/ethercat/device/types";

/** The AssignActivate bit that turns on SYNC1. */
const SYNC1_ACTIVE = 0x0400;

/** Whether the mode is SM sync, which runs no clock ticks. */
export const isSM = (mode: OpMode): boolean => mode.assignActivate === 0;

/** Whether the mode runs SYNC1 when no cycle replaces its own. */
const hasSync1 = (mode: OpMode): boolean => (mode.assignActivate & SYNC1_ACTIVE) !== 0;

/** The name of the mode, or "Mode N" when the device gives none. */
export const modeName = (mode: OpMode, index: number): string =>
  mode.name === "" ? `Mode ${index + 1}` : mode.name;

/**
 * The index of the mode the Driver runs with no choice: the SM mode when the device
 * lists one, else its first mode. Null when the device lists no modes.
 */
export const autoIndex = (modes: OpMode[]): number | null => {
  if (modes.length === 0) return null;
  const sm = modes.findIndex(isSM);
  return sm === -1 ? 0 : sm;
};

/** Whether the device no longer lists the mode of the choice. */
export const isStale = (modes: OpMode[], choice: OpModeChoice): boolean =>
  modes[choice.index]?.assignActivate !== choice.assignActivate;

/** A choice being edited. Index null is auto. A cycle of 0 is the mode's own. */
export interface Draft {
  index: number | null;
  sync0: number;
  sync1: number;
}

export const draftFromProperties = ({ opMode }: SlaveProperties): Draft =>
  opMode == null
    ? { index: null, sync0: 0, sync1: 0 }
    : { index: opMode.index, sync0: opMode.sync0Cycle, sync1: opMode.sync1Cycle };

/** The stored choice for the draft, or undefined for auto. */
export const choiceFromDraft = (
  modes: OpMode[],
  draft: Draft,
  stored?: OpModeChoice,
): OpModeChoice | undefined => {
  if (draft.index == null) return undefined;
  const assignActivate =
    modes[draft.index]?.assignActivate ?? stored?.assignActivate ?? 0;
  return {
    index: draft.index,
    assignActivate,
    sync0Cycle: draft.sync0,
    sync1Cycle: draft.sync1,
  };
};

const span = (ns: number): string => new TimeSpan(ns).toString("full");

const busCycles = (factor: number): string =>
  factor === 1 ? "each bus cycle" : `every ${factor} bus cycles`;

/** What the mode gives for SYNC0, such as "each bus cycle". */
const sync0Text = (mode: OpMode): string => {
  if (mode.sync0Cycle !== 0) return `every ${span(mode.sync0Cycle)}`;
  if (mode.sync0Factor > 0) return busCycles(mode.sync0Factor);
  if (mode.sync0Factor < 0) return `${-mode.sync0Factor} times each bus cycle`;
  return "each bus cycle";
};

/** What the mode gives for SYNC1, or null when the mode leaves SYNC1 off. */
const sync1Text = (mode: OpMode): string | null => {
  if (!hasSync1(mode)) return null;
  if (mode.sync1Factor > 1) return `every ${mode.sync1Factor} SYNC0 ticks`;
  if (mode.sync1Factor < 0) return busCycles(-mode.sync1Factor);
  return "with each SYNC0 tick";
};

/** One sentence that says when the clock ticks of a DC mode run. */
export const describeTicks = (mode: OpMode, draft: Draft): string => {
  const sync0 = draft.sync0 !== 0 ? `every ${span(draft.sync0)}` : sync0Text(mode);
  const sync1 = draft.sync1 !== 0 ? `every ${span(draft.sync1)}` : sync1Text(mode);
  if (sync1 == null) return `SYNC0 ticks ${sync0}. SYNC1 is off.`;
  const on = hasSync1(mode)
    ? ""
    : " The cycle turns on SYNC1, which the mode leaves off.";
  return `SYNC0 ticks ${sync0}. SYNC1 ticks ${sync1}.${on}`;
};

/** A short description of the mode for its option in the select. */
export const describeMode = (mode: OpMode): string => {
  if (isSM(mode)) return "Each frame from the Driver times the device.";
  const sync1 = sync1Text(mode);
  return sync1 == null
    ? `Clock ticks: SYNC0 ${sync0Text(mode)}.`
    : `Clock ticks: SYNC0 ${sync0Text(mode)}, SYNC1 ${sync1}.`;
};

/** The error of a SYNC1 cycle that cannot run with the SYNC0 cycle, or null. */
export const sync1Error = (draft: Draft): string | null => {
  if (draft.sync0 === 0 || draft.sync1 === 0) return null;
  if (draft.sync1 % draft.sync0 !== 0)
    return "SYNC1 cycle must be a multiple of the SYNC0 cycle";
  return null;
};
