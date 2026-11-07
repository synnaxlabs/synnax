// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";

import { type OpMode, type OpModeChoice } from "@/feature/ethercat/device/types";

/** The AssignActivate bit that gives the sync unit to the PDI. */
const PDI_SYNC_UNIT = 0x0001;
/** The AssignActivate bit that turns on SYNC0. */
const SYNC0_ACTIVE = 0x0200;
/** The AssignActivate bit that turns on SYNC1. */
const SYNC1_ACTIVE = 0x0400;

/** Whether the mode is SM sync, which runs no clock ticks. */
export const isSM = (mode: OpMode): boolean => mode.assignActivate === 0;

const ownedByPDI = (mode: OpMode): boolean =>
  (mode.assignActivate & PDI_SYNC_UNIT) !== 0;

const runsSync0 = (mode: OpMode): boolean => (mode.assignActivate & SYNC0_ACTIVE) !== 0;

/**
 * Whether the Driver can set SYNC0 and SYNC1 cycles for the mode: it runs SYNC0, and
 * the master keeps its sync unit. A cycle on any other mode fails the start.
 */
export const takesCycles = (mode: OpMode): boolean =>
  runsSync0(mode) && !ownedByPDI(mode);

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

/** The choice of the mode at the index, with the mode's own cycles. */
export const choose = (modes: OpMode[], index: number): OpModeChoice => ({
  index,
  assignActivate: modes[index].assignActivate,
  sync0Cycle: 0,
  sync1Cycle: 0,
});

const span = (ns: number): string => new TimeSpan(ns).toString("full");

const busCycles = (factor: number): string =>
  factor === 1 ? "each bus cycle" : `every ${factor} bus cycles`;

/** What the mode gives for SYNC0, such as "each bus cycle", or null for nothing. */
const sync0Text = (mode: OpMode): string | null => {
  if (mode.sync0Cycle !== 0) return `every ${span(mode.sync0Cycle)}`;
  if (mode.sync0Factor > 0) return busCycles(mode.sync0Factor);
  if (mode.sync0Factor < 0) return `${-mode.sync0Factor} times each bus cycle`;
  return null;
};

const NO_SYNC0 = "This mode gives no SYNC0 cycle. Type one.";

/** What the mode gives for SYNC1, or null when the mode leaves SYNC1 off. */
const sync1Text = (mode: OpMode): string | null => {
  if (!hasSync1(mode)) return null;
  if (mode.sync1Factor > 1) return `every ${mode.sync1Factor} SYNC0 ticks`;
  if (mode.sync1Factor < 0) return busCycles(-mode.sync1Factor);
  return "with each SYNC0 tick";
};

/**
 * One sentence that says when the clock ticks of a DC mode run. A null choice runs the
 * mode's own cycles.
 */
export const describeTicks = (mode: OpMode, choice: OpModeChoice | null): string => {
  const sync0Cycle = choice == null ? 0 : choice.sync0Cycle;
  const sync1Cycle = choice == null ? 0 : choice.sync1Cycle;
  const sync0 = sync0Cycle !== 0 ? `every ${span(sync0Cycle)}` : sync0Text(mode);
  if (sync0 == null) return NO_SYNC0;
  const sync1 = sync1Cycle !== 0 ? `every ${span(sync1Cycle)}` : sync1Text(mode);
  if (sync1 == null) return `SYNC0 ticks ${sync0}. SYNC1 is off.`;
  const on = hasSync1(mode)
    ? ""
    : " The cycle turns on SYNC1, which the mode leaves off.";
  return `SYNC0 ticks ${sync0}. SYNC1 ticks ${sync1}.${on}`;
};

/** A short description of the mode for its option in the select. */
export const describeMode = (mode: OpMode): string => {
  if (isSM(mode)) return "Each frame from the Driver times the device.";
  if (ownedByPDI(mode)) return "The device sets its own clock ticks.";
  if (!runsSync0(mode)) return "The DC clock runs with no SYNC0 ticks.";
  const sync0 = sync0Text(mode);
  if (sync0 == null) return "You must type a SYNC0 cycle for this mode.";
  const sync1 = sync1Text(mode);
  return sync1 == null
    ? `Clock ticks: SYNC0 ${sync0}.`
    : `Clock ticks: SYNC0 ${sync0}, SYNC1 ${sync1}.`;
};

/** The error of an auto SYNC0 cycle on a mode that gives none, or null. */
export const sync0Error = (mode: OpMode, choice: OpModeChoice | null): string | null =>
  (choice == null || choice.sync0Cycle === 0) && sync0Text(mode) == null
    ? NO_SYNC0
    : null;

/**
 * The error of a chosen SYNC1 cycle that is not a multiple of the SYNC0 cycle, or null.
 * A SYNC0 cycle that comes from the bus period is not known here, so the Driver checks
 * it at start.
 */
export const sync1Error = (
  mode: OpMode,
  choice: OpModeChoice | null,
): string | null => {
  if (choice == null || choice.sync1Cycle === 0) return null;
  const sync0 = choice.sync0Cycle !== 0 ? choice.sync0Cycle : mode.sync0Cycle;
  if (sync0 === 0 || choice.sync1Cycle % sync0 === 0) return null;
  return "SYNC1 cycle must be a multiple of the SYNC0 cycle";
};
