// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { device } from "@synnaxlabs/client";
import { unique } from "@synnaxlabs/x";
import { z } from "zod";

export const PREFIX = "ethercat";

export const SCAN_TYPE = `${PREFIX}_scan`;

export const READ_TYPE = `${PREFIX}_read`;

export const WRITE_TYPE = `${PREFIX}_write`;

const devicesZ = z.object({ channels: z.array(z.object({ device: z.string() })) });

/**
 * @returns the keys of the slaves that the channels of a read or write task config use,
 * or none when the config does not hold channels.
 */
export const devicesOf = (config: unknown): device.Key[] => {
  const parsed = devicesZ.safeParse(config);
  if (!parsed.success) return [];
  return unique
    .unique(parsed.data.channels.map(({ device }) => device))
    .filter((key) => key !== "");
};
