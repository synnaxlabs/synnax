// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


/** @returns the separator of a path on the Driver: a backslash for a Windows path. */
export const separatorOf = (path: string): string =>
  path.includes("\\") && !path.includes("/") ? "\\" : "/";

/** @returns the folder that holds the path, or an empty string for a relative name. */
export const dirnameOf = (path: string): string => {
  const sep = separatorOf(path);
  const i = path.lastIndexOf(sep);
  if (i < 0) return "";
  if (i === 0) return sep;
  const dir = path.slice(0, i);
  return /^[A-Za-z]:$/.test(dir) ? `${dir}${sep}` : dir;
};

/** @returns whether the path is the folder or inside it. */
export const inside = (path: string, folder: string): boolean => {
  if (path === folder) return true;
  const sep = separatorOf(folder);
  return path.startsWith(folder.endsWith(sep) ? folder : `${folder}${sep}`);
};

/** A folder on the way to a path, from its root. */
export interface Segment {
  label: string;
  path: string;
}

/** @returns the folders on the way to the path, from its root. */
export const segmentsOf = (path: string): Segment[] => {
  if (path === "") return [];
  const sep = separatorOf(path);
  const segments: Segment[] = [];
  let acc = "";
  if (path.startsWith(sep)) {
    acc = sep;
    segments.push({ label: sep, path: sep });
  }
  path
    .split(sep)
    .filter((part) => part !== "")
    .forEach((part, i) => {
      if (i === 0 && sep === "\\") acc = `${part}${sep}`;
      else acc = acc.endsWith(sep) ? `${acc}${part}` : `${acc}${sep}${part}`;
      segments.push({ label: part, path: acc });
    });
  return segments;
};
