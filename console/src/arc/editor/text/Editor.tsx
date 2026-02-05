// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Controls } from "@/arc/editor/Controls";
import { useSelect } from "@/arc/selectors";
import { setRawText } from "@/arc/slice";
import { Editor as BaseEditor } from "@/code/Editor";
import { type Layout } from "@/layout";

export const Editor: Layout.Renderer = ({ layoutKey }) => {
  const state = useSelect(layoutKey);
  const dispatch = useDispatch();
  const onChange = useCallback(
    (value: string) => {
      dispatch(setRawText({ key: layoutKey, raw: value }));
    },
    [layoutKey, dispatch],
  );
  return (
    <>
      <BaseEditor
        value={state.text.raw}
        onChange={onChange}
        language="arc"
        scrollBeyondLastLine
      />
      <Controls state={state} />
    </>
  );
};
