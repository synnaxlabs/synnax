// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { strings, unique } from "@synnaxlabs/x";

import { Device } from "@/platform/device";

// NI MAX names a network chassis after its serial number, such as cDAQ9189-1D71E7C, and
// each of its modules after the chassis. No form of a name with the serial fits.
const SERIAL = /-[0-9a-f]{6,}(?=mod\d+$|$)/i;

/**
 * @returns the NI MAX name of the device as an identifier, such as "cdaq1mod1", then a
 * short form of it, such as "c1m1". The name leaves out a chassis serial number. When
 * no form fits, returns short forms of the device name.
 */
export const suggestIdentifiers = ({ location, name }: device.Device): string[] => {
  const parts =
    location
      .replace(SERIAL, "")
      .toLowerCase()
      .match(/[a-z]+|\d+/g) ?? [];
  const full = parts.join("");
  const short = parts.map((p) => (/\d/.test(p) ? p : p[0])).join("");
  const ids = unique
    .unique([full, short])
    .filter((id) => Device.identifierZ.safeParse(id).success);
  return ids.length > 0 ? ids : strings.createShortIdentifiers(name);
};
