// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";

import { Device } from "@/platform/device";

/**
 * @returns the NI MAX name of the device as an identifier, such as "cdaq1mod1", then a
 * short form of it, such as "c1m1".
 */
export const suggestIdentifiers = ({ location }: device.Device): string[] => {
  const parts = location.toLowerCase().match(/[a-z]+|\d+/g) ?? [];
  const full = parts.join("");
  const short = parts.map((p) => (/\d/.test(p) ? p : p[0])).join("");
  return [...new Set([full, short])].filter(
    (id) => Device.identifierZ.safeParse(id).success,
  );
};
