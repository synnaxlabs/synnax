// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/code/Editor.css";

import {
  Flex,
  Icon,
  type Input,
  Menu,
  Theming,
  TimeSpan,
  type Triggers,
} from "@synnaxlabs/pluto";
import { type RefObject, useCallback, useEffect, useRef } from "react";

import { type Monaco, useMonaco } from "@/code/Provider";
import { CSS } from "@/css";

const CUT_TRIGGER: Triggers.Trigger = ["Control", "X"];
const COPY_TRIGGER: Triggers.Trigger = ["Control", "C"];
const PASTE_TRIGGER: Triggers.Trigger = ["Control", "V"];
const RENAME_TRIGGER: Triggers.Trigger = ["F2"];
const FORMAT_TRIGGER: Triggers.Trigger = ["Shift", "Alt", "F"];

const ZERO_OPTIONS: Monaco.editor.IEditorConstructionOptions = {
  automaticLayout: true,
  minimap: { enabled: false },
  bracketPairColorization: { enabled: false },
  lineNumbersMinChars: 3,
  folding: true,
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
  showFoldingControls: "mouseover",
  hover: { above: false },
};

const disableMonacoCommandPalette = (
  mon: Pick<typeof Monaco, "editor" | "KeyMod" | "KeyCode">,
): void => {
  const NOOP_COMMAND = "noop";
  mon.editor.addKeybindingRule({
    keybinding: mon.KeyMod.CtrlCmd | mon.KeyCode.KeyP,
    command: NOOP_COMMAND,
  });
  mon.editor.addKeybindingRule({
    keybinding: mon.KeyMod.CtrlCmd | mon.KeyCode.KeyP | mon.KeyMod.Shift,
    command: NOOP_COMMAND,
  });
};

const hasGlobalModifier = (e: KeyboardEvent): boolean =>
  e.ctrlKey || e.metaKey || e.altKey;

const redispatchToWindow = (e: KeyboardEvent): void => {
  const synthetic = new KeyboardEvent(e.type, {
    key: e.key,
    code: e.code,
    ctrlKey: e.ctrlKey,
    shiftKey: e.shiftKey,
    altKey: e.altKey,
    metaKey: e.metaKey,
    bubbles: true,
  });
  window.dispatchEvent(synthetic);
};

const forwardGlobalTriggers = (
  editor: Monaco.editor.IStandaloneCodeEditor,
): Monaco.IDisposable => {
  const downDispose = editor.onKeyDown((e) => {
    if (hasGlobalModifier(e.browserEvent)) redispatchToWindow(e.browserEvent);
  });
  const upDispose = editor.onKeyUp((e) => {
    if (hasGlobalModifier(e.browserEvent)) redispatchToWindow(e.browserEvent);
  });
  return {
    dispose: () => {
      downDispose.dispose();
      upDispose.dispose();
    },
  };
};

interface UseProps extends Input.Control<string> {
  language: string;
  isBlock?: boolean;
  openContextMenu?: Menu.ContextMenuProps["open"];
}

