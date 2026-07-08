// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, screen } from "@testing-library/react";
import { createRef } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Editor, type EditorHandle } from "@/code/Editor";
import { type EditorExtension, type Language } from "@/code/language";

// Mock the Provider module — the real one pulls in the monaco-vscode-api runtime, whose
// deep CSS imports cannot be evaluated under Node's ESM loader. We feed Editor a fake
// monaco that implements only the surface Editor drives (mirrors lsp.spec mocking the
// language client and handing the hook a trivial monaco).
const { useMonacoMock, useLanguageMock } = vi.hoisted(() => ({
  useMonacoMock: vi.fn(),
  useLanguageMock: vi.fn(),
}));
vi.mock("@/code/Provider", () => ({
  useMonaco: useMonacoMock,
  useLanguage: useLanguageMock,
}));

// useRenameAvailable dynamically imports this module to reach the LSP rename providers.
const { getServiceMock } = vi.hoisted(() => ({ getServiceMock: vi.fn() }));
vi.mock("@codingame/monaco-vscode-api/services", () => ({
  getService: getServiceMock,
  ILanguageFeaturesService: Symbol("ILanguageFeaturesService"),
}));

interface Disposable {
  dispose: () => void;
}

const makeEmitter = <T,>() => {
  const listeners = new Set<(e: T) => void>();
  const on = (cb: (e: T) => void): Disposable => {
    listeners.add(cb);
    return { dispose: () => listeners.delete(cb) };
  };
  const emit = (e: T): void => listeners.forEach((l) => l(e));
  return { on, emit, listeners };
};

const offsetToPosition = (text: string, offset: number) => {
  let lineNumber = 1;
  let column = 1;
  for (let i = 0; i < offset && i < text.length; i++)
    if (text[i] === "\n") {
      lineNumber++;
      column = 1;
    } else column++;
  return { lineNumber, column };
};

const positionToOffset = (text: string, line: number, column: number): number => {
  let offset = 0;
  let current = 1;
  while (current < line) {
    const nl = text.indexOf("\n", offset);
    if (nl === -1) break;
    offset = nl + 1;
    current++;
  }
  return offset + (column - 1);
};

interface Range {
  startLineNumber: number;
  startColumn: number;
  endLineNumber: number;
  endColumn: number;
}

interface Edit {
  range: Range;
  text: string;
}

// A model backed by a real string so Editor's offset/position math and minimal-edit
// reconciliation actually round-trip. notify forwards an applyEdits mutation to the owning
// editor's content emitter, as monaco's model→editor wiring does.
const createFakeModel = (initial: string, language: string, uri?: unknown) => {
  let value = initial;
  let notify: ((changes: unknown[]) => void) | null = null;
  const model = {
    uri,
    language,
    getValue: () => value,
    getPositionAt: (offset: number) => offsetToPosition(value, offset),
    applyEdits: vi.fn((edits: Edit[]) => {
      for (const edit of edits) {
        const start = positionToOffset(
          value,
          edit.range.startLineNumber,
          edit.range.startColumn,
        );
        const end = positionToOffset(
          value,
          edit.range.endLineNumber,
          edit.range.endColumn,
        );
        value = value.slice(0, start) + edit.text + value.slice(end);
      }
      notify?.(edits);
    }),
    dispose: vi.fn(),
    setValueDirect: (next: string) => {
      value = next;
    },
    _setNotify: (fn: (changes: unknown[]) => void) => {
      notify = fn;
    },
  };
  return model;
};

type FakeModel = ReturnType<typeof createFakeModel>;

