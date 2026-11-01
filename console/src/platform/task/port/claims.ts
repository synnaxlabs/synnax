// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReportIssue } from "@/platform/task/check";
import { type Cell, type State } from "@/platform/task/port/Grid";

/** A channel's hold on a port. */
export interface Claim {
  /** The port's name, such as "ai3" or "FIO4". */
  port: string;
  channel: string;
  /** The Driver skips a disabled channel, so its claim never conflicts. */
  disabled: boolean;
}

/** The claims on each port, by the port's name. */
export type Claims<C extends Claim = Claim> = Map<string, C[]>;

/** Groups the claims by the port each holds. */
export const group = <C extends Claim>(claims: C[]): Claims<C> => {
  const result: Claims<C> = new Map();
  claims.forEach((c) => {
    const list = result.get(c.port);
    if (list == null) result.set(c.port, [c]);
    else list.push(c);
  });
  return result;
};

/** Whether more than one enabled channel holds the port. */
export const conflicts = (claims: Claim[]): boolean =>
  claims.filter(({ disabled }) => !disabled).length > 1;

/** The state of the ports with the claims: in conflict when any port is. */
export const stateOf = (claims: Claim[][]): State => {
  if (claims.some(conflicts)) return "conflict";
  return claims.some((list) => list.length > 0) ? "used" : "free";
};

/** How many of the named ports hold a channel. */
export const usedCount = (ports: string[], claims: Claims): number =>
  ports.filter((port) => claims.has(port)).length;

/** Names the channel with the key by the name it shows by, or "" for none. */
export const describer =
  <C extends { key: string }>(channels: C[], name: (channel: C) => string) =>
  (key: string): string => {
    const channel = channels.find((c) => c.key === key);
    return channel == null ? "" : name(channel);
  };

export interface CellParams extends Pick<Cell, "key" | "labels" | "row" | "column"> {
  /** The ports the cell covers: one, or a port and the negative input of its pair. */
  ports: string[];
  claims: Claims;
  /** The keys of the selected channels. */
  selected: string[];
  /** Names the channel with the key. */
  describe: (channel: string) => string;
}

/**
 * A cell in the state of the claims on its ports. A click selects its first enabled
 * channel, and its tooltip names each channel on it.
 */
export const cell = ({
  ports,
  claims,
  selected,
  describe,
  ...rest
}: CellParams): Cell => {
  const lists = ports.map((port) => claims.get(port) ?? []);
  const all = lists.flat();
  const holders = [...new Map(all.map((c) => [c.channel, c])).values()];
  const names = holders
    .map(({ channel, disabled }) =>
      disabled ? `${describe(channel)} (disabled)` : describe(channel),
    )
    .join(" and ");
  const paired = ports.length > 1;
  const subject = ports.join(" and ");
  let tooltip = `${subject} is free`;
  if (holders.length === 1) tooltip = `${subject}: ${names}`;
  else if (holders.length > 1)
    tooltip = `${subject} ${paired ? "are" : "is"} used by ${names}`;
  return {
    ...rest,
    name: ports[0],
    paired,
    state: stateOf(lists),
    disabled: all.length > 0 && all.every((c) => c.disabled),
    selected: all.some((c) => selected.includes(c.channel)),
    channel: (all.find((c) => !c.disabled) ?? all[0])?.channel,
    tooltip,
  };
};

export interface ReportConflictsOptions<C extends Claim> {
  /** The field of the channel that names its port. Defaults to "port". */
  field?: string;
  /** Defaults to "Another channel uses ai3". */
  message?: (port: string, claim: C, enabled: C[]) => string;
}

/**
 * Reports each enabled channel that shares a port with another, once, on its field in
 * the config's channels. A channel that holds two such ports gets its first.
 */
export const reportConflicts = <C extends Claim>(
  claims: Claims<C>,
  channels: { key: string }[],
  report: ReportIssue,
  {
    field = "port",
    message = (port) => `Another channel uses ${port}`,
  }: ReportConflictsOptions<C> = {},
): void => {
  const indexOf = new Map(channels.map(({ key }, i) => [key, i]));
  const reported = new Set<string>();
  claims.forEach((list, port) => {
    if (!conflicts(list)) return;
    const enabled = list.filter((c) => !c.disabled);
    enabled.forEach((c) => {
      const i = indexOf.get(c.channel);
      if (i == null || reported.has(c.channel)) return;
      reported.add(c.channel);
      report(["channels", i, field], message(port, c, enabled));
    });
  });
};
