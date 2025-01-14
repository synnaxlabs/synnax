// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/code/Editor.css";

import { Align, type Input, Theming } from "@synnaxlabs/pluto";
import * as monaco from "monaco-editor";
import EditorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";
import { useEffect, useRef } from "react";

import { CSS } from "@/css";

export interface EditorProps
  extends Input.Control<string>,
    Omit<Align.SpaceProps, "value" | "onChange"> {}

export const Editor = ({ value, onChange, className, ...props }: EditorProps) => {
  const editorRef = useRef<HTMLDivElement | null>(null); // A ref to store the editor DOM element
  const monacoRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null); // A ref to store the Monaco editor instance
  const theme = Theming.use();

  useEffect(() => {
    if (editorRef.current === null) return;
    self.MonacoEnvironment = { getWorker: () => new EditorWorker() };

    const isDark = theme.key === "synnaxDark";

    monaco.editor.defineTheme("vs-dark-custom", {
      base: isDark ? "vs-dark" : "vs",
      inherit: true,
      rules: [
        {
          foreground: "#cc255f",
          token: "keyword",
        },
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
        "editor.background": theme.colors.gray.l2.hex,
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
    monacoRef.current = monaco.editor.create(editorRef.current, {
      value,
      language: "python",
      theme: "vs-dark-custom",
      automaticLayout: true,
      minimap: { enabled: false },
      bracketPairColorization: { enabled: false },
      lineNumbersMinChars: 3,
    });
    const dispose = monacoRef.current.onDidChangeModelContent(() => {
      if (monacoRef.current === null) return;
      onChange(monacoRef.current.getValue());
    });

    return () => {
      dispose.dispose();
      if (monacoRef.current) monacoRef.current.dispose();
    };
  }, [theme.key]);

  return (
    <Align.Space
      direction="y"
      grow
      {...props}
      className={CSS(className, CSS.B("editor"))}
    >
      <div ref={editorRef} style={{ height: "100%", position: "relative" }} />
    </Align.Space>
  );
};
