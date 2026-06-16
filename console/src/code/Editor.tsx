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
  getService,
  ILanguageFeaturesService,
} from "@codingame/monaco-vscode-api/services";
import {
  Flex,
  Icon,
  type Input,
  Menu,
  Theming,
  type Triggers,
} from "@synnaxlabs/pluto";
import { debounce, TimeSpan } from "@synnaxlabs/x";
import { type RefObject, useCallback, useEffect, useRef, useState } from "react";

import { type Monaco, useMonaco } from "@/code/Provider";
import { ContextMenu } from "@/components";
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
  suggestOnTriggerCharacters: true,
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

// Plug-in point for attaching language-specific behavior to a Monaco editor.
export type EditorExtension = (
  editor: Monaco.editor.IStandaloneCodeEditor,
) => Monaco.IDisposable;

interface UseProps extends Input.Control<string> {
  language: string;
  isBlock?: boolean;
  scrollBeyondLastLine?: boolean;
  openContextMenu?: Menu.ContextMenuProps["open"];
  extensions?: EditorExtension[];
}

const useTheme = (language: string) => {
  const theme = Theming.use();
  const prefersDark = theme.key.includes("Dark");
  if (language === "arc") return prefersDark ? "Default Dark+" : "Default Light+";
  return prefersDark ? "vs-dark" : "vs";
};

interface UseReturn {
  containerRef: RefObject<HTMLDivElement | null>;
  editorRef: RefObject<Monaco.editor.IStandaloneCodeEditor | null>;
  cursorRenameable: boolean;
}

// Query the registered rename providers for the model and ask each one
// whether the symbol at position can be renamed (LSP prepareRename). Any
// non-null, non-rejection result means the editor's rename action would
// succeed at that position. Returns false if no provider opts in.
const checkRenameAvailable = async (
  monaco: typeof Monaco,
  features: ILanguageFeaturesService,
  model: Monaco.editor.ITextModel,
  position: Monaco.Position,
  signal: AbortSignal,
): Promise<boolean> => {
  const providers = features.renameProvider.ordered(model);
  if (providers.length === 0) return false;
  const tokenSource = new monaco.CancellationTokenSource();
  signal.addEventListener("abort", () => tokenSource.cancel(), { once: true });
  try {
    for (const provider of providers) {
      if (provider.resolveRenameLocation == null) return true;
      const result = await provider.resolveRenameLocation(
        model,
        position,
        tokenSource.token,
      );
      if (signal.aborted) return false;
      if (result != null && result.rejectReason == null) return true;
    }
    return false;
  } finally {
    tokenSource.dispose();
  }
};

const RENAME_CHECK_DEBOUNCE = TimeSpan.milliseconds(80);

const use = ({
  value,
  onChange,
  language,
  isBlock = false,
  scrollBeyondLastLine = false,
  openContextMenu,
  extensions,
}: UseProps): UseReturn => {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
  const openContextMenuRef = useRef(openContextMenu);
  openContextMenuRef.current = openContextMenu;
  const theme = useTheme(language);
  const monaco = useMonaco();
  const [cursorRenameable, setCursorRenameable] = useState(false);

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

    const editor = monaco.editor.create(container, {
      value: customURI != null ? undefined : value,
      model: model ?? undefined,
      language: customURI != null ? undefined : language,
      theme,
      ...ZERO_OPTIONS,
      scrollBeyondLastLine,
    });
    editorRef.current = editor;

    disableMonacoCommandPalette(monaco);

    const contentDispose = editor.onDidChangeModelContent(() => {
      onChange(editor.getValue());
    });
    const triggerDispose = forwardGlobalTriggers(editor);
    const contextMenuDispose = editor.onContextMenu((e) =>
      openContextMenuRef.current?.({
        clientX: e.event.posx,
        clientY: e.event.posy,
        preventDefault: () => e.event.preventDefault(),
        stopPropagation: () => e.event.stopPropagation(),
        target: container,
      }),
    );

    // Resolve the language features service once per effect — it is stable
    // across the editor's lifetime, so re-resolving on each cursor move is
    // wasted work. checks scheduled before resolution buffer their request
    // via the latest-args debounce and run once the service lands.
    let features: ILanguageFeaturesService | null = null;
    const featuresPromise = getService(ILanguageFeaturesService);
    featuresPromise
      .then((s) => (features = s))
      .catch((err: unknown) => {
        console.error("failed to resolve language features service", err);
      });

    let renameCheckAbort: AbortController | null = null;
    const runRenameCheck = () => {
      renameCheckAbort?.abort();
      const ctrl = new AbortController();
      renameCheckAbort = ctrl;
      const m = editor.getModel();
      const position = editor.getPosition();
      if (m == null || position == null) {
        setCursorRenameable(false);
        return;
      }
      const exec = (svc: ILanguageFeaturesService) =>
        checkRenameAvailable(monaco, svc, m, position, ctrl.signal)
          .then((renameable) => {
            if (!ctrl.signal.aborted) setCursorRenameable(renameable);
          })
          .catch(() => {
            if (!ctrl.signal.aborted) setCursorRenameable(false);
          });
      if (features != null) void exec(features);
      else
        void featuresPromise.then((svc) => {
          if (!ctrl.signal.aborted) void exec(svc);
        });
    };
    const debouncedRenameCheck = debounce(runRenameCheck, RENAME_CHECK_DEBOUNCE);
    const cursorDispose = editor.onDidChangeCursorPosition(debouncedRenameCheck);
    runRenameCheck();

    const extensionDisposables = extensions?.map((ext) => ext(editor)) ?? [];

    return () => {
      contentDispose.dispose();
      triggerDispose.dispose();
      contextMenuDispose.dispose();
      cursorDispose.dispose();
      debouncedRenameCheck.cancel();
      renameCheckAbort?.abort();
      extensionDisposables.forEach((d) => d.dispose());
      editor.dispose();
      model?.dispose();
    };
  }, [monaco, customURI, extensions]);

  useEffect(() => {
    if (monaco == null) return;
    monaco.editor.setTheme(theme);
  }, [monaco, theme]);

  return { containerRef, editorRef, cursorRenameable };
};
export interface EditorProps
  extends Input.Control<string>, Omit<Flex.BoxProps, "value" | "onChange"> {
  language: string;
  isBlock?: boolean;
  scrollBeyondLastLine?: boolean;
  extensions?: EditorExtension[];
}

