import type * as Monaco from "@codingame/monaco-vscode-editor-api";
import { type Synnax } from "@synnaxlabs/client";
import { type LSPStream } from "./lsp";
/** EditorExtension attaches language-specific behavior to a Monaco editor instance,
 * returning a disposable that detaches it when the editor is torn down. */
export type EditorExtension = (editor: Monaco.editor.IStandaloneCodeEditor) => Monaco.IDisposable;
/** TextMateRule colors a TextMate scope. It mirrors the shape VS Code's
 * `tokenColorCustomizations.textMateRules` setting expects. */
export interface TextMateRule {
    scope: string;
    settings: {
        foreground: string;
    };
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
/** BASE_THEMES are the settings ids of the VS Code themes that a language with a theme
 * builds on. VS Code renames these between releases, and a stale id silently drops every
 * token color, so `language.spec.ts` pins them against the installed theme extension. */
export declare const BASE_THEMES: {
    readonly dark: "Dark+";
    readonly light: "Light+";
};
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
    grammar: {
        scopeName: string;
        raw: string;
    };
    /** theme colors the language's tokens. Languages with a theme use the VS Code base
     * themes in BASE_THEMES; languages without one fall back to vs-dark/vs. */
    theme?: LanguageTheme;
    /** editorExtensions are applied to every editor opened for this language unless the
     * editor overrides them via its own `extensions` prop. */
    editorExtensions?: EditorExtension[];
    /** languageServer opens an LSP message stream against the cluster. When present, the
     * provider runs a reconnecting language client for the language. */
    languageServer?: (client: Synnax) => Promise<LSPStream>;
}
/** registerLanguage contributes a language to the running Monaco instance: it registers
 * the grammar and language configuration as a local extension and applies the token
 * theme. It must be called after monaco services have initialized. */
export declare const registerLanguage: (language: Language) => Promise<void>;
//# sourceMappingURL=language.d.ts.map