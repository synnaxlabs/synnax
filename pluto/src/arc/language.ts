// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { grammarRaw, languageConfigurationRaw } from "@synnaxlabs/arc";
import { type Synnax } from "@synnaxlabs/client";

import { EXTENSIONS } from "@/arc/text/placeholderSuggest";
import { type Code } from "@/code";

/** NAME is the Arc language id used by Monaco models and the LSP document selector. */
export const NAME = "arc";

const SCOPE_NAME = "source.arc";

const TOKEN_CONFIG = {
  keyword: {
    dark: "#CC255F",
    light: "#CC255F",
    scopes: [
      "keyword.control.arc",
      "keyword.other.arc",
      "keyword.operator.logical.arc",
      "constant.language.boolean.arc",
      "constant.language.null.arc",
    ],
  },
  operator: {
    dark: "#dadada",
    light: "#292929",
    scopes: [
      "keyword.operator.arithmetic.arc",
      "keyword.operator.comparison.arc",
      "keyword.operator.assignment.arc",
      "keyword.operator.assignment.declare.arc",
      "keyword.operator.channel.arc",
    ],
  },
  statefulVariable: {
    dark: "#dadada",
    light: "#292929",
    scopes: ["keyword.operator.assignment.stateful.arc"],
  },
  edgeConditional: {
    dark: "#E06C75",
    light: "#BE3E4A",
    scopes: ["keyword.operator.transition.arc"],
  },
  edgeContinuous: {
    dark: "#56c8d8",
    light: "#0097A7",
    scopes: ["keyword.operator.flow.arc"],
  },
  string: {
    dark: "#98C379",
    light: "#0A7D00",
    scopes: ["string.quoted.arc"],
  },
  stringPlaceholder: {
    dark: "#CC255F",
    light: "#CC255F",
    scopes: [],
  },
  number: {
    dark: "#98C379",
    light: "#0A7D00",
    scopes: ["constant.numeric"],
  },
  type: {
    dark: "#4EC9B0",
    light: "#267F99",
    scopes: ["support.type.primitive.arc", "support.type.composite.arc"],
  },
  channel: {
    dark: "#61AFEF",
    light: "#0070C1",
    scopes: ["support.type.channel.arc"],
  },
  namespace: {
    dark: "#E5C07B",
    light: "#9C5404",
    scopes: [],
  },
  comment: {
    dark: "#5C6370",
    light: "#9DA5B4",
    scopes: ["comment"],
  },
  function: {
    dark: "#556bf8",
    light: "#3774D0",
    scopes: [
      "entity.name.function.arc",
      "support.function.builtin.arc",
      "support.function.arc",
      "storage.type.string.arc",
    ],
  },
  stage: {
    dark: "#dadada",
    light: "#292929",
    scopes: ["support.function.builtin.stage.arc", "entity.name.type.stage.arc"],
  },
  sequence: {
    dark: "#dadada",
    light: "#292929",
    scopes: ["entity.name.type.sequence.arc"],
  },
  variable: {
    dark: "#dadada",
    light: "#292929",
    scopes: ["variable.other.arc"],
  },
  channelVariable: {
    dark: "#61AFEF",
    light: "#0070C1",
    scopes: [],
  },
  block: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  parameter: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  input: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  output: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
  constant: {
    dark: "#dadada",
    light: "#292929",
    scopes: [],
  },
} as const;

/** SemanticTokenType enumerates the semantic token kinds the Arc analyzer emits. */
export type SemanticTokenType = keyof typeof TOKEN_CONFIG;

type Theme = "dark" | "light";

const semanticTokenColors = (theme: Theme): Record<string, string> =>
  Object.fromEntries(
    Object.entries(TOKEN_CONFIG).map(([key, value]) => [key, value[theme]]),
  );

const textMateRules = (theme: Theme): Code.TextMateRule[] =>
  Object.values(TOKEN_CONFIG).flatMap((config) =>
    config.scopes.map((scope) => ({
      scope,
      settings: { foreground: config[theme] },
    })),
  );

/** LANGUAGE is the Arc language definition consumed by Code.Provider: grammar and
 * configuration from @synnaxlabs/arc, token theming, editor extensions, and the LSP
 * stream opener. */
export const LANGUAGE: Code.Language = {
  name: NAME,
  aliases: ["Arc", "arc"],
  extensions: [".arc"],
  configuration: languageConfigurationRaw,
  grammar: { scopeName: SCOPE_NAME, raw: grammarRaw },
  theme: {
    semanticTokenColors: {
      dark: semanticTokenColors("dark"),
      light: semanticTokenColors("light"),
    },
    textMateRules: { dark: textMateRules("dark"), light: textMateRules("light") },
  },
  editorExtensions: EXTENSIONS,
  languageServer: (client: Synnax) => client.arcs.openLSP(),
};
