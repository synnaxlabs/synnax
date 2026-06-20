// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc as PlutoArc, Synnax } from "@synnaxlabs/pluto";
import { useEffect, useMemo, useState } from "react";

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
// the local cursor and selection are preserved across remote edits.
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

// noOp satisfies the base editor's required onChange. The collaborative binding drives
// the model through collabExtension instead of the whole-value onChange path.
const noOp = (): void => {};

// collabExtension binds the Monaco model to the collaborative session in both directions:
// local content changes are translated to precise CRDT operations and dispatched, and
// remote edits are applied to the model. applyingRemote guards the local handler so the
// edits applyRemote makes are not re-dispatched as local changes.
const collabExtension =
  (session: PlutoArc.CollabSession): EditorExtension =>
  (editor) => {
    let applyingRemote = false;
    const content = editor.onDidChangeModelContent((e) => {
      if (applyingRemote) return;
      session.applyChanges(PlutoArc.changesToDiffs(session.value(), e.changes));
    });
    const unsubscribe = session.subscribe((value) => {
      const model = editor.getModel();
      if (model == null) return;
      applyingRemote = true;
      try {
        applyRemote(model, value);
      } finally {
        applyingRemote = false;
      }
    });
    return {
      dispose: () => {
        content.dispose();
        unsubscribe();
      },
    };
  };

export const Editor: Layout.Renderer = ({ layoutKey }) => {
  const client = Synnax.use();
  const state = useSelect(layoutKey);
  const [session, setSession] = useState<PlutoArc.CollabSession | null>(null);
  useEffect(() => {
    if (client == null) return;
    let opened: PlutoArc.CollabSession | null = null;
    let cancelled = false;
    PlutoArc.CollabSession.open(client, layoutKey)
      .then((s) => {
        if (cancelled) {
          void s.close();
          return;
        }
        opened = s;
        setSession(s);
      })
      .catch((err: unknown) => {
        console.error("failed to open arc collaborative session", err);
      });
    return () => {
      cancelled = true;
      setSession(null);
      if (opened != null) void opened.close();
    };
  }, [client, layoutKey]);
  const extensions = useMemo(
    () => (session == null ? EXTENSIONS : [...EXTENSIONS, collabExtension(session)]),
    [session],
  );
  if (session == null) return null;
  return (
    <>
      <BaseEditor
        value={session.value()}
        onChange={noOp}
        language="arc"
        scrollBeyondLastLine
        extensions={extensions}
      />
      <Controls state={state} />
    </>
  );
};
