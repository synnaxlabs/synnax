import { Graph } from "@/arc/editor/graph";

export interface ToolbarProps {
  layoutKey: string;
}

export const Toolbar = ({ layoutKey }: ToolbarProps) => (
  <Graph.Toolbar layoutKey={layoutKey} />
);
