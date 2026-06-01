// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithConsole } from "@/testUtils";

type CursorListener = () => void;

interface FakeEditor {
  onKeyDown: ReturnType<typeof vi.fn>;
  onKeyUp: ReturnType<typeof vi.fn>;
  onContextMenu: ReturnType<typeof vi.fn>;
  onDidChangeModelContent: ReturnType<typeof vi.fn>;
  onDidChangeCursorPosition: ReturnType<typeof vi.fn>;
  getModel: () => unknown;
  getPosition: () => unknown;
  getValue: () => string;
  getSelection: () => null;
  focus: ReturnType<typeof vi.fn>;
  trigger: ReturnType<typeof vi.fn>;
  dispose: ReturnType<typeof vi.fn>;
  _emitCursor: () => void;
}

const cursorListeners: CursorListener[] = [];

const fakeModel = { uri: "file:///test.arc" };
const fakePosition = { lineNumber: 1, column: 1 };

const createFakeEditor = (): FakeEditor => ({
  onKeyDown: vi.fn(() => ({ dispose: vi.fn() })),
  onKeyUp: vi.fn(() => ({ dispose: vi.fn() })),
  onContextMenu: vi.fn(() => ({ dispose: vi.fn() })),
  onDidChangeModelContent: vi.fn(() => ({ dispose: vi.fn() })),
  onDidChangeCursorPosition: vi.fn((cb: CursorListener) => {
    cursorListeners.push(cb);
    return {
      dispose: () => {
        const idx = cursorListeners.indexOf(cb);
        if (idx >= 0) cursorListeners.splice(idx, 1);
      },
    };
  }),
  getModel: () => fakeModel,
  getPosition: () => fakePosition,
  getValue: () => "",
  getSelection: () => null,
  focus: vi.fn(),
  trigger: vi.fn(),
  dispose: vi.fn(),
  _emitCursor() {
    cursorListeners.forEach((cb) => cb());
  },
});

let currentEditor: FakeEditor | null = null;

const resolveRenameLocation = vi.fn(async () => ({ range: {}, text: "x" }));

class FakeCancellationTokenSource {
  token = {};
  cancel = vi.fn();
  dispose = vi.fn();
}

const fakeMonaco = {
  editor: {
    create: vi.fn(() => {
      currentEditor = createFakeEditor();
      return currentEditor;
    }),
    addKeybindingRule: vi.fn(),
    setTheme: vi.fn(),
    createModel: vi.fn(),
  },
  Uri: { parse: vi.fn() },
  KeyMod: { CtrlCmd: 1, Shift: 2 },
  KeyCode: { KeyP: 3 },
  CancellationTokenSource: FakeCancellationTokenSource,
};

vi.mock("@/code/Provider", () => ({
  useMonaco: () => fakeMonaco,
}));

vi.mock("@codingame/monaco-vscode-api/services", () => ({
  getService: vi.fn(async () => ({
    renameProvider: {
      ordered: () => [{ resolveRenameLocation }],
    },
  })),
  ILanguageFeaturesService: Symbol("ILanguageFeaturesService"),
}));

import { Editor } from "@/code/Editor";

const flushMicrotasks = async (): Promise<void> => {
  for (let i = 0; i < 8; i++) await Promise.resolve();
};

const DEBOUNCE_PADDING = TimeSpan.milliseconds(500);

describe("Editor", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    cursorListeners.length = 0;
    currentEditor = null;
    resolveRenameLocation.mockClear();
    fakeMonaco.editor.create.mockClear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("should cancel the pending rename check when unmounted before the debounce fires", async () => {
    const { unmount } = renderWithConsole(
      <Editor value="" onChange={() => {}} language="arc" />,
    );
    await act(async () => {
      await flushMicrotasks();
    });
    expect(currentEditor).not.toBeNull();
    resolveRenameLocation.mockClear();

    currentEditor!._emitCursor();
    unmount();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(DEBOUNCE_PADDING.milliseconds);
      await flushMicrotasks();
    });

    expect(resolveRenameLocation).not.toHaveBeenCalled();
    expect(currentEditor!.dispose).toHaveBeenCalledTimes(1);
  });

  it("should re-run the rename check after a cursor change while still mounted", async () => {
    renderWithConsole(<Editor value="" onChange={() => {}} language="arc" />);
    await act(async () => {
      await flushMicrotasks();
    });
    expect(currentEditor).not.toBeNull();
    resolveRenameLocation.mockClear();

    await act(async () => {
      currentEditor!._emitCursor();
      await vi.advanceTimersByTimeAsync(DEBOUNCE_PADDING.milliseconds);
      await flushMicrotasks();
    });

    expect(resolveRenameLocation).toHaveBeenCalled();
  });
});
