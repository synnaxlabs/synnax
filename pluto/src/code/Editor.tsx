// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/code/Editor.css";

import { type ILanguageFeaturesService } from "@codingame/monaco-vscode-api/services";
import { debounce, TimeSpan, url } from "@synnaxlabs/x";
import {
  type ReactNode,
  type Ref,
  type RefObject,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";

import { type EditorExtension } from "@/code/language";
import { type Monaco, useLanguage, useMonaco } from "@/code/Provider";
import { diff, utf16Offset } from "@/code/text";
import { CSS } from "@/css";
import { Errors } from "@/errors";
import { Flex } from "@/flex";
import { useSyncedRef } from "@/hooks";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { Theming } from "@/theming";
import { type Triggers } from "@/triggers";

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
  autoClosingQuotes: "always",
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

// applyMinimalEdit reflects next into the model as the single contiguous edit that turns
// its current value into it, so the local cursor and selection survive a programmatic set.
const applyMinimalEdit = (model: Monaco.editor.ITextModel, next: string): void => {
  const old = model.getValue();
  if (old === next) return;
  const d = diff(old, next);
  const start = model.getPositionAt(utf16Offset(old, d.index));
  const end = model.getPositionAt(utf16Offset(old, d.index + d.deleteCount));
  model.applyEdits([
    {
      range: {
        startLineNumber: start.lineNumber,
        startColumn: start.column,
        endLineNumber: end.lineNumber,
        endColumn: end.column,
      },
      text: d.insert,
    },
  ]);
};

/** EditorHandle is the imperative surface an Editor exposes through its ref, for consumers
 * that own the editor's text and update it programmatically rather than through React
 * state. */
export interface EditorHandle {
  /** setValue reconciles the editor's model to next as a minimal edit, preserving the
   * local cursor and selection. It is a no-op when the model already holds next, and does
   * not fire onChange, since the edit did not originate from the user. */
  setValue: (next: string) => void;
}

interface UseProps {
  initialValue?: string;
  onValueChange?: (value: string) => void;
  onEdit?: (changes: readonly Monaco.editor.IModelContentChange[]) => void;
  language: string;
  isBlock?: boolean;
  scrollBeyondLastLine?: boolean;
  openContextMenu?: Menu.ContextMenuProps["open"];
  extensions?: EditorExtension[];
}

const useTheme = (hasCustomTheme: boolean) => {
  const theme = Theming.use();
  const prefersDark = theme.key.includes("Dark");
  if (hasCustomTheme) return prefersDark ? "Default Dark+" : "Default Light+";
  return prefersDark ? "vs-dark" : "vs";
};

interface UseReturn {
  containerRef: RefObject<HTMLDivElement | null>;
  editorRef: RefObject<Monaco.editor.IStandaloneCodeEditor | null>;
  cursorRenameable: boolean;
  handle: EditorHandle;
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

// useRenameAvailable reports whether the editor's rename action would succeed at the
// current cursor position, re-probing the LSP rename providers (debounced) on each cursor
// move. It returns false while no editor or monaco instance is available.
const useRenameAvailable = (
  editor: Monaco.editor.IStandaloneCodeEditor | null,
  monaco: typeof Monaco | null,
): boolean => {
  const [renameable, setRenameable] = useState(false);
  useEffect(() => {
    if (editor == null || monaco == null) {
      setRenameable(false);
      return;
    }
    // Resolve the language features service once — it is stable across the editor's
    // lifetime. Checks scheduled before it resolves run once it lands.
    let features: ILanguageFeaturesService | null = null;
    const featuresPromise = import("@codingame/monaco-vscode-api/services").then(
      ({ getService, ILanguageFeaturesService }) =>
        getService(ILanguageFeaturesService),
    );
    featuresPromise
      .then((s) => (features = s))
      .catch((err: unknown) => {
        console.error("failed to resolve language features service", err);
      });

    let abort: AbortController | null = null;
    const run = () => {
      abort?.abort();
      const ctrl = new AbortController();
      abort = ctrl;
      const model = editor.getModel();
      const position = editor.getPosition();
      if (model == null || position == null) {
        setRenameable(false);
        return;
      }
      const exec = (svc: ILanguageFeaturesService) =>
        checkRenameAvailable(monaco, svc, model, position, ctrl.signal)
          .then((r) => {
            if (!ctrl.signal.aborted) setRenameable(r);
          })
          .catch(() => {
            if (!ctrl.signal.aborted) setRenameable(false);
          });
      if (features != null) void exec(features);
      else
        void featuresPromise.then((svc) => {
          if (!ctrl.signal.aborted) void exec(svc);
        });
    };
    const debounced = debounce(run, RENAME_CHECK_DEBOUNCE);
    const cursorDispose = editor.onDidChangeCursorPosition(debounced);
    run();
    return () => {
      debounced.cancel();
      abort?.abort();
      cursorDispose.dispose();
    };
  }, [editor, monaco]);
  return renameable;
};

const use = ({
  initialValue = "",
  onValueChange,
  onEdit,
  language,
  isBlock = false,
  scrollBeyondLastLine = false,
  openContextMenu,
  extensions,
}: UseProps): UseReturn => {
  const containerRef = useRef<HTMLDivElement>(null);
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
  const openContextMenuRef = useSyncedRef(openContextMenu);
  const onValueChangeRef = useSyncedRef(onValueChange);
  const onEditRef = useSyncedRef(onEdit);
  const suppressChange = useRef(false);
  const languageDef = useLanguage(language);
  const theme = useTheme(languageDef?.theme != null);
  const monaco = useMonaco();
  const resolvedExtensions = extensions ?? languageDef?.editorExtensions;
  const [editor, setEditor] = useState<Monaco.editor.IStandaloneCodeEditor | null>(
    null,
  );
  const cursorRenameable = useRenameAvailable(editor, monaco);

  const customURIRef = useRef<string | undefined>(undefined);
  if (customURIRef.current === undefined && isBlock) {
    const encoded = url.encodeBase64(JSON.stringify({ is_block: true }));
    const id = Math.random().toString(36).substring(7);
    customURIRef.current = `arc://block/${id}#${encoded}`;
  }
  const customURI = customURIRef.current;

  useEffect(() => {
    if (containerRef.current == null) return;
    const container = containerRef.current;

    // Create model with custom URI if this is a block
    let model: Monaco.editor.ITextModel | null = null;
    if (customURI != null) {
      const uri = monaco.Uri.parse(customURI);
      model = monaco.editor.createModel(initialValue, language, uri);
    }

    const editor = monaco.editor.create(container, {
      value: customURI != null ? undefined : initialValue,
      model: model ?? undefined,
      language: customURI != null ? undefined : language,
      theme,
      ...ZERO_OPTIONS,
      scrollBeyondLastLine,
    });
    editorRef.current = editor;
    setEditor(editor);

    disableMonacoCommandPalette(monaco);

    const contentDispose = editor.onDidChangeModelContent((e) => {
      if (suppressChange.current) return;
      onValueChangeRef.current?.(editor.getValue());
      onEditRef.current?.(e.changes);
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

    const extensionDisposables = resolvedExtensions?.map((ext) => ext(editor)) ?? [];

    return () => {
      contentDispose.dispose();
      triggerDispose.dispose();
      contextMenuDispose.dispose();
      extensionDisposables.forEach((d) => d.dispose());
      editor.dispose();
      model?.dispose();
      setEditor(null);
    };
  }, [monaco, customURI, resolvedExtensions]);

  useEffect(() => {
    monaco.editor.setTheme(theme);
  }, [monaco, theme]);

  const handle = useMemo<EditorHandle>(
    () => ({
      setValue: (next) => {
        const model = editorRef.current?.getModel();
        if (model == null) return;
        suppressChange.current = true;
        try {
          applyMinimalEdit(model, next);
        } finally {
          suppressChange.current = false;
        }
      },
    }),
    [],
  );

  return { containerRef, editorRef, cursorRenameable, handle };
};
export interface EditorProps
  extends Omit<Flex.BoxProps, "value" | "onChange" | "ref">, UseProps {
  ref?: Ref<EditorHandle>;
  loading?: ReactNode;
}

const MENU_EDITOR_ACTIONS: Record<string, string> = {
  cut: "editor.action.clipboardCutAction",
  copy: "editor.action.clipboardCopyAction",
  paste: "editor.action.clipboardPasteAction",
  rename: "editor.action.rename",
  format: "editor.action.formatDocument",
};

const EditorInternal = ({
  ref,
  initialValue,
  onEdit,
  onValueChange,
  className,
  language,
  isBlock,
  scrollBeyondLastLine,
  extensions,
  ...rest
}: Omit<EditorProps, "loading">) => {
  const { className: menuClassName, ...menuProps } = Menu.useContextMenu();
  const { containerRef, editorRef, cursorRenameable, handle } = use({
    initialValue,
    onEdit,
    onValueChange,
    language,
    isBlock,
    scrollBeyondLastLine,
    openContextMenu: menuProps.open,
    extensions,
  });
  useImperativeHandle(ref, () => handle, [handle]);

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
      <Menu.Menu level="small" gap="small">
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
            <Menu.Item
              itemKey="rename"
              trigger={RENAME_TRIGGER}
              triggerIndicator
              onClick={createMenuAction("rename")}
            >
              <Icon.Rename />
              Rename
            </Menu.Item>
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
      </Menu.Menu>
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

/** Editor is a Monaco-backed code editor. It suspends while the Monaco runtime
 * initializes on first use, rendering `loading` until ready, and surfaces an
 * initialization failure to the boundary it provides. */
export const Editor = ({ loading, ...props }: EditorProps) => (
  <Errors.SuspenseBoundary loading={loading}>
    <EditorInternal {...props} />
  </Errors.SuspenseBoundary>
);
