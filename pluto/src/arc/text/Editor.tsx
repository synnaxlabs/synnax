// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useMemo } from "react";

import { NAME } from "@/arc/language";
import { type FluxSubStore, useSelectHasText, useSingleDispatch } from "@/arc/queries";
import { useKey } from "@/arc/Suspended";
import { changesToDiffs, CollabText, type TextChange } from "@/arc/text/collab";
import { Code } from "@/code";
import { Flux } from "@/flux";

export const Editor = () => {
  const resourceKey = useKey();
  const store = Flux.useStore<FluxSubStore>();
  const dispatch = useSingleDispatch();
  const hasText = useSelectHasText();

  // text is the working CRDT replica. It is bootstrapped once the document loads and lives
  // for the editor's lifetime, materializing the value and translating edits to operations.
  const text = useMemo<CollabText | null>(() => {
    const doc = store.arcs.get(resourceKey)?.text.doc;
    return doc != null ? CollabText.bootstrap(doc) : null;
  }, [store, resourceKey, hasText]);

  const handleEdit = useCallback(
    (edits: readonly TextChange[]) => {
      if (text == null) return;
      const actions = text.applyChanges(changesToDiffs(text.value(), edits));
      if (actions.length > 0) dispatch(actions);
    },
    [text, dispatch],
  );

  // connect subscribes the editor to operations from other editors: each store update for
  // this arc is merged into the replica and reflected into the editor as a minimal edit.
  // Returning the unsubscribe lets React tear the subscription down on unmount.
  const connect = useCallback(
    (handle: Code.EditorHandle | null) => {
      if (handle == null || text == null) return;
      return store.arcs.onSet((a) => {
        text.sync(a.text.doc);
        handle.setValue(text.value());
      }, resourceKey);
    },
    [store, text, resourceKey],
  );

  if (text == null) return null;
  return (
    <Code.Editor
      ref={connect}
      initialValue={text.value()}
      onEdit={handleEdit}
      language={NAME}
      scrollBeyondLastLine
    />
  );
};
