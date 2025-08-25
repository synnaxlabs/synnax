// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/code/Editor.css";

import { Flex, type Input, Theming, TimeSpan } from "@synnaxlabs/pluto";
import { type RefObject, useEffect, useRef } from "react";

import { type Monaco, useMonaco } from "@/code/Provider";
import { CSS } from "@/css";

const ZERO_OPTIONS: Monaco.editor.IEditorConstructionOptions = {
  automaticLayout: true,
  minimap: { enabled: false },
  bracketPairColorization: { enabled: false },
  lineNumbersMinChars: 3,
  folding: false,
  links: false,
  contextmenu: false,
  renderControlCharacters: false,
  renderWhitespace: "none",
  scrollBeyondLastLine: false,
  wordWrap: "off",
  renderLineHighlight: "none",
  formatOnPaste: false,
  formatOnType: true,
  suggestOnTriggerCharacters: false,
  showFoldingControls: "never",
};

const disableCommandPalette = (
  mon: Pick<typeof Monaco, "editor" | "KeyMod" | "KeyCode" | "KeyMod">,
) => {
  const CMD_ID = "ctrl-p";
  mon.editor.addKeybindingRule({
    keybinding: mon.KeyMod.CtrlCmd | mon.KeyCode.KeyP,
    command: CMD_ID,
  });
  mon.editor.addKeybindingRule({
    keybinding: mon.KeyMod.CtrlCmd | mon.KeyCode.KeyP | mon.KeyMod.Shift,
    command: CMD_ID,
  });
};

interface UseProps extends Input.Control<string> {
  language: string;
}

const useTheme = () => {
  const theme = Theming.use();
  const prefersDark = theme.key.includes("Dark");
  return prefersDark ? "vs-dark" : "vs";
};

const TRIGGER_SMALL_DELAY = TimeSpan.milliseconds(100).milliseconds;

/** @brief triggers a small model change to the editor so that it activates any language server features. */
const triggerSmallModelChangeToActiveLanguageServerFeatures = (
  editor: Monaco.editor.IStandaloneCodeEditor,
  value: string,
) => {
  setTimeout(() => {
    const model = editor.getModel();
    if (model != null)
      model.pushEditOperations(
        [],
        [{ range: model.getFullModelRange(), text: value }],
        () => null,
      );
  }, TRIGGER_SMALL_DELAY);
};

const use = ({
  value,
  onChange,
  language,
}: UseProps): RefObject<HTMLDivElement | null> => {
  const editorContainerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
  const theme = useTheme();
  const monaco = useMonaco();
  useEffect(() => {
    if (monaco == null || editorContainerRef.current == null) return;
    editorRef.current = monaco.editor.create(editorContainerRef.current, {
      value,
      language,
      theme,
      ...ZERO_OPTIONS,
    });

    // Trigger language features by making a temporary edit
    triggerSmallModelChangeToActiveLanguageServerFeatures(editorRef.current, value);

    disableCommandPalette(monaco);
    const dispose = editorRef.current.onDidChangeModelContent(() => {
      if (editorRef.current == null) return;
      onChange(editorRef.current.getValue());
    });
    return () => {
      dispose.dispose();
      if (editorRef.current != null) editorRef.current.dispose();
    };
  }, [theme, monaco]);
  return editorContainerRef;
};
export interface EditorProps
  extends Input.Control<string>,
    Omit<Flex.BoxProps, "value" | "onChange"> {
  language: string;
}

export const Editor = ({
  value,
  onChange,
  className,
  language,
  ...rest
}: EditorProps) => {
  const editorContainerRef = use({ value, onChange, language });
  return (
    <Flex.Box
      y
      grow
      {...rest}
      className={CSS(className, CSS.B("editor"))}
      style={{ height: "100%", position: "relative", overflow: "hidden" }}
    >
      <div ref={editorContainerRef} style={{ height: "100%" }} />
    </Flex.Box>
  );
};
