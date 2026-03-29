// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Raw JSON string for Monaco/VSCode extension registration
import arcGrammarRaw from "../../../go/lsp/extensions/vscode/syntaxes/arc.tmLanguage.json?raw";

/**
 * TextMate grammar for the Arc language.
 * This interface satisfies Shiki's LanguageRegistration requirements.
 *
 * The type uses index signatures for nested objects to avoid TypeScript's strict
 * type inference from JSON imports, which creates union types with optional undefined
 * values that are incompatible with Shiki's IRawCapturesMap type.
 */
export interface Grammar {
  name: string;
  scopeName: string;
  patterns: Array<Record<string, unknown>>;
  repository: Record<string, Record<string, unknown>>;
  [key: string]: unknown;
}

/**
 * Arc language grammar for syntax highlighting.
 * Use with Shiki or other TextMate-compatible syntax highlighters.
 *
 * @example
 * ```typescript
 * import { grammar } from '@synnaxlabs/arc';
 *
 * // Use with Shiki in Astro
 * export default defineConfig({
 *   markdown: {
 *     shikiConfig: {
 *       langs: [grammar],
 *     },
 *   },
 * });
 * ```
 */
// eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
export const grammar: Grammar = {
  ...JSON.parse(arcGrammarRaw),
  name: "arc",
  scopeName: "source.arc",
};

/**
 * Raw JSON string of the Arc grammar.
 * Use for Monaco/VSCode extension registration that requires string input.
 *
 * @example
 * ```typescript
 * import { grammarRaw } from '@synnaxlabs/arc';
 *
 * const dataUrl = `data:application/json;base64,${btoa(grammarRaw)}`;
 * ```
 */
export const grammarRaw: string = arcGrammarRaw;

export default grammar;
