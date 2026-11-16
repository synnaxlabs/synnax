// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "path";

import { type Source } from "@/util/checks/check";

const escapeData = (s: string): string =>
  s.replace(/%/g, "%25").replace(/\r/g, "%0D").replace(/\n/g, "%0A");

const escapeProperty = (s: string): string =>
  escapeData(s).replace(/:/g, "%3A").replace(/,/g, "%2C");

/**
 * Formats a warning as a GitHub Actions workflow command, which shows on the pull
 * request at the source line when at is known. GitHub reads the file relative to root,
 * the root of the repository.
 */
export const warning = (message: string, at: Source | null, root: string): string => {
  if (at == null) return `::warning::${escapeData(message)}`;
  const file = path.relative(root, path.resolve(at.file)).split(path.sep).join("/");
  return `::warning file=${escapeProperty(file)},line=${at.line}::${escapeData(message)}`;
};
