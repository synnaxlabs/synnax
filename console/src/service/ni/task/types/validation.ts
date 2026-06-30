// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { type z } from "zod";

type PortToIndexMap = Map<number, number>;

/**
 * Creates a port validator that ensures ports are not duplicated within the same device.
 * @param portTypeLabel - Optional label for the port type (e.g., "Counter", "Analog", "Digital").
 *                        If not provided or empty, the message will be "Port X has already been used..."
 * @returns A validator function for use with zod schemas
 */
export const createPortValidator =
  (portTypeLabel?: string) =>
  ({
    value: channels,
    issues,
  }: z.core.ParsePayload<{ port: number; device: device.Key }[]>) => {
    const deviceToPortMap = new Map<device.Key, PortToIndexMap>();
    channels.forEach(({ device, port }, i) => {
      if (!deviceToPortMap.has(device)) deviceToPortMap.set(device, new Map());
      const portToIndexMap = deviceToPortMap.get(device) as PortToIndexMap;
      if (!portToIndexMap.has(port)) {
        portToIndexMap.set(port, i);
        return;
      }
      const index = portToIndexMap.get(port) as number;
      const code = "custom";
      const prefix = portTypeLabel ? `${portTypeLabel} port` : "Port";
      const message = `${prefix} ${port} has already been used on another channel on the same device`;
      issues.push({ path: [index, "port"], code, message, input: channels });
      issues.push({ path: [i, "port"], code, message, input: channels });
    });
  };

/**
 * Simple port validator for v0 tasks (without device awareness).
 * Validates that ports are not duplicated within a channel array.
 * @param portTypeLabel - Optional label for the port type (e.g., "Counter").
 *                        If not provided or empty, the message will be "Port X has already been used..."
 * @returns A validator function for use with zod schemas
 */
export const createSimplePortValidator =
  (portTypeLabel?: string) =>
  ({ value: channels, issues }: z.core.ParsePayload<{ port: number }[]>) => {
    const portToIndexMap = new Map<number, number>();
    channels.forEach(({ port }, i) => {
      if (!portToIndexMap.has(port)) {
        portToIndexMap.set(port, i);
        return;
      }
      const index = portToIndexMap.get(port) as number;
      const code = "custom";
      const prefix = portTypeLabel ? `${portTypeLabel} port` : "Port";
      const message = `${prefix} ${port} has already been used on another channel`;
      issues.push({ path: [index, "port"], code, message, input: channels });
      issues.push({ path: [i, "port"], code, message, input: channels });
    });
  };
