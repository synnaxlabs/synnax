import { Flex } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Controls, useLoadRemote } from "@/arc/editor/Editor";
import { useSelect } from "@/arc/selectors";
import { setRawText } from "@/arc/slice";
import { Editor } from "@/code/Editor";
import { type Layout } from "@/layout";

const Loaded: Layout.Renderer = ({ layoutKey }) => {
  const state = useSelect(layoutKey);
  const dispatch = useDispatch();
  const onChange = useCallback(
    (value: string) => {
      dispatch(setRawText({ key: layoutKey, raw: value }));
    },
    [layoutKey, dispatch],
  );
  return (
    <Flex.Box style={{ padding: 0, height: "100%", minHeight: 0 }} y empty>
      <Editor value={state.text.raw} onChange={onChange} language="arc" />;
      <Controls arc={state} />
    </Flex.Box>
  );
};

export const Text: Layout.Renderer = ({ layoutKey, ...rest }) => {
  const arc = useLoadRemote(layoutKey);
  if (arc == null) return null;
  return <Loaded layoutKey={layoutKey} {...rest} />;
};
