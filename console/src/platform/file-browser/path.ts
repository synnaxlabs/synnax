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
 * @returns the root of an absolute path, such as `/`, `C:\`, or `\\server\share`, and
 * the rest of the path. Returns null for a path that is not absolute.
 */
const splitRoot = (path: string): [root: string, rest: string] | null => {
  const unc = /^\\\\([^\\/]+)[\\/]+([^\\/]+)/.exec(path);
  if (unc != null) return [`\\\\${unc[1]}\\${unc[2]}`, path.slice(unc[0].length)];
  if (/^[A-Za-z]:[\\/]/.test(path)) return [`${path.slice(0, 2)}\\`, path.slice(3)];
  if (path.startsWith("/")) return ["/", path.slice(1)];
  return null;
};

const partsOf = (root: string, rest: string): string[] =>
  rest.split(root === "/" ? "/" : /[\\/]/).filter((part) => part !== "");

const join = (root: string, parts: string[]): string => {
  if (parts.length === 0) return root;
  const sep = root === "/" ? "/" : "\\";
  return `${root}${root.endsWith(sep) ? "" : sep}${parts.join(sep)}`;
};

/**
 * @returns the absolute path in the form that the Driver lists it: a Windows path with
 * backslashes, and no `.`, `..`, or repeated separators. Returns an empty string for a
 * path that is not absolute.
 */
export const normalize = (path: string): string => {
  const split = splitRoot(path);
  if (split == null) return "";
  const [root, rest] = split;
  const parts: string[] = [];
  for (const part of partsOf(root, rest))
    if (part === "..") parts.pop();
    else if (part !== ".") parts.push(part);
  return join(root, parts);
};

/**
 * @returns the folder that holds the path, the root itself for a root, or an empty
 * string for a relative name.
 */
export const dirnameOf = (path: string): string => {
  const split = splitRoot(path);
  if (split == null) {
    const i = path.lastIndexOf(separatorOf(path));
    return i < 0 ? "" : path.slice(0, i);
  }
  const [root, rest] = split;
  // A trailing separator ends the path with an empty name, so the folder is the path.
  const names = rest.split(root === "/" ? "/" : /[\\/]/).slice(0, -1);
  return join(root, names.filter(Boolean));
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
  const split = splitRoot(path);
  if (split == null) return [];
  const [root, rest] = split;
  const parts = partsOf(root, rest);
  return [
    { label: root === "/" ? root : root.replace(/\\$/, ""), path: root },
    ...parts.map((label, i) => ({ label, path: join(root, parts.slice(0, i + 1)) })),
  ];
};
