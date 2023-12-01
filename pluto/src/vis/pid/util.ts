import { box } from "@synnaxlabs/x";
import { type ReactFlowInstance } from "reactflow";

export const selectNode = (key: string): HTMLDivElement => {
  const el = document.querySelector(`[data-id="${key}"]`);
  if (el == null) throw new Error(`[pid] - cannot find node with key: ${key}`);
  return el as HTMLDivElement;
};

export const selectNodeBox = (flow: ReactFlowInstance, key: string): box.Box => {
  const n = selectNode(key);
  const flowN = flow.getNodes().find((n) => n.id === key);
  if (flowN == null) throw new Error(`[pid] - cannot find node with key: ${key}`);
  return box.construct(flowN.position, box.dims(box.construct(n)));
};
