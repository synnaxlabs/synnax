import * as monaco from "@codingame/monaco-vscode-editor-api";

import { type Theming } from "@/theming";

export const THEME_NAME = "vs-dark-custom";

export const defineTheme = (theme: Theming.Theme) => {
  const isDark = theme.key === "synnaxDark";
  monaco.editor.defineTheme(THEME_NAME, {
    base: isDark ? "vs-dark" : "vs",
    inherit: true,
    rules: [
      { foreground: "#cc255f", token: "keyword" },
      {
        token: "delimiter.bracket",
        foreground: theme.colors.gray.l9.hex,
        background: theme.colors.gray.l9.hex,
      },
      {
        token: "delimiter.parenthesis",
        foreground: "#cc255f",
        background: "#cc255f",
      },
      {
        token: "number",
        foreground: theme.colors.secondary.m1.hex,
        background: theme.colors.secondary.m1.hex,
      },
    ],
    colors: {
      "editor.background": theme.colors.gray.l1.hex,
      "editor.foreground": theme.colors.gray.l9.hex,
      "editor.selectionBackground": theme.colors.gray.l4.hex,
      "editor.lineHighlightBackground": theme.colors.gray.l3.hex,
      "editorCursor.foreground": theme.colors.primary.z.hex,
      "editorWhitespace.foreground": theme.colors.gray.l2.hex,
      "editorSuggestWidget.background": theme.colors.gray.l2.hex,
      "editorSuggestWidget.foreground": theme.colors.gray.l9.hex,
      "editorSuggestWidget.selectedBackground": theme.colors.gray.l3.hex,
      "editorSuggestWidget.selectedForeground": theme.colors.gray.l9.hex,
      "editorSuggestWidget.highlightForeground": theme.colors.primary.z.hex,
      "editorSuggestWidget.border": theme.colors.gray.l4.hex,
    },
  });
};
