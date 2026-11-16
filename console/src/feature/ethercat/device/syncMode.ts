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

/** Whether the device no longer lists the mode of the choice. */
export const isStale = (modes: OpMode[], choice: OpModeChoice): boolean =>
  modes[choice.index]?.assignActivate !== choice.assignActivate;

/**
 * The index of the mode that the Driver runs with no choice: that of catalog, the
 * choice that the catalog of the device gives and the device lists, else the SM mode
 * when the device lists one, else its first mode. Null when the device lists no modes.
 */
export const autoIndex = (
  modes: OpMode[],
  catalog: OpModeChoice | null = null,
): number | null => {
  if (catalog != null) return catalog.index;
  if (modes.length === 0) return null;
  const sm = modes.findIndex(isSM);
  return sm === -1 ? 0 : sm;
};

/** The choice of the mode at the index, with the mode's own cycles. */
export const choose = (modes: OpMode[], index: number): OpModeChoice => ({
  index,
  assignActivate: modes[index].assignActivate,
  sync0Cycle: 0,
  sync1Cycle: 0,
});

const span = (ns: number): string => new TimeSpan(ns).toString("full") || "0s";

/** The largest value that the SYNC0 and SYNC1 cycle registers hold. */
const REGISTER_LIMIT = 0xffff_ffff;

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

/** The SYNC0 cycle the Driver runs, or 0 when it comes from the bus period. */
const sync0Cycle = (mode: OpMode, choice: OpModeChoice | null): number =>
  choice != null && choice.sync0Cycle !== 0 ? choice.sync0Cycle : mode.sync0Cycle;

/**
 * Why the Driver fails to start with the SYNC0 cycle, or null: the mode gives no cycle,
 * or the cycle is over the limit of its register.
 */
export const sync0Error = (
  mode: OpMode,
  choice: OpModeChoice | null,
): string | null => {
  if ((choice == null || choice.sync0Cycle === 0) && sync0Text(mode) == null)
    return NO_SYNC0;
  const sync0 = sync0Cycle(mode, choice);
  if (sync0 <= REGISTER_LIMIT) return null;
  return `SYNC0 cycle of ${span(sync0)} is over the limit of ${span(REGISTER_LIMIT)}`;
};

/**
 * Why the Driver fails to start with the SYNC1 cycle, or null: it is not a multiple of
 * the SYNC0 cycle, or its register value is out of range. A cycle that comes from the
 * bus period is not known here, so the Driver checks it at start.
 */
export const sync1Error = (
  mode: OpMode,
  choice: OpModeChoice | null,
): string | null => {
  const chosen = choice == null ? 0 : choice.sync1Cycle;
  const sync0 = sync0Cycle(mode, choice);
  if ((chosen === 0 && !hasSync1(mode)) || sync0 === 0) return null;
  if (chosen === 0 && mode.sync1Factor < 0) return null;
  const sync1 = chosen !== 0 ? chosen : sync0 * Math.max(mode.sync1Factor, 1);
  if (sync1 % sync0 !== 0) return "SYNC1 cycle must be a multiple of the SYNC0 cycle";
  const shift = mode.sync1Shift;
  const register = sync1 - sync0 + shift;
  if (register < 0)
    return `SYNC1 shift of ${span(shift)} is under the limit of ${span(sync0 - sync1)}`;
  if (register <= REGISTER_LIMIT) return null;
  const limit = REGISTER_LIMIT + sync0 - shift;
  return `SYNC1 cycle of ${span(sync1)} is over the limit of ${span(limit)}`;
};
