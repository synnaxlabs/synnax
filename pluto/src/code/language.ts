// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type * as Monaco from "@codingame/monaco-vscode-editor-api";
import { type Synnax } from "@synnaxlabs/client";

import { type LSPStream } from "@/code/lsp";

/** EditorExtension attaches language-specific behavior to a Monaco editor instance,
 * returning a disposable that detaches it when the editor is torn down. */
export type EditorExtension = (
  editor: Monaco.editor.IStandaloneCodeEditor,
) => Monaco.IDisposable;

/** TextMateRule colors a TextMate scope. It mirrors the shape VS Code's
 * `tokenColorCustomizations.textMateRules` setting expects. */
export interface TextMateRule {
  scope: string;
  settings: { foreground: string };
}

/** LanguageTheme declares how a language's tokens are colored, split by editor theme
 * (`dark`/`light`). `semanticTokenColors` maps semantic token types to colors;
 * `textMateRules` colors raw TextMate scopes for languages without a semantic provider. */
export interface LanguageTheme {
  semanticTokenColors?: {
    dark: Record<string, string>;
    light: Record<string, string>;
  };
  textMateRules?: {
    dark: TextMateRule[];
    light: TextMateRule[];
  };
}

/** Language is the declarative contribution a consumer passes to Code.Provider. It is
 * plain data plus an optional language-server opener: the provider owns every Monaco
 * interaction, so contributors never import monaco themselves. */
export interface Language {
  /** name is the language id used by Monaco models and the LSP document selector. */
  name: string;
  /** aliases are display names for the language. */
  aliases?: string[];
  /** extensions are the file extensions associated with the language (e.g. [".arc"]). */
  extensions?: string[];
  /** configuration is the raw language-configuration.json contents (comments, brackets,
   * folding, indentation rules). */
  configuration: string;
  /** grammar is the TextMate grammar: its top-level scope name and the raw
   * tmLanguage.json contents. */
  grammar: { scopeName: string; raw: string };
  /** theme colors the language's tokens. Languages with a theme use VS Code's
   * Default Dark+/Light+ base themes; languages without one fall back to vs-dark/vs. */
  theme?: LanguageTheme;
  /** editorExtensions are applied to every editor opened for this language unless the
   * editor overrides them via its own `extensions` prop. */
  editorExtensions?: EditorExtension[];
  /** languageServer opens an LSP message stream against the cluster. When present, the
   * provider runs a reconnecting language client for the language. */
  languageServer?: (client: Synnax) => Promise<LSPStream>;
}

const toDataURL = (raw: string): string => `data:application/json;base64,${btoa(raw)}`;

const applyTheme = async (theme: LanguageTheme): Promise<void> => {
  try {
    const vscode = await import("@codingame/monaco-vscode-extension-api");
    const config = vscode.workspace.getConfiguration("editor");
    if (theme.semanticTokenColors != null)
      await config.update(
        "semanticTokenColorCustomizations",
        {
          "[Default Dark+]": { rules: theme.semanticTokenColors.dark },
          "[Default Light+]": { rules: theme.semanticTokenColors.light },
        },
        vscode.ConfigurationTarget.Global,
      );
    if (theme.textMateRules != null)
      await config.update(
        "tokenColorCustomizations",
        {
          "[Default Dark+]": { textMateRules: theme.textMateRules.dark },
          "[Default Light+]": { textMateRules: theme.textMateRules.light },
        },
        vscode.ConfigurationTarget.Global,
      );
  } catch (err) {
    console.error("failed to apply language token colors", err);
  }
};

/** registerLanguage contributes a language to the running Monaco instance: it registers
 * the grammar and language configuration as a local extension and applies the token
 * theme. It must be called after monaco services have initialized. */
export const registerLanguage = async (language: Language): Promise<void> => {
  const { registerExtension, ExtensionHostKind } =
    await import("@codingame/monaco-vscode-api/extensions");
  const grammarPath = `./${language.name}.tmLanguage.json`;
  const configPath = `./${language.name}.language-configuration.json`;
  const { registerFileUrl } = registerExtension(
    {
      name: `${language.name}-language`,
      publisher: "synnaxlabs",
      version: "1.0.0",
      engines: { vscode: "*" },
      contributes: {
        languages: [
          {
            id: language.name,
            aliases: language.aliases,
            extensions: language.extensions,
            configuration: configPath,
          },
        ],
        grammars: [
          {
            language: language.name,
            scopeName: language.grammar.scopeName,
            path: grammarPath,
          },
        ],
      },
    },
    ExtensionHostKind.LocalProcess,
  );
  registerFileUrl(grammarPath, toDataURL(language.grammar.raw));
  registerFileUrl(configPath, toDataURL(language.configuration));
  if (language.theme != null) await applyTheme(language.theme);
};
