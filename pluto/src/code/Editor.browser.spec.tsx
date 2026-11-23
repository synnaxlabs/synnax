// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/code/Editor.css";

import { Menu } from "@synnaxlabs/lyra/menu";
import {
  computed,
  renderThemed,
  resolve,
  selectedByTripleClick,
} from "@synnaxlabs/lyra/styletest";
import { afterEach, describe, expect, it, vi } from "vitest";
import { type Locator, page, userEvent } from "vitest/browser";

import { Code } from "@/code";
import { CSS } from "@/css";

const CODE = "const pressure = 1;";

// Monaco loads its whole runtime on the first editor of the page.
const MONACO_TIMEOUT = 60_000;

/** Renders a real editor and resolves once Monaco has created it. */
const renderEditor = async (
  props: Partial<Code.EditorProps> = {},
): Promise<Code.Monaco.editor.IStandaloneCodeEditor> => {
  let editor: Code.Monaco.editor.IStandaloneCodeEditor | undefined;
  const capture: Code.EditorExtension = (e) => {
    editor = e;
    return { dispose: () => {} };
  };
  await renderThemed(
    <Code.Provider>
      <Code.Editor
        language="plaintext"
        initialValue={CODE}
        extensions={[capture]}
        style={{ width: 600, height: 300 }}
        {...props}
      />
    </Code.Provider>,
  );
  await expect.poll(() => editor, { timeout: MONACO_TIMEOUT }).toBeDefined();
  if (editor == null) throw new Error("editor was not created");
  return editor;
};

const codeLine = (): Element => {
  const line = document.querySelector(".monaco-editor .view-line > span");
  if (line == null) throw new Error("editor shows no code");
  return line;
};

const openMenu = async (): Promise<void> => {
  await expect
    .poll(() => document.querySelector(".monaco-editor .view-line > span"))
    .not.toBeNull();
  await userEvent.click(codeLine(), { button: "right" });
};

const item = (name: string): Locator => page.getByRole("menuitem", { name });

const SELECT_ALL = {
  startLineNumber: 1,
  startColumn: 1,
  endLineNumber: 1,
  endColumn: CODE.length + 1,
};

/** Registers a rename provider and re-checks rename at the cursor. */
const registerRename = async (
  editor: Code.Monaco.editor.IStandaloneCodeEditor,
  resolveRenameLocation: Code.Monaco.languages.RenameProvider["resolveRenameLocation"],
): Promise<Code.Monaco.IDisposable> => {
  const monaco = await import("@codingame/monaco-vscode-editor-api");
  const registration = monaco.languages.registerRenameProvider("plaintext", {
    provideRenameEdits: () => ({ edits: [] }),
    resolveRenameLocation,
  });
  editor.setPosition({ lineNumber: 1, column: 2 });
  return registration;
};

/** @returns the marker row of the hover that a real editor opens on a warning. */
const hoverWarning = async (code?: string): Promise<Element> => {
  let editor: Code.Monaco.editor.IStandaloneCodeEditor | undefined;
  const capture: Code.EditorExtension = (e) => {
    editor = e;
    return { dispose: () => {} };
  };
  await renderThemed(
    <Code.Provider>
      <Code.Editor
        language="plaintext"
        initialValue={CODE}
        extensions={[capture]}
        style={{ width: 600, height: 300 }}
      />
    </Code.Provider>,
  );
  await expect.poll(() => editor, { timeout: MONACO_TIMEOUT }).toBeDefined();
  const model = editor?.getModel();
  if (editor == null || model == null) throw new Error("editor has no model");
  const monaco = await import("@codingame/monaco-vscode-editor-api");
  monaco.editor.setModelMarkers(model, "spec", [
    {
      severity: monaco.MarkerSeverity.Warning,
      message: "experimental",
      source: "arc-analyzer",
      code,
      startLineNumber: 1,
      startColumn: 1,
      endLineNumber: 1,
      endColumn: CODE.length + 1,
    },
  ]);
  const text = () => document.querySelector(".monaco-editor .view-line > span");
  await expect.poll(text).not.toBeNull();
  await userEvent.hover(text() as Element);
  const row = () => document.querySelector(".monaco-hover .marker.hover-contents");
  await expect.poll(row, { timeout: 5000 }).not.toBeNull();
  return row() as Element;
};

const expectColors = (row: Element, token: string): void => {
  expect(computed(row, "background-color")).toEqual(
    resolve(`var(--pluto-${token}-z-10)`, "background-color"),
  );
  expect(computed(row, "border-top-color")).toEqual(
    resolve(`var(--pluto-${token}-z-70)`, "border-top-color"),
  );
};

