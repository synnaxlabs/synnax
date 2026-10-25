// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { Rack } from "@synnaxlabs/pluto";

/** Whether the rack's Driver is running. */
export const isRunning = (r: rack.Payload): boolean => r.status?.variant === "success";

export interface BlockedReasonArgs {
  /** The integration the rack must run, such as "opc". */
  integration: string;
  /** The integration's display name, such as "OPC UA". */
  name: string;
  /** Checks only this rack. */
  rack?: rack.Key;
}

const canConnect = (r: rack.Payload, integration: string): boolean =>
  isRunning(r) && r.integrations.includes(integration);

/**
 * Returns the rack a new connection starts with: the only running rack with the
 * integration, or undefined when there are none or several.
 */
export const getDefaultRack = (
  racks: rack.Payload[],
  integration: string,
): rack.Key | undefined => {
  const usable = racks.filter((r) => canConnect(r, integration));
  return usable.length === 1 ? usable[0].key : undefined;
};

/** Returns why no rack can connect a device of the integration, or null if one can. */
export const getBlockedReason = (
  racks: rack.Payload[],
  { integration, name, rack: key }: BlockedReasonArgs,
): string | null => {
  if (key == null)
    return racks.some((r) => canConnect(r, integration))
      ? null
      : `No running Driver has ${name} enabled`;
  const r = racks.find((r) => r.key === key);
  if (r == null) return "This Driver no longer exists";
  if (!isRunning(r)) return `${r.name} is not running`;
  if (!r.integrations.includes(integration))
    return `${r.name} does not have ${name} enabled`;
  return null;
};

export interface UseBlockedReasonArgs extends Omit<BlockedReasonArgs, "rack"> {
  /** The rack to check. Null or 0 checks every rack. */
  rack: rack.Key | null;
}

/** Returns {@link getBlockedReason} for the Core's racks, or null until they resolve. */
export const useBlockedReason = ({
  integration,
  name,
  rack: key,
}: UseBlockedReasonArgs): string | null => {
  const { data } = Rack.useResultMultiple({});
  if (data == null) return null;
  return getBlockedReason(data, {
    integration,
    name,
    rack: key == null || key === 0 ? undefined : key,
  });
};
