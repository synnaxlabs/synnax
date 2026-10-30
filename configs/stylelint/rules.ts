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
import { type Document, type Root, type Rule as Node } from "postcss";
import selectorParser from "postcss-selector-parser";
import valueParser from "postcss-value-parser";
import stylelint, { type Plugin, type PostcssResult, type Rule } from "stylelint";

import { BEM_CLASS, type Source } from "./source.ts";

const {
  createPlugin,
  utils: { report, ruleMessages, validateOptions },
} = stylelint;

/** Uses from before a rule existed. Entries only leave it. */
export interface Baseline {
  /** Classes a file may style although another package owns them, by file path. */
  foreignClasses: Record<string, string[]>;
  /** Astro files that may keep a global style block. */
  globalStyles: string[];
}

export interface PluginsConfig {
  /** The repository root that baseline paths are relative to. */
  root: string;
  source: Source;
  baseline: Baseline;
}

const UNKNOWN_CLASS = "synnax/no-unknown-class";
const UNKNOWN_PROPERTY = "synnax/no-unknown-custom-property";
const FOREIGN_CLASS = "synnax/no-foreign-class";
const GLOBAL_STYLE = "synnax/no-global-astro-style";

const STYLE_TAG = /<style\b([^>]*)>/g;
const LAYOUT = /\/src\/layouts\//;

/**
 * An Astro file parses into one root per style block or style attribute, and Stylelint
 * runs each rule once per root. A rule that judges the whole file runs on the first.
 */
const documentOf = (result: PostcssResult): Document | undefined =>
  result.root.type === "document" ? result.root : undefined;

const isFirstRoot = (root: Root, result: PostcssResult): boolean => {
  const document = documentOf(result);
  return document == null || document.first === root;
};

const walkClasses = (
  root: Root | Document,
  f: (cls: string, rule: Node) => void,
): void => {
  root.walkRules((rule) => {
    if (rule.parent?.type === "atrule" && /keyframes$/i.test(rule.parent.name)) return;
    selectorParser((s) => s.walkClasses((c) => f(c.value, rule))).processSync(
      rule.selector,
    );
  });
};

const unknownClass = (source: Source): Rule => {
  const messages = ruleMessages(UNKNOWN_CLASS, {
    rejected: (cls: string) => `Unexpected class "${cls}" that no code emits`,
  });
  const rule: Rule = (primary) => (root, result) => {
    if (!validateOptions(result, UNKNOWN_CLASS, { actual: primary, possible: [true] }))
      return;
    walkClasses(root, (cls, node) => {
      if (!BEM_CLASS.test(cls) || source.owners(cls).size > 0) return;
      report({
        ruleName: UNKNOWN_CLASS,
        result,
        node,
        word: cls,
        message: messages.rejected(cls),
      });
    });
  };
  rule.ruleName = UNKNOWN_CLASS;
  rule.messages = messages;
  return rule;
};

const unknownProperty = (source: Source): Rule => {
  const messages = ruleMessages(UNKNOWN_PROPERTY, {
    rejected: (property: string) =>
      `Unexpected custom property "${property}" that nothing defines`,
  });
  const rule: Rule = (primary) => (root, result) => {
    if (
      !validateOptions(result, UNKNOWN_PROPERTY, { actual: primary, possible: [true] })
    )
      return;
    root.walkDecls((node) => {
      // The parser skips strings, so content: "var(--x)" names no property.
      valueParser(node.value).walk((value) => {
        if (value.type !== "function" || value.value !== "var") return;
        const property = value.nodes.find((n) => n.type === "word")?.value;
        if (property == null || source.defined(property)) return;
        report({
          ruleName: UNKNOWN_PROPERTY,
          result,
          node,
          word: property,
          message: messages.rejected(property),
        });
      });
    });
  };
  rule.ruleName = UNKNOWN_PROPERTY;
  rule.messages = messages;
  return rule;
};

const relativeTo = (repo: string, file: string): string =>
  path.relative(repo, file).split(path.sep).join("/");