describe("Editor", () => {
  // Monaco's parts that set no user-select of their own inherit it from the editor.
  it("should let a user select code under a body with selection off", async () => {
    const { getByText } = await renderThemed(
      <div className={CSS.B("editor")}>
        <div>{CODE}</div>
      </div>,
    );
    expect(await selectedByTripleClick(getByText(CODE).element())).toContain(CODE);
  });

  describe("hover severity", () => {
    it(
      "should color the hover of a warning that has a code as a warning",
      async () => expectColors(await hoverWarning("W2000"), "warning"),
      MONACO_TIMEOUT,
    );

    it(
      "should color the hover of a warning that has no code as an error",
      async () => expectColors(await hoverWarning(), "error"),
      MONACO_TIMEOUT,
    );
  });

  describe("context menu", () => {
    const disposables: Code.Monaco.IDisposable[] = [];

    afterEach(() => {
      disposables.splice(0).forEach((d) => d.dispose());
      vi.restoreAllMocks();
    });

    it(
      "should open a menu with the default actions on editor context menu",
      async () => {
        await renderEditor();
        await openMenu();
        await expect.element(item("Cut")).toBeInTheDocument();
        await expect.element(item("Copy")).toBeInTheDocument();
        await expect.element(item("Paste")).toBeInTheDocument();
        await expect.element(item("Format")).toBeInTheDocument();
      },
      MONACO_TIMEOUT,
    );

    it(
      "should append the consumer's extra items to the menu",
      async () => {
        await renderEditor({
          extraMenuItems: <Menu.Item itemKey="extra">Extra</Menu.Item>,
        });
        await openMenu();
        await expect.element(item("Extra")).toBeInTheDocument();
      },
      MONACO_TIMEOUT,
    );

    it.each([
      { label: "Cut", action: "editor.action.clipboardCutAction" },
      { label: "Copy", action: "editor.action.clipboardCopyAction" },
      { label: "Paste", action: "editor.action.clipboardPasteAction" },
      { label: "Format", action: "editor.action.formatDocument" },
    ])(
      "should focus the editor and trigger $label",
      async ({ label, action }) => {
        const editor = await renderEditor();
        editor.setSelection(SELECT_ALL);
        const focus = vi.spyOn(editor, "focus");
        const trigger = vi.spyOn(editor, "trigger");
        // A headless browser denies Paste the clipboard.
        vi.spyOn(navigator.clipboard, "readText").mockResolvedValue("");
        await openMenu();
        await item(label).click();
        expect(focus).toHaveBeenCalled();
        expect(trigger).toHaveBeenCalledWith("contextMenu", action, null);
      },
      MONACO_TIMEOUT,
    );

    it(
      "should disable cut and copy when there is no selection",
      async () => {
        const editor = await renderEditor();
        const trigger = vi.spyOn(editor, "trigger");
        await openMenu();
        await item("Cut").click({ force: true });
        expect(trigger).not.toHaveBeenCalled();
      },
      MONACO_TIMEOUT,
    );

    it(
      "should hide the rename action when the cursor is not renameable",
      async () => {
        await renderEditor();
        await openMenu();
        await expect.element(item("Paste")).toBeInTheDocument();
        await expect.element(item("Rename")).not.toBeInTheDocument();
      },
      MONACO_TIMEOUT,
    );

    it(
      "should hide the rename action and report when a provider fails",
      async () => {
        const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
        const editor = await renderEditor();
        disposables.push(
          await registerRename(editor, () => Promise.reject(new Error("boom"))),
        );
        await vi.waitFor(() =>
          expect(errorSpy).toHaveBeenCalledWith(
            "failed to check rename availability",
            expect.anything(),
          ),
        );
        await openMenu();
        await expect.element(item("Paste")).toBeInTheDocument();
        await expect.element(item("Rename")).not.toBeInTheDocument();
        errorSpy.mockRestore();
      },
      MONACO_TIMEOUT,
    );

    it(
      "should show and trigger rename when the cursor is renameable",
      async () => {
        const editor = await renderEditor();
        disposables.push(
          await registerRename(editor, () => ({
            range: SELECT_ALL,
            text: "const",
          })),
        );
        const trigger = vi.spyOn(editor, "trigger");
        await openMenu();
        await expect.element(item("Rename")).toBeInTheDocument();
        await item("Rename").click();
        expect(trigger).toHaveBeenCalledWith(
          "contextMenu",
          "editor.action.rename",
          null,
        );
      },
      MONACO_TIMEOUT,
    );
  });
});
