// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { useEffect, useMemo, useRef } from "react";

import { useDispatch, useSelectTextDoc } from "@/arc/queries";
import { changesToDiffs, CollabText, diff } from "@/arc/text/collab";
import { EXTENSIONS } from "@/arc/text/placeholderSuggest";
import { Editor as BaseEditor, type EditorExtension } from "@/code/Editor";
import { type Monaco } from "@/code/Provider";

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

export interface EditorProps {
  resourceKey: arc.Key;
}

export const Editor = ({ resourceKey }: EditorProps) => {
  const doc = useSelectTextDoc({ key: resourceKey });
  const { dispatch } = useDispatch();

  // text is the working CRDT replica: it generates operations for local edits and
  // materializes the document for the editor. editorRef exposes the model to the
  // store-sync effect, and applyingRemote guards the local handler against re-dispatching
  // the edits that store-sync makes.
  const textRef = useRef<CollabText | null>(null);
  const editorRef = useRef<Monaco.editor.IStandaloneCodeEditor | null>(null);
  const applyingRemote = useRef(false);
  if (textRef.current == null && doc != null)
    textRef.current = CollabText.bootstrap(doc);

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
        const actions = text.applyChanges(changesToDiffs(text.value(), e.changes));
        if (actions.length > 0) void dispatch({ key: resourceKey, actions });
      });
      return {
        dispose: () => {
          content.dispose();
          editorRef.current = null;
        },
      };
    };
    return [...EXTENSIONS, collab];
  }, [dispatch, resourceKey]);

  if (textRef.current == null) return null;
  return (
    <BaseEditor
      value={textRef.current.value()}
      onChange={() => {}}
      language="arc"
      scrollBeyondLastLine
      extensions={extensions}
    />
  );
};