const useTheme = (language: string) => {
  const theme = Theming.use();
  const prefersDark = theme.key.includes("Dark");
  if (language === "arc") return prefersDark ? "Default Dark+" : "Default Light+";
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

interface UseReturn {
  containerRef: RefObject<HTMLDivElement | null>;
  editorRef: RefObject<Monaco.editor.IStandaloneCodeEditor | null>;
}

const use = ({
  value,
  onChange,
  language,
  isBlock = false,
  openContextMenu,
}: UseProps): UseReturn => {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
  const openContextMenuRef = useRef(openContextMenu);
  openContextMenuRef.current = openContextMenu;
  const theme = useTheme(language);
  const monaco = useMonaco();

  const customURIRef = useRef<string | undefined>(undefined);
  if (customURIRef.current === undefined && isBlock) {
    const metadata = { is_block: true };
    const json = JSON.stringify(metadata);
    const encoded = btoa(json);
    const id = Math.random().toString(36).substring(7);
    const uri = `arc://block/${id}#${encoded}`;
    customURIRef.current = uri;
  }
  const customURI = customURIRef.current;

  useEffect(() => {
    if (monaco == null || containerRef.current == null) return;
    const container = containerRef.current;

    // Create model with custom URI if this is a block
    let model: Monaco.editor.ITextModel | null = null;
    if (customURI != null) {
      const uri = monaco.Uri.parse(customURI);
      model = monaco.editor.createModel(value, language, uri);
    }

    editorRef.current = monaco.editor.create(container, {
      value: customURI != null ? undefined : value,
      model: model ?? undefined,
      language: customURI != null ? undefined : language,
      theme,
      ...ZERO_OPTIONS,
    });

    triggerSmallModelChangeToActiveLanguageServerFeatures(editorRef.current, value);
    disableMonacoCommandPalette(monaco);

    const contentDispose = editorRef.current.onDidChangeModelContent(() => {
      if (editorRef.current == null) return;
      onChange(editorRef.current.getValue());
    });
    const triggerDispose = forwardGlobalTriggers(editorRef.current);
    const contextMenuDispose = editorRef.current.onContextMenu((e) =>
      openContextMenuRef.current?.({
        clientX: e.event.posx,
        clientY: e.event.posy,
        preventDefault: () => e.event.preventDefault(),
        stopPropagation: () => e.event.stopPropagation(),
        target: container,
      }),
    );

    return () => {
      contentDispose.dispose();
      triggerDispose.dispose();
      contextMenuDispose.dispose();
      editorRef.current?.dispose();
      model?.dispose();
    };
  }, [monaco, customURI]);

  useEffect(() => {
    if (monaco == null) return;
    monaco.editor.setTheme(theme);
  }, [monaco, theme]);

  return { containerRef, editorRef };
};
export interface EditorProps
  extends Input.Control<string>, Omit<Flex.BoxProps, "value" | "onChange"> {
  language: string;
  isBlock?: boolean;
}

export const Editor = ({
  value,
  onChange,
  className,
  language,
  isBlock,
  ...rest
}: EditorProps) => {
  const { className: menuClassName, ...menuProps } = Menu.useContextMenu();
  const { containerRef, editorRef } = use({
    value,
    onChange,
    language,
    isBlock,
    openContextMenu: menuProps.open,
  });

  const handleMenuSelect = useCallback((key: string) => {
    const editor = editorRef.current;
    if (editor == null) return;
    switch (key) {
      case "cut":
        editor.trigger("contextMenu", "editor.action.clipboardCutAction", null);
        break;
      case "copy":
        editor.trigger("contextMenu", "editor.action.clipboardCopyAction", null);
        break;
      case "paste":
        editor.trigger("contextMenu", "editor.action.clipboardPasteAction", null);
        break;
      case "rename":
        editor.trigger("contextMenu", "editor.action.rename", null);
        break;
      case "format":
        editor.trigger("contextMenu", "editor.action.formatDocument", null);
        break;
    }
  }, []);

  const menuContent = useCallback(
    () => (
      <Menu.Menu level="small" onChange={handleMenuSelect}>
        <Menu.Item itemKey="cut" trigger={CUT_TRIGGER} triggerIndicator>
          <Icon.Cut />
          Cut
        </Menu.Item>
        <Menu.Item itemKey="copy" trigger={COPY_TRIGGER} triggerIndicator>
          <Icon.Copy />
          Copy
        </Menu.Item>
        <Menu.Item itemKey="paste" trigger={PASTE_TRIGGER} triggerIndicator>
          <Icon.Paste />
          Paste
        </Menu.Item>
        <Menu.Divider />
        <Menu.Item itemKey="rename" trigger={RENAME_TRIGGER} triggerIndicator>
          <Icon.Rename />
          Rename
        </Menu.Item>
        <Menu.Item itemKey="format" trigger={FORMAT_TRIGGER} triggerIndicator>
          <Icon.TextAlign.Left />
          Format
        </Menu.Item>
      </Menu.Menu>
    ),
    [handleMenuSelect],
  );

  return (
    <Flex.Box y grow {...rest} className={CSS(className, CSS.B("editor"))}>
      <Menu.ContextMenu
        className={CSS(CSS.BE("editor", "context-menu"), className)}
        menu={menuContent}
        {...menuProps}
      >
        <Flex.Box ref={containerRef} full role="textbox" />
      </Menu.ContextMenu>
    </Flex.Box>
  );
};
