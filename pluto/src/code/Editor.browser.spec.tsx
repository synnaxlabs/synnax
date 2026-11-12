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
  computed,
  renderThemed,
  resolve,
  selectedByTripleClick,
} from "@synnaxlabs/lyra/styletest";
import { describe, expect, it, vi } from "vitest";
import { userEvent } from "vitest/browser";

import { Code } from "@/code";
import { CSS } from "@/css";

const CODE = "const pressure = 1;";

// Monaco loads its whole runtime on the first editor of the page.
const MONACO_TIMEOUT = 60_000;

/** @returns a real editor once Monaco has loaded. */
const renderEditor = async (): Promise<Code.Monaco.editor.IStandaloneCodeEditor> => {
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
  if (editor == null) throw new Error("Monaco did not load");
  return editor;
};

/** @returns the marker row of the hover that a real editor opens on a warning. */
const hoverWarning = async (code?: string): Promise<Element> => {
  const editor = await renderEditor();
  const model = editor.getModel();
  if (model == null) throw new Error("editor has no model");
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

  it(
    "should write Monaco's errors to the console but not its warnings",
    async () => {
      await renderEditor();
      const { getService, ILogService } = await import("@codingame/monaco-vscode-api");
      const log = await getService(ILogService);
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      const error = vi.spyOn(console, "error").mockImplementation(() => {});
      log.warn("slow startup step");
      log.error("failed startup step");
      expect(warn).not.toHaveBeenCalled();
      expect(error).toHaveBeenCalledOnce();
    },
    MONACO_TIMEOUT,
  );

  describe("hover severity", () => {
    it(
      "should color the hover of a warning that has a code as a warning",
      async () => expectColors(await hoverWarning("ARC2005"), "warning"),
      MONACO_TIMEOUT,
    );

    it(
      "should color the hover of a warning that has no code as an error",
      async () => expectColors(await hoverWarning(), "error"),
      MONACO_TIMEOUT,
    );
  });
});
