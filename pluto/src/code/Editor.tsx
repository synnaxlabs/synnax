// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/code/Editor.css";

import { LogLevel } from "@codingame/monaco-vscode-api";
import * as monaco from "@codingame/monaco-vscode-editor-api";
import { Align, type Input, Theming } from "@synnaxlabs/pluto";
import { configureDefaultWorkerFactory } from "monaco-editor-wrapper/workers/workerLoaders";
import { MonacoLanguageClient } from "monaco-languageclient";
import { ConsoleLogger } from "monaco-languageclient/tools";
import { initServices } from "monaco-languageclient/vscode/services";
import { useEffect, useRef } from "react";
import {
  CloseAction,
  ErrorAction,
  type MessageTransports,
} from "vscode-languageclient/browser";

import { defineTheme, THEME_NAME } from "@/code/theme";
import { CSS } from "@/css";
import { useAsyncEffect } from "@/hooks";

export interface EditorProps
  extends Input.Control<string>,
    Omit<Align.SpaceProps, "value" | "onChange"> {
  messageTransports?: MessageTransports;
}

const BASE_OPTIONS: monaco.editor.IStandaloneEditorConstructionOptions = {
  language: "lua",
  theme: THEME_NAME,
  // automaticLayout: true,
  // minimap: { enabled: false },
  // bracketPairColorization: { enabled: false },
  // lineNumbersMinChars: 3,
  // folding: false,
  // links: false,
  // contextmenu: false,
  // quickSuggestions: false,
  // renderControlCharacters: false,
  // renderWhitespace: "none",
  // scrollBeyondLastLine: false,
  // wordWrap: "off",
  // renderLineHighlight: "none",
  // formatOnPaste: false,
  // formatOnType: false,
  // suggestOnTriggerCharacters: false,
};

export const Editor = ({
  value,
  onChange,
  className,
  messageTransports,
  ...rest
}: EditorProps) => {
  const editorRef = useRef<HTMLDivElement | null>(null);
  const monacoRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const theme = Theming.use();

  useAsyncEffect(async () => {
    console.log("useAsyncEffect");
    if (editorRef.current === null || !messageTransports) return;

    const logger = new ConsoleLogger(LogLevel.Debug);
    configureDefaultWorkerFactory(logger);
    await initServices({}, { htmlContainer: editorRef.current, logger });

    defineTheme(theme);

    // Create the editor
    monacoRef.current = monaco.editor.create(editorRef.current, {
      ...BASE_OPTIONS,
      value,
    });

    // Create the language client
    const languageClient = new MonacoLanguageClient({
      name: "Lua Language Client",
      clientOptions: {
        documentSelector: ["lua"],
        errorHandler: {
          error: () => ({ action: ErrorAction.Continue }),
          closed: () => ({ action: CloseAction.DoNotRestart }),
        },
      },
      messageTransports,
    });

    // Start the language client
    languageClient.start();

    const dispose = monacoRef.current.onDidChangeModelContent(() => {
      if (monacoRef.current === null) return;
      onChange(monacoRef.current.getValue());
    });

    return () => {
      dispose.dispose();
      languageClient.dispose();
      if (monacoRef.current) monacoRef.current.dispose();
    };
  }, [theme.key, messageTransports]);

  return (
    <Align.Space
      direction="y"
      grow
      {...rest}
      className={CSS(className, CSS.B("editor"))}
    >
      <div ref={editorRef} style={{ height: "100%", position: "relative" }} />
    </Align.Space>
  );
};
