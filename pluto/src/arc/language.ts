// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { grammarRaw, languageConfigurationRaw, tokens } from "@synnaxlabs/arc";
import { type Synnax } from "@synnaxlabs/client";

import { EXTENSIONS } from "@/arc/text/placeholderSuggest";
import { type Code } from "@/code";

/** NAME is the Arc language id used by Monaco models and the LSP document selector. */
export const NAME = "arc";

const SCOPE_NAME = "source.arc";

type Theme = "dark" | "light";

const semanticTokenColors = (theme: Theme): Record<string, string> =>
  Object.fromEntries(Object.entries(tokens).map(([key, value]) => [key, value[theme]]));

const textMateRules = (theme: Theme): Code.TextMateRule[] =>
  Object.values(tokens).flatMap((config) =>
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
