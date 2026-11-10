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

/** The packages Stylelint checks, as directories relative to the repository root. */
export const PACKAGES = [
  "lyra",
  "pluto",
  "console",
  "site/docs",
  "site/www",
  "site/portal",
  "site/foundation",
  "site/common",
] as const;

/** A package Stylelint checks. */
export type Package = (typeof PACKAGES)[number];

/** Matches a class that a package's `CSS` helpers build. */
export const BEM_CLASS = /^(?:lyra|pluto|console)(?:-|--|__)/;

// Custom properties keep the prefix of the theme tokens in every package.
const PROPERTY_PREFIX = "pluto";

/** The class names and custom properties that the code of every package emits. */
export interface Source {
  /** @returns the package that holds the file, or undefined for one outside them. */
  packageOf: (file: string) => Package | undefined;
  /** @returns the packages whose code emits the class; empty when none does. */
  owners: (cls: string) => Set<Package>;
  /** @returns whether a stylesheet or code defines the custom property. */
  defined: (property: string) => boolean;
}

/**
 * An emitted name. A pattern stands for a name built from a value the scan cannot see.
 * Each of its groups must be a number or equal a string literal in the code of any
 * package.
 */
interface Name {
  exact?: string;
  pattern?: RegExp;
}

const DYNAMIC = "([\\w-]+)";
const NUMBER = /^\d+$/;
const CODE = /\.(tsx?|astro|html|mdx?)$/;
const STYLE = /\.(css|astro)$/;
const TEST = /\.(spec|test)\.tsx?$/;
const STYLE_BLOCK = /<style\b[^>]*>[\s\S]*?<\/style>/g;
const SKIPPED_DIRECTORIES = new Set(["node_modules", "dist", ".astro", ".turbo"]);

