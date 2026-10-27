// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The browser imports this file, so it must not import anything that reaches the
// database.

import { type schema } from "@/server/db";

export const MAX_NAME_LENGTH = 64;

/** VERSION_PATTERN matches a major.minor Core version, like 0.62. */
export const VERSION_PATTERN = /^\d+\.\d+$/;

/** KEY_FILE_EXTENSION is what the Console's file picker filters on. */
export const KEY_FILE_EXTENSION = "lic";

const slug = (text: string): string =>
  text.replace(/[^a-z0-9]+/gi, "-").replace(/^-|-$/g, "");

/**
 * filename is the name a license key downloads as: its license label, then the name of
 * the machine the key is for, if any.
 */
export const filename = (label: string, machine?: string | null): string => {
  const stem = [label, machine ?? ""]
    .map(slug)
    .filter((s) => s !== "")
    .join("-");
  return `${stem === "" ? "synnax" : stem}.${KEY_FILE_EXTENSION}`;
};

/** term describes how long a license lasts and which Core versions it covers. */
export const term = (lic: schema.License): string => {
  if (lic.term === "perpetual") return `Perpetual, up to v${lic.maxVersion}`;
  const until = `Until ${lic.expiresAt?.toISOString().slice(0, 10) ?? ""}`;
  return lic.maxVersion == null ? until : `${until}, then up to v${lic.maxVersion}`;
};

/** channels describes a cap on channels per Core, where 0 is no cap. */
export const channels = (cap: number): string =>
  cap === 0 ? "Unlimited" : String(cap);
