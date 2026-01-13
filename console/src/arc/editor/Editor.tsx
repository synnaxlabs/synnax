import { Graph } from "@/arc/editor/graph";
import { useLoadRemote } from "@/arc/editor/hooks";
import { Text } from "@/arc/editor/text";
import { useSelectMode } from "@/arc/selectors";
import { type Layout } from "@/layout";

const Loaded: Layout.Renderer = (props) => {
  const { layoutKey } = props;
  const mode = useSelectMode(layoutKey);
  if (mode === "graph") return <Graph.Editor {...props} />;
  return <Text.Editor {...props} />;
};

export const Editor: Layout.Renderer = (props) => {
  const arc = useLoadRemote(props.layoutKey);
  if (arc == null) return null;
  return <Loaded {...props} />;
};
