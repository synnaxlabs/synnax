// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const SPS = 7;

const hex = (byte: number): string => byte.toString(16).padStart(2, "0");

/**
 * @returns the NAL units of an access unit in Annex B format, without their start codes.
 * The units share the memory of the access unit.
 */
export const split = (accessUnit: Uint8Array): Uint8Array[] => {
  const units: Uint8Array[] = [];
  let start = -1;
  const push = (end: number): void => {
    // A 4-byte start code leaves its leading zero on the end of the previous unit.
    while (end > start && accessUnit[end - 1] === 0) end--;
    if (end > start) units.push(accessUnit.subarray(start, end));
  };
  for (let i = 0; i + 2 < accessUnit.length; i++) {
    if (accessUnit[i] !== 0 || accessUnit[i + 1] !== 0 || accessUnit[i + 2] !== 1)
      continue;
    if (start >= 0) push(i);
    start = i + 3;
    i += 2;
  }
  if (start >= 0) push(accessUnit.length);
  return units;
};

/**
 * @returns the sequence parameter set among the units, or undefined when they hold
 * none long enough to name the profile and level.
 */
export const spsOf = (units: Uint8Array[]): Uint8Array | undefined =>
  units.find((u) => (u[0] & 0x1f) === SPS && u.length >= 4);

/** @returns the WebCodecs codec string of a sequence parameter set, such as avc1.42e01f. */
export const codecOf = (sps: Uint8Array): string =>
  `avc1.${hex(sps[1])}${hex(sps[2])}${hex(sps[3])}`;