const createFakeEditor = (model: FakeModel) => {
  const content = makeEmitter<{ changes: unknown[] }>();
  const keyDown = makeEmitter<{ browserEvent: KeyboardEvent }>();
  const keyUp = makeEmitter<{ browserEvent: KeyboardEvent }>();
  const contextMenu = makeEmitter<{
    event: {
      posx: number;
      posy: number;
      preventDefault: () => void;
      stopPropagation: () => void;
    };
  }>();
  const cursor = makeEmitter<unknown>();
  model._setNotify((changes) => content.emit({ changes }));
  let position = { lineNumber: 1, column: 1 };
  let selection: Range | null = null;
  const editor = {
    getModel: () => model,
    getValue: () => model.getValue(),
    getPosition: () => position,
    getSelection: () => selection,
    onDidChangeModelContent: content.on,
    onKeyDown: keyDown.on,
    onKeyUp: keyUp.on,
    onContextMenu: contextMenu.on,
    onDidChangeCursorPosition: cursor.on,
    focus: vi.fn(),
    trigger: vi.fn(),
    dispose: vi.fn(),
    setSelection: (s: Range | null) => {
      selection = s;
    },
    setPosition: (p: { lineNumber: number; column: number }) => {
      position = p;
    },
    userType: (next: string) => {
      model.setValueDirect(next);
      content.emit({ changes: [{ text: next }] });
    },
    emitKeyDown: keyDown.emit,
    emitKeyUp: keyUp.emit,
    emitContextMenu: contextMenu.emit,
    emitCursor: cursor.emit,
    listeners: { content: content.listeners, keyDown: keyDown.listeners },
  };
  return editor;
};

type FakeEditor = ReturnType<typeof createFakeEditor>;

const createFakeMonaco = () => {
  let lastEditor: FakeEditor | null = null;
  let lastCreatedModel: FakeModel | null = null;
  const editor = {
    create: vi.fn(
      (
        _container: HTMLElement,
        options: {
          value?: string;
          model?: FakeModel;
          language?: string;
          theme?: string;
          scrollBeyondLastLine?: boolean;
          [key: string]: unknown;
        },
      ) => {
        const model =
          options.model ?? createFakeModel(options.value ?? "", options.language ?? "");
        const ed = createFakeEditor(model);
        (ed as FakeEditor & { options: unknown }).options = options;
        lastEditor = ed;
        return ed;
      },
    ),
    createModel: vi.fn((value: string, language: string, uri: unknown) => {
      lastCreatedModel = createFakeModel(value, language, uri);
      return lastCreatedModel;
    }),
    setTheme: vi.fn(),
    addKeybindingRule: vi.fn(),
  };
  return {
    editor,
    Uri: { parse: vi.fn((raw: string) => ({ raw, toString: () => raw })) },
    KeyMod: { CtrlCmd: 2048, Shift: 1024 },
    KeyCode: { KeyP: 46 },
    CancellationTokenSource: class {
      token = { isCancellationRequested: false };
      cancel = vi.fn();
      dispose = vi.fn();
    },
    get editorInstance(): FakeEditor {
      if (lastEditor == null) throw new Error("editor not created");
      return lastEditor;
    },
    get createdModel(): FakeModel | null {
      return lastCreatedModel;
    },
  };
};

type FakeMonaco = ReturnType<typeof createFakeMonaco>;

// A rename provider that opts the cursor into being renameable.
const renameableProvider = {
  resolveRenameLocation: vi.fn().mockResolvedValue({ range: {}, text: "name" }),
};

const SELECTION: Range = {
  startLineNumber: 1,
  startColumn: 1,
  endLineNumber: 1,
  endColumn: 5,
};

const setRenameProviders = (providers: unknown[]): void => {
  getServiceMock.mockReturnValue({
    renameProvider: { ordered: () => providers },
  });
};

let monaco: FakeMonaco;

const renderEditor = (props: Partial<Parameters<typeof Editor>[0]> = {}) =>
  render(<Editor language="arc" {...props} />);