const foreignClass = ({ root: repo, source, baseline }: PluginsConfig): Rule => {
  const messages = ruleMessages(FOREIGN_CLASS, {
    rejected: (cls: string, owners: string) =>
      `Unexpected class "${cls}" that ${owners} owns. Pass your own class through ` +
      `className, or set a custom property that ${owners} documents.`,
    stale: (cls: string) =>
      `Unexpected baseline entry "${cls}" that this file no longer styles. Delete it ` +
      `from configs/stylelint/baseline.json.`,
  });
  const rule: Rule = (primary) => (root, result) => {
    if (!validateOptions(result, FOREIGN_CLASS, { actual: primary, possible: [true] }))
      return;
    const file = root.source?.input.file;
    if (file == null || !isFirstRoot(root, result)) return;
    const pkg = source.packageOf(file);
    if (pkg == null) return;
    const allowed = new Set(baseline.foreignClasses[relativeTo(repo, file)] ?? []);
    const used = new Set<string>();
    walkClasses(documentOf(result) ?? root, (cls, node) => {
      if (!BEM_CLASS.test(cls)) return;
      const owners = source.owners(cls);
      if (owners.size === 0 || owners.has(pkg)) return;
      if (allowed.has(cls)) {
        used.add(cls);
        return;
      }
      report({
        ruleName: FOREIGN_CLASS,
        result,
        node,
        word: cls,
        message: messages.rejected(cls, [...owners].join(" and ")),
      });
    });
    for (const cls of allowed)
      if (!used.has(cls))
        report({
          ruleName: FOREIGN_CLASS,
          result,
          node: root,
          message: messages.stale(cls),
        });
  };
  rule.ruleName = FOREIGN_CLASS;
  rule.messages = messages;
  return rule;
};

const globalStyle = ({ root: repo, baseline }: PluginsConfig): Rule => {
  const messages = ruleMessages(GLOBAL_STYLE, {
    rejected: () =>
      "Unexpected global style block outside a layout. Write a scoped <style>, and wrap " +
      "a selector for a child component's markup in :global().",
    stale: () =>
      "Unexpected baseline entry for a file with no global style block. Delete it " +
      "from configs/stylelint/baseline.json.",
  });
  const allowed = new Set(baseline.globalStyles);
  const rule: Rule = (primary) => (root, result) => {
    if (!validateOptions(result, GLOBAL_STYLE, { actual: primary, possible: [true] }))
      return;
    const file = root.source?.input.file;
    const document = documentOf(result);
    const text = document?.source?.input.css;
    if (file?.endsWith(".astro") !== true || text == null || !isFirstRoot(root, result))
      return;
    const relative = relativeTo(repo, file);
    if (LAYOUT.test(`/${relative}`)) return;
    const global = [...text.matchAll(STYLE_TAG)].filter((m) =>
      /\bis:global\b/.test(m[1]),
    );
    if (allowed.has(relative)) {
      if (global.length === 0)
        report({
          ruleName: GLOBAL_STYLE,
          result,
          node: root,
          message: messages.stale(),
        });
      return;
    }
    for (const tag of global) {
      const end = tag.index + tag[0].length;
      const node =
        document?.nodes.find((r) => (r.source?.start?.offset ?? -1) >= end) ?? root;
      report({ ruleName: GLOBAL_STYLE, result, node, message: messages.rejected() });
    }
  };
  rule.ruleName = GLOBAL_STYLE;
  rule.messages = messages;
  return rule;
};

/**
 * @returns the Stylelint plugins that check stylesheets against the code of every
 * package: a class must have an emitter, a custom property must have a definition, a
 * package must not style a class that only other packages emit, and an Astro component
 * must scope its styles.
 *
 * @throws if the baseline lists a file that does not exist.
 */
export const plugins = (config: PluginsConfig): Plugin[] => {
  const { root, baseline } = config;
  const missing = [...Object.keys(baseline.foreignClasses), ...baseline.globalStyles]
    .filter((file) => !fs.existsSync(path.join(root, file)))
    .sort();
  if (missing.length > 0)
    throw new Error(
      `configs/stylelint/baseline.json lists files that do not exist: ` +
        `${missing.join(", ")}. Delete their entries.`,
    );
  return [
    createPlugin(UNKNOWN_CLASS, unknownClass(config.source)),
    createPlugin(UNKNOWN_PROPERTY, unknownProperty(config.source)),
    createPlugin(FOREIGN_CLASS, foreignClass(config)),
    createPlugin(GLOBAL_STYLE, globalStyle(config)),
  ];
};
