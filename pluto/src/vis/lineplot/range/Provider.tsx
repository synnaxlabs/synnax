import { Aether } from "@/aether";
import { range } from "@/vis/lineplot/range/aether";
import { z } from "zod";

interface ProviderProps extends z.input<typeof range.providerStateZ> {}

export const Provider = Aether.wrap<ProviderProps>(
  "Annotation.Provider",
  ({ aetherKey, ...props }): null => {
    Aether.use({
      aetherKey,
      type: range.Provider.TYPE,
      schema: range.providerStateZ,
      initialState: props,
    });
    // const gridStyle = useGridEntry(
    //   {
    //     key: aetherKey,
    //     loc: "top",
    //     size: 12,
    //     order: "last",
    //   },
    //   "Annotation.Provider",
    // );
    return null;
  },
);