const MENU_EDITOR_ACTIONS: Record<string, string> = {
  cut: "editor.action.clipboardCutAction",
  copy: "editor.action.clipboardCopyAction",
  paste: "editor.action.clipboardPasteAction",
  rename: "editor.action.rename",
  format: "editor.action.formatDocument",
};

export const Editor = ({
  value,
  onChange,
  className,
  language,
  isBlock,
  scrollBeyondLastLine,
  extensions,
  ...rest
}: EditorProps) => {
  const { className: menuClassName, ...menuProps } = Menu.useContextMenu();
  const { containerRef, editorRef, cursorRenameable } = use({
    value,
    onChange,
    language,
    isBlock,
    scrollBeyondLastLine,
    openContextMenu: menuProps.open,
    extensions,
  });

  const createMenuAction = useCallback(
    (key: string) => () => {
      const editor = editorRef.current;
      if (editor == null) return;
      editor.focus();
      const action = MENU_EDITOR_ACTIONS[key];
      editor.trigger("contextMenu", action, null);
    },
    [],
  );

  const menuContent = useCallback(() => {
    const editor = editorRef.current;
    const selection = editor?.getSelection();
    const hasSelection =
      selection != null &&
      (selection.startLineNumber !== selection.endLineNumber ||
        selection.startColumn !== selection.endColumn);

    return (
      <ContextMenu.Menu>
        <Menu.Item
          itemKey="cut"
          trigger={CUT_TRIGGER}
          triggerIndicator
          disabled={!hasSelection}
          onClick={createMenuAction("cut")}
        >
          <Icon.Cut />
          Cut
        </Menu.Item>
        <Menu.Item
          itemKey="copy"
          trigger={COPY_TRIGGER}
          triggerIndicator
          disabled={!hasSelection}
          onClick={createMenuAction("copy")}
        >
          <Icon.Copy />
          Copy
        </Menu.Item>
        <Menu.Item
          itemKey="paste"
          trigger={PASTE_TRIGGER}
          triggerIndicator
          onClick={createMenuAction("paste")}
        >
          <Icon.Paste />
          Paste
        </Menu.Item>
        {cursorRenameable && (
          <>
            <Menu.Divider />
            <ContextMenu.RenameItem
              trigger={RENAME_TRIGGER}
              triggerIndicator
              onClick={createMenuAction("rename")}
            />
          </>
        )}
        <Menu.Divider />
        <Menu.Item
          itemKey="format"
          trigger={FORMAT_TRIGGER}
          triggerIndicator
          onClick={createMenuAction("format")}
        >
          <Icon.TextAlign.Left />
          Format
        </Menu.Item>
        <Menu.Divider />
        <ContextMenu.ReloadConsoleItem />
      </ContextMenu.Menu>
    );
  }, [createMenuAction, cursorRenameable]);

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
