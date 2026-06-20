// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc as PlutoArc } from "@synnaxlabs/pluto";
import { useEffect, useMemo, useRef } from "react";

import { Controls } from "@/arc/editor/Controls";
import { EXTENSIONS } from "@/arc/editor/text/placeholderSuggest";
import { useSelect } from "@/arc/selectors";
import { Editor as BaseEditor, type EditorExtension } from "@/code/Editor";
import { type Monaco } from "@/code/Provider";
import { type Layout } from "@/layout";

// utf16Offset converts a code-point index into the UTF-16 offset Monaco addresses with.
const utf16Offset = (s: string, codePointIndex: number): number =>
  Array.from(s).slice(0, codePointIndex).join("").length;

// applyRemote reflects a new document value into the Monaco model as a minimal edit so
// the local cursor and selection are preserved across remote edits. It is a no-op when
// the model already holds the value, so reflecting the store's own optimistic update does
// not disturb the editor.
const applyRemote = (model: Monaco.editor.ITextModel, next: string): void => {
  const old = model.getValue();
  if (old === next) return;
  const d = PlutoArc.diff(old, next);
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

// noOp satisfies the base editor's required onChange. The collaborative binding drives the
// model through the extension and the Flux store instead of the whole-value onChange path.
const noOp = (): void => {};

export const Editor: Layout.Renderer = ({ layoutKey }) => {
  const state = useSelect(layoutKey);
  // Ensure the arc is loaded into the Flux store, then track its replicated document.
  PlutoArc.useRetrieve({ key: layoutKey }, { addStatusOnFailure: false });
  const doc = PlutoArc.useSelectTextDoc({ key: layoutKey });
  const { dispatch } = PlutoArc.useDispatch();

  // text is the working CRDT replica: it generates operations for local edits and
  // materializes the document for the editor. editorRef exposes the model to the
  // store-sync effect, and applyingRemote guards the local handler against re-dispatching
  // the edits that store-sync makes.
  const textRef = useRef<PlutoArc.CollabText | null>(null);
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
  const applyingRemote = useRef(false);
  if (textRef.current == null && doc != null)
    textRef.current = PlutoArc.CollabText.bootstrap(doc);

  useEffect(() => {
    const text = textRef.current;
    if (text == null || doc == null) return;
    text.sync(doc);
    const model = editorRef.current?.getModel();
    if (model == null) return;
    applyingRemote.current = true;
    try {
      applyRemote(model, text.value());
    } finally {
      applyingRemote.current = false;
    }
  }, [doc]);

  const extensions = useMemo(() => {
    const collab: EditorExtension = (editor) => {
      editorRef.current = editor;
      const content = editor.onDidChangeModelContent((e) => {
        const text = textRef.current;
        if (text == null || applyingRemote.current) return;
        const actions = text.applyChanges(
          PlutoArc.changesToDiffs(text.value(), e.changes),
        );
        if (actions.length > 0) void dispatch({ key: layoutKey, actions });
      });
      return {
        dispose: () => {
          content.dispose();
          editorRef.current = null;
        },
      };
    };
    return [...EXTENSIONS, collab];
  }, [dispatch, layoutKey]);

  if (textRef.current == null) return null;
  return (
    <>
      <BaseEditor
        value={textRef.current.value()}
        onChange={noOp}
        language="arc"
        scrollBeyondLastLine
        extensions={extensions}
      />
      <Controls state={state} />
    </>
  );
};
