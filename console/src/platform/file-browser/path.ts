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

/**
 * @returns the absolute path in the form that the Driver lists it: a Windows path with
 * backslashes, and no `.`, `..`, or repeated separators. Returns an empty string for a
 * path that is not absolute.
 */
export const normalize = (path: string): string => {
  let root: string;
  if (/^[A-Za-z]:[\\/]/.test(path)) root = `${path.slice(0, 2)}\\`;
  else if (path.startsWith("\\\\")) root = "\\\\";
  else if (path.startsWith("/")) root = "/";
  else return "";
  const sep = root === "/" ? "/" : "\\";
  const parts: string[] = [];
  for (const part of path.slice(root.length).split(sep === "/" ? "/" : /[\\/]/))
    if (part === "..") parts.pop();
    else if (part !== "" && part !== ".") parts.push(part);
  return `${root}${parts.join(sep)}`;
};

/** @returns the folder that holds the path, or an empty string for a relative name. */
export const dirnameOf = (path: string): string => {
  const sep = separatorOf(path);
  const i = path.lastIndexOf(sep);
  if (i < 0) return "";
  if (i === 0) return sep;
  const dir = path.slice(0, i);
  return /^[A-Za-z]:$/.test(dir) ? `${dir}${sep}` : dir;
};

/** @returns the path with one separator at its end, as a folder is typed. */
export const withSeparator = (path: string): string => {
  const sep = separatorOf(path);
  return path.endsWith(sep) ? path : `${path}${sep}`;
};

/** @returns the last name in the path, or the path itself for a root. */
export const basenameOf = (path: string): string => {
  const name = path.slice(path.lastIndexOf(separatorOf(path)) + 1);
  return name === "" ? path : name;
};

/** @returns whether a name is hidden, as a name that starts with a dot is. */
export const hidden = (name: string): boolean => name.startsWith(".");

/** @returns the text with a leading ~ replaced by the home folder, once it is known. */
export const expandHome = (text: string, home: string | undefined): string =>
  home != null && text.startsWith("~") ? `${home}${text.slice(1)}` : text;

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
