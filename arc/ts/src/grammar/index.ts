// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// TextMate grammar definition for Arc language syntax highlighting
import arcGrammar from "../../../go/lsp/extensions/vscode/syntaxes/arc.tmLanguage.json" with { type: "json" };
// Raw JSON string for Monaco/VSCode extension registration
import arcGrammarRaw from "../../../go/lsp/extensions/vscode/syntaxes/arc.tmLanguage.json?raw";

/**
 * TextMate grammar for the Arc language.
 * Compatible with Shiki syntax highlighter and other TextMate-based tools.
 */
export interface Grammar {
  name: string;
  scopeName: string;
  patterns: unknown[];
  repository: Record<string, unknown>;
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
export const grammar = {
  ...arcGrammar,
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