const HELPER = /(?<![\w.])(?:CSS\.)?(BEM|BE|BM|B|E|M|variable)\(/g;
const QUALIFIED_HELPER = /\bCSS\.(BEM|BE|BM|B|E|M|variable)\(/g;
// A name after a dot is a selector that reads the class, not one that sets it.
const CLASS_LITERAL =
  /(?<![\w.-])((?:lyra|pluto|console)(?:-|--|__)[a-z0-9][\w-]*)(\$\{)?/g;
// A property that code sets: a setProperty call, a style key, or an inline declaration.
// A read such as getPropertyValue("--x") defines nothing.
const PROPERTY_LITERAL =
  /(setProperty\(\s*)?["'`](--[a-z][\w-]*)(\$\{[^}]*\}[\w-]*)?(["'`]\s*\]?\s*:|\s*:)?/g;
const STRING_LITERAL = /(["'`])([\w-]+)\1/g;
const NAMED_EXPORT = /^\s*export (?:const|function) (\w+)/gm;
const MEMBER = /\bCSS\.(\w+)/g;
const PROPERTY_DECLARATION = /(?<![\w-])(--[\w-]+)\s*:/g;
const PROPERTY_REGISTRATION = /@property\s+(--[\w-]+)/g;
const DEFINE_VARS = /define:vars=\{(\{[^}]*\}|\w+)\}/g;
const OBJECT_KEY = /^["']?([\w-]+)["']?\s*(?::|$)/;

const escape = (text: string): string => text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

const toName = (source: string, exact: boolean): Name =>
  exact
    ? { exact: source.replace(/\\(.)/g, "$1") }
    : { pattern: new RegExp(`^${source}$`) };

/** @returns the source of the bracketed span that opens at `open`, or undefined. */
const span = (text: string, open: number): string | undefined => {
  const closers: string[] = [];
  for (let i = open; i < text.length; i++) {
    const c = text[i];
    const top = closers[closers.length - 1];
    if (top === '"' || top === "'") {
      if (c === "\\") i++;
      else if (c === top) closers.pop();
    } else if (top === "`") {
      if (c === "\\") i++;
      else if (c === "`") closers.pop();
      else if (c === "$" && text[i + 1] === "{") {
        closers.push("}");
        i++;
      }
    } else if (c === '"' || c === "'" || c === "`") closers.push(c);
    else if (c === "(") closers.push(")");
    else if (c === "[") closers.push("]");
    else if (c === "{") closers.push("}");
    else if (c === top) {
      closers.pop();
      if (closers.length === 0) return text.slice(open + 1, i);
    }
  }
  return undefined;
};

/** @returns the top-level, comma-separated parts of a bracketed span's source. */
const splitTopLevel = (source: string): string[] => {
  const parts: string[] = [];
  let from = 0;
  let depth = 0;
  let quote: string | undefined;
  for (let i = 0; i < source.length; i++) {
    const c = source[i];
    if (quote != null) {
      if (c === "\\") i++;
      else if (c === quote) quote = undefined;
    } else if (c === '"' || c === "'" || c === "`") quote = c;
    else if ("([{".includes(c)) depth++;
    else if (")]}".includes(c)) depth--;
    else if (c === "," && depth === 0) {
      parts.push(source.slice(from, i));
      from = i + 1;
    }
  }
  parts.push(source.slice(from));
  return parts.map((p) => p.trim()).filter((p) => p !== "");
};

interface Fragment {
  source: string;
  exact: boolean;
}

const STRING = /^(["'])([\w-]*)\1$/;
const TEMPLATE = /^`([^`]*)`$/;
const TERNARY = /^[^"'`]+\?\s*(["'])([\w-]*)\1\s*:\s*(["'])([\w-]*)\3$/;

const toFragment = (arg: string): Fragment => {
  const string = STRING.exec(arg);
  if (string != null) return { source: escape(string[2]), exact: true };
  const template = TEMPLATE.exec(arg);
  if (template != null)
    return {
      source: template[1]
        .split(/\$\{[^}]*\}/)
        .map(escape)
        .join(DYNAMIC),
      exact: !template[1].includes("${"),
    };
  const ternary = TERNARY.exec(arg);
  if (ternary != null)
    return { source: `(?:${escape(ternary[2])}|${escape(ternary[4])})`, exact: false };
  return { source: DYNAMIC, exact: false };
};

const literal = (text: string): Fragment => ({ source: escape(text), exact: true });

const join = (fragments: Fragment[], separator = ""): Fragment => ({
  source: fragments.map((f) => f.source).join(escape(separator)),
  exact: fragments.every((f) => f.exact),
});

/** @returns the name a BEM helper call builds, mirroring `newBEM` in Lyra. */
const helperName = (
  prefix: string,
  helper: string,
  args: string[],
): Name | undefined => {
  const parts = args.map((a) => toFragment(a.startsWith("...") ? "" : a));
  const [first, ...rest] = parts;
  if (first == null) return undefined;
  const block = join([literal(prefix), literal("-"), first]);
  let name: Fragment;
  switch (helper) {
    case "B":
      name = join([literal(prefix), literal("-"), join(parts, "-")]);
      break;
    case "E":
      name = join([literal(prefix), literal("__"), first]);
      break;
    case "M":
      name = join([literal(prefix), literal("--"), join(parts, "-")]);
      break;
    case "BE":
      name = join([block, literal("__"), join(rest, "-")]);
      break;
    case "BM":
      name = join([block, literal("--"), join(rest, "-")]);
      break;
    case "BEM": {
      const [element, ...modifiers] = rest;
      if (element == null) return undefined;
      name = join([block, literal("__"), element, literal("--"), join(modifiers, "-")]);
      break;
    }
    case "variable":
      name = join([
        literal("--"),
        literal(PROPERTY_PREFIX),
        literal("-"),
        join(parts, "-"),
      ]);
      break;
    default:
      throw new Error(`unknown BEM helper ${helper}`);
  }
  return toName(name.source, name.exact);
};

/** @returns the prefix of the `CSS` helpers that a file of the package calls. */
const helperPrefix = (pkg: Package, text: string): string | undefined => {
  if (pkg === "console" && text.includes('from "@/platform/css"')) return "console";
  if (pkg === "pluto" && text.includes('from "@/css"')) return "pluto";
  if (/\bCSS\b/.test(text)) return "lyra";
  return undefined;
};

/** @returns the prefix of a file that builds its own helpers with `newBEM`. */
const ownPrefix = (text: string): string | undefined => {
  const call = /(?<!\.)newBEM\(\s*(?:"([\w-]+)"|(\w+))\s*\)/.exec(text);
  if (call == null) return undefined;
  if (call[1] != null) return call[1];
  return new RegExp(`const ${call[2]} = "([\\w-]+)"`).exec(text)?.[1];
};

/** @returns the keys of an object literal's body, shorthand keys included. */
const objectKeys = (body: string): string[] =>
  splitTopLevel(body).flatMap((entry) => OBJECT_KEY.exec(entry)?.[1] ?? []);

/** @returns the body of the object literal assigned to a const in the text. */
const objectBody = (text: string, identifier: string): string => {
  const declaration = new RegExp(`const ${identifier}\\b[^=]*=\\s*\\{`).exec(text);
  if (declaration == null) return "";
  return span(text, declaration.index + declaration[0].length - 1) ?? "";
};

interface Emitted {
  classes: Name[];
  properties: Name[];
  literals: Set<string>;
}

const scanHelpers = (
  prefix: string,
  calls: RegExp,
  text: string,
  out: Emitted,
): void => {
  for (const m of text.matchAll(calls)) {
    const args = span(text, m.index + m[0].length - 1);
    if (args == null) continue;
    const name = helperName(prefix, m[1], splitTopLevel(args));
    if (name == null) continue;
    if (m[1] === "variable") out.properties.push(name);
    else out.classes.push(name);
  }
};

/**
 * @returns the classes that each named export of a file that calls newBEM builds, such
 * as Lyra's `CSS.selected`. A file that calls the export emits those classes too.
 */
const namedHelpers = (text: string): Array<[string, Name[]]> => {
  const prefix = ownPrefix(text);
  if (prefix == null) return [];
  const exports = [...text.matchAll(NAMED_EXPORT)];
  return exports.map((m, i) => {
    const out: Emitted = { classes: [], properties: [], literals: new Set() };
    scanHelpers(prefix, HELPER, text.slice(m.index, exports[i + 1]?.index), out);
    return [m[1], out.classes];
  });
};

const scanCode = (
  pkg: Package,
  text: string,
  helpers: Map<string, Name[]>,
  out: Emitted,
): void => {
  const own = ownPrefix(text);
  const prefix = own ?? helperPrefix(pkg, text);
  if (prefix != null)
    scanHelpers(prefix, own != null ? HELPER : QUALIFIED_HELPER, text, out);
  for (const m of text.matchAll(MEMBER)) out.classes.push(...(helpers.get(m[1]) ?? []));
  for (const m of text.matchAll(CLASS_LITERAL)) {
    // A literal that ends in a separator, such as "lyra-theme-", names a family.
    const family = m[2] == null && /[-_]$/.test(m[1]);
    const tail = m[2] != null ? DYNAMIC : family ? "[\\w-]+" : "";
    out.classes.push(toName(escape(m[1]) + tail, tail === ""));
  }
  for (const m of text.matchAll(PROPERTY_LITERAL))
    if (m[1] != null || m[4] != null)
      out.properties.push(
        toName(escape(m[2]) + (m[3] != null ? DYNAMIC : ""), m[3] == null),
      );
  for (const m of text.matchAll(STRING_LITERAL)) out.literals.add(m[2]);
};

const scanStyle = (text: string, out: Emitted): void => {
  for (const re of [PROPERTY_DECLARATION, PROPERTY_REGISTRATION])
    for (const m of text.matchAll(re)) out.properties.push({ exact: m[1] });
  for (const m of text.matchAll(DEFINE_VARS)) {
    const body = m[1].startsWith("{") ? m[1].slice(1, -1) : objectBody(text, m[1]);
    for (const key of objectKeys(body)) out.properties.push({ exact: `--${key}` });
  }
};

const walk = (dir: string): string[] => {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    if (SKIPPED_DIRECTORIES.has(entry.name)) return [];
    const full = path.join(dir, entry.name);
    return entry.isDirectory() ? walk(full) : [full];
  });
};

/** A file of a package, with its text as code, as a stylesheet, or as both. */
interface File {
  pkg: Package;
  code?: string;
  style?: string;
}

const packageFiles = (root: string, pkg: Package): File[] => {
  const dir = path.join(root, pkg);
  return [...walk(path.join(dir, "src")), path.join(dir, "index.html")].flatMap(
    (file) => {
      const isCode = CODE.test(file) && !TEST.test(file);
      const isStyle = STYLE.test(file);
      if ((!isCode && !isStyle) || !fs.existsSync(file)) return [];
      const text = fs.readFileSync(file, "utf-8");
      return [
        {
          pkg,
          code: isCode ? text.replace(STYLE_BLOCK, "") : undefined,
          style: isStyle ? text : undefined,
        },
      ];
    },
  );
};

/** The names one package emits. */
interface Names {
  exact: Set<string>;
  patterns: RegExp[];
}

const toNames = (names: Name[]): Names => ({
  exact: new Set(names.flatMap((n) => n.exact ?? [])),
  patterns: names.flatMap((n) => n.pattern ?? []),
});

/**
 * Reads the code and stylesheets of every package under root. The result reflects the
 * files at the time of the call.
 */
export const read = (root: string): Source => {
  const files = PACKAGES.flatMap((pkg) => packageFiles(root, pkg));
  const helpers = new Map(
    files.flatMap((f) => (f.code != null ? namedHelpers(f.code) : [])),
  );
  const classes = new Map<Package, Names>();
  const properties: Name[] = [];
  // Values cross packages: Lyra's Field builds a class from a path the Console passes.
  const literals = new Set<string>();
  for (const pkg of PACKAGES) {
    const emitted: Emitted = { classes: [], properties, literals };
    for (const file of files) {
      if (file.pkg !== pkg) continue;
      if (file.code != null) scanCode(pkg, file.code, helpers, emitted);
      if (file.style != null) scanStyle(file.style, emitted);
    }
    classes.set(pkg, toNames(emitted.classes));
  }
  const allProperties = toNames(properties);
  const matches = ({ exact, patterns }: Names, value: string): boolean =>
    exact.has(value) ||
    patterns.some((p) => {
      const m = p.exec(value);
      return m?.slice(1).every((g) => NUMBER.test(g) || literals.has(g)) === true;
    });
  return {
    packageOf: (file) => {
      const relative = path.relative(root, file).split(path.sep).join("/");
      return PACKAGES.find((pkg) => relative.startsWith(`${pkg}/`));
    },
    owners: (cls) => new Set(PACKAGES.filter((pkg) => matches(classes.get(pkg)!, cls))),
    defined: (property) => matches(allProperties, property),
  };
};
