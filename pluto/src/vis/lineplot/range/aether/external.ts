import { aether } from "@/aether/aether";
import { Annotation } from "@/vis/lineplot/range/aether/annotation";
import { Provider } from "@/vis/lineplot/range/aether/provider";

export * from "@/vis/lineplot/range/aether/annotation";
export * from "@/vis/lineplot/range/aether/provider";

export const REGISTRY: aether.ComponentRegistry = {
  [Annotation.TYPE]: Annotation,
  [Provider.TYPE]: Provider,
};
