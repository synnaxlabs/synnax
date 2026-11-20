// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import fs from "fs";
import path from "path";

/** @returns whether the file is inside the directory. */
export const contains = (dir: string, file: string): boolean =>
  file.startsWith(`${dir}${path.sep}`);

/**
 * @param file - An absolute path.
 * @returns the src directory of the package that holds the file, or null if the file is
 * not inside that directory.
 */
export const findSrc = (file: string): string | null => {
  for (let dir = path.dirname(file); dir !== path.dirname(dir); dir = path.dirname(dir))
    if (fs.existsSync(path.join(dir, "package.json"))) {
      const src = path.join(dir, "src");
      return contains(src, file) ? src : null;
    }
  return null;
};