describe("Editor", () => {
  beforeEach(() => {
    monaco = createFakeMonaco();
    useMonacoMock.mockReturnValue(monaco);
    useLanguageMock.mockReturnValue(undefined);
    setRenameProviders([]);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe("initialization", () => {
    it("should create a monaco editor inside the container", () => {
      renderEditor({ initialValue: "hello" });
      expect(monaco.editor.create).toHaveBeenCalledTimes(1);
      const [container, options] = monaco.editor.create.mock.calls[0];
      expect(container).toBeInstanceOf(HTMLElement);
      expect(options).toMatchObject({
        value: "hello",
        language: "arc",
        automaticLayout: true,
        formatOnType: true,
      });
    });

    it("should disable the monaco command palette keybindings", () => {
      renderEditor();
      const bindings = monaco.editor.addKeybindingRule.mock.calls.map(
        (c) => c[0].keybinding,
      );
      // Ctrl+P and Ctrl+Shift+P both routed to a noop.
      expect(bindings).toContain(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyP);
      expect(bindings).toContain(
        monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyP | monaco.KeyMod.Shift,
      );
      monaco.editor.addKeybindingRule.mock.calls.forEach((c) =>
        expect(c[0].command).toEqual("noop"),
      );
    });

    it("should always auto-close quotes so f-string prefixes close", () => {
      renderEditor();
      expect(monaco.editor.create.mock.calls[0][1]).toMatchObject({
        autoClosingQuotes: "always",
      });
    });

    it("should default scrollBeyondLastLine to false", () => {
      renderEditor();
      expect(monaco.editor.create.mock.calls[0][1]).toMatchObject({
        scrollBeyondLastLine: false,
      });
    });

    it("should honor the scrollBeyondLastLine override", () => {
      renderEditor({ scrollBeyondLastLine: true });
      expect(monaco.editor.create.mock.calls[0][1]).toMatchObject({
        scrollBeyondLastLine: true,
      });
    });

    it("should create a regular model from value and language when not a block", () => {
      renderEditor({ initialValue: "x" });
      expect(monaco.editor.createModel).not.toHaveBeenCalled();
      const options = monaco.editor.create.mock.calls[0][1];
      expect(options.value).toEqual("x");
      expect(options.language).toEqual("arc");
      expect(options.model).toBeUndefined();
    });

    it("should create a block model with a custom arc:// uri when isBlock is set", () => {
      renderEditor({ initialValue: "y", isBlock: true });
      expect(monaco.Uri.parse).toHaveBeenCalledTimes(1);
      const uri = monaco.Uri.parse.mock.calls[0][0];
      expect(uri).toMatch(/^arc:\/\/block\//);
      expect(monaco.editor.createModel).toHaveBeenCalledWith(
        "y",
        "arc",
        expect.anything(),
      );
      const options = monaco.editor.create.mock.calls[0][1];
      expect(options.model).toBe(monaco.createdModel);
      expect(options.value).toBeUndefined();
      expect(options.language).toBeUndefined();
    });
  });

  describe("change propagation", () => {
    it("should invoke onValueChange and onEdit on a user edit", () => {
      const onValueChange = vi.fn();
      const onEdit = vi.fn();
      renderEditor({ initialValue: "a", onValueChange, onEdit });
      act(() => monaco.editorInstance.userType("ab"));
      expect(onValueChange).toHaveBeenCalledWith("ab");
      expect(onEdit).toHaveBeenCalledTimes(1);
      expect(onEdit.mock.calls[0][0]).toEqual([{ text: "ab" }]);
    });
  });

  describe("imperative setValue", () => {
    it("should reconcile the model as a single minimal edit", () => {
      const ref = createRef<EditorHandle>();
      renderEditor({ initialValue: "abc\ndef", ref });
      act(() => ref.current?.setValue("abc\nXef"));
      const model = monaco.editorInstance.getModel();
      expect(model.getValue()).toEqual("abc\nXef");
      expect(model.applyEdits).toHaveBeenCalledTimes(1);
      const edit = model.applyEdits.mock.calls[0][0][0];
      expect(edit.text).toEqual("X");
      expect(edit.range).toEqual({
        startLineNumber: 2,
        startColumn: 1,
        endLineNumber: 2,
        endColumn: 2,
      });
    });

    it("should be a no-op when the model already holds the value", () => {
      const ref = createRef<EditorHandle>();
      renderEditor({ initialValue: "same", ref });
      act(() => ref.current?.setValue("same"));
      expect(monaco.editorInstance.getModel().applyEdits).not.toHaveBeenCalled();
    });

    it("should not fire onValueChange or onEdit for a programmatic edit", () => {
      const onValueChange = vi.fn();
      const onEdit = vi.fn();
      const ref = createRef<EditorHandle>();
      renderEditor({ initialValue: "a", onValueChange, onEdit, ref });
      act(() => ref.current?.setValue("abc"));
      expect(monaco.editorInstance.getModel().getValue()).toEqual("abc");
      expect(onValueChange).not.toHaveBeenCalled();
      expect(onEdit).not.toHaveBeenCalled();
    });
  });

  describe("theme", () => {
    it("should use the plain monaco theme for a language without a custom theme", () => {
      renderEditor();
      expect(monaco.editor.create.mock.calls[0][1].theme).toEqual("vs");
      expect(monaco.editor.setTheme).toHaveBeenCalledWith("vs");
    });

    it("should use a VS Code base theme for a language with a custom theme", () => {
      useLanguageMock.mockReturnValue({
        name: "arc",
        configuration: "{}",
        grammar: { scopeName: "source.arc", raw: "{}" },
        theme: { semanticTokenColors: { dark: {}, light: {} } },
      } satisfies Language);
      renderEditor();
      expect(monaco.editor.create.mock.calls[0][1].theme).toEqual("Default Light+");
      expect(monaco.editor.setTheme).toHaveBeenCalledWith("Default Light+");
    });
  });

  describe("context menu", () => {
    const openMenu = (editor: FakeEditor) =>
      act(() =>
        editor.emitContextMenu({
          event: {
            posx: 10,
            posy: 20,
            preventDefault: vi.fn(),
            stopPropagation: vi.fn(),
          },
        }),
      );

    it("should open a menu with the default actions on editor context menu", () => {
      renderEditor();
      openMenu(monaco.editorInstance);
      expect(screen.getByText("Cut")).toBeTruthy();
      expect(screen.getByText("Copy")).toBeTruthy();
      expect(screen.getByText("Paste")).toBeTruthy();
      expect(screen.getByText("Format")).toBeTruthy();
    });

    it.each([
      { label: "Cut", action: "editor.action.clipboardCutAction" },
      { label: "Copy", action: "editor.action.clipboardCopyAction" },
      { label: "Paste", action: "editor.action.clipboardPasteAction" },
      { label: "Format", action: "editor.action.formatDocument" },
    ])("should focus the editor and trigger $label", ({ label, action }) => {
      renderEditor();
      const editor = monaco.editorInstance;
      editor.setSelection(SELECTION);
      openMenu(editor);
      fireEvent.click(screen.getByText(label));
      expect(editor.focus).toHaveBeenCalled();
      expect(editor.trigger).toHaveBeenCalledWith("contextMenu", action, null);
    });

    it("should disable cut and copy when there is no selection", () => {
      renderEditor();
      const editor = monaco.editorInstance;
      editor.setSelection(null);
      openMenu(editor);
      fireEvent.click(screen.getByText("Cut"));
      expect(editor.trigger).not.toHaveBeenCalled();
    });

    it("should hide the rename action when the cursor is not renameable", () => {
      setRenameProviders([]);
      renderEditor();
      openMenu(monaco.editorInstance);
      expect(screen.getByText("Paste")).toBeTruthy();
      expect(screen.queryByText("Rename")).toBeNull();
    });

    it("should show and trigger rename when the cursor is renameable", async () => {
      setRenameProviders([renameableProvider]);
      renderEditor();
      const editor = monaco.editorInstance;
      openMenu(editor);
      const rename = await screen.findByText("Rename");
      expect(rename).toBeTruthy();
      fireEvent.click(rename);
      expect(editor.trigger).toHaveBeenCalledWith(
        "contextMenu",
        "editor.action.rename",
        null,
      );
    });
  });

  describe("global trigger forwarding", () => {
    it("should redispatch modifier key events to the window", () => {
      renderEditor();
      const editor = monaco.editorInstance;
      const onWindowKeyDown = vi.fn();
      const onWindowKeyUp = vi.fn();
      window.addEventListener("keydown", onWindowKeyDown);
      window.addEventListener("keyup", onWindowKeyUp);
      editor.emitKeyDown({
        browserEvent: new KeyboardEvent("keydown", { ctrlKey: true, key: "p" }),
      });
      editor.emitKeyUp({
        browserEvent: new KeyboardEvent("keyup", { metaKey: true, key: "p" }),
      });
      expect(onWindowKeyDown).toHaveBeenCalledTimes(1);
      expect(onWindowKeyUp).toHaveBeenCalledTimes(1);
      window.removeEventListener("keydown", onWindowKeyDown);
      window.removeEventListener("keyup", onWindowKeyUp);
    });

    it("should not redispatch key events without a modifier", () => {
      renderEditor();
      const editor = monaco.editorInstance;
      const onWindowKeyDown = vi.fn();
      window.addEventListener("keydown", onWindowKeyDown);
      editor.emitKeyDown({
        browserEvent: new KeyboardEvent("keydown", { key: "a" }),
      });
      expect(onWindowKeyDown).not.toHaveBeenCalled();
      window.removeEventListener("keydown", onWindowKeyDown);
    });
  });

  describe("extensions", () => {
    it("should apply the editor extensions passed as a prop", () => {
      const dispose = vi.fn();
      const ext: EditorExtension = vi.fn(() => ({ dispose }));
      renderEditor({ extensions: [ext] });
      expect(ext).toHaveBeenCalledWith(monaco.editorInstance);
    });

    it("should fall back to the language's editorExtensions when no prop is given", () => {
      const langExt: EditorExtension = vi.fn(() => ({ dispose: vi.fn() }));
      useLanguageMock.mockReturnValue({
        name: "arc",
        configuration: "{}",
        grammar: { scopeName: "source.arc", raw: "{}" },
        editorExtensions: [langExt],
      } satisfies Language);
      renderEditor();
      expect(langExt).toHaveBeenCalledWith(monaco.editorInstance);
    });

    it("should prefer the prop extensions over the language's", () => {
      const propExt: EditorExtension = vi.fn(() => ({ dispose: vi.fn() }));
      const langExt: EditorExtension = vi.fn(() => ({ dispose: vi.fn() }));
      useLanguageMock.mockReturnValue({
        name: "arc",
        configuration: "{}",
        grammar: { scopeName: "source.arc", raw: "{}" },
        editorExtensions: [langExt],
      } satisfies Language);
      renderEditor({ extensions: [propExt] });
      expect(propExt).toHaveBeenCalledWith(monaco.editorInstance);
      expect(langExt).not.toHaveBeenCalled();
    });

    it("should dispose the extensions on unmount", () => {
      const dispose = vi.fn();
      const ext: EditorExtension = vi.fn(() => ({ dispose }));
      const { unmount } = renderEditor({ extensions: [ext] });
      unmount();
      expect(dispose).toHaveBeenCalledTimes(1);
    });
  });

  describe("teardown", () => {
    it("should dispose the editor and its listeners on unmount", () => {
      const { unmount } = renderEditor();
      const editor = monaco.editorInstance;
      expect(editor.listeners.content.size).toBeGreaterThan(0);
      unmount();
      expect(editor.dispose).toHaveBeenCalledTimes(1);
      expect(editor.listeners.content.size).toEqual(0);
      expect(editor.listeners.keyDown.size).toEqual(0);
    });

    it("should dispose the block model on unmount", () => {
      const { unmount } = renderEditor({ isBlock: true });
      const model = monaco.createdModel;
      unmount();
      expect(model?.dispose).toHaveBeenCalledTimes(1);
    });

    it("should stop propagating changes after the editor is torn down", () => {
      const onValueChange = vi.fn();
      const { unmount } = renderEditor({ onValueChange });
      const editor = monaco.editorInstance;
      unmount();
      act(() => editor.userType("after"));
      expect(onValueChange).not.toHaveBeenCalled();
    });
  });

  describe("loading", () => {
    it("should render the loading node while monaco is initializing", () => {
      const pending = new Promise<never>(() => {});
      useMonacoMock.mockImplementation(() => {
        // Suspense suspends on a thrown promise; this is how useMonaco signals it is
        // still initializing.
        // eslint-disable-next-line @typescript-eslint/only-throw-error
        throw pending;
      });
      renderEditor({ loading: <div>booting</div> });
      expect(screen.getByText("booting")).toBeTruthy();
    });
  });
});
