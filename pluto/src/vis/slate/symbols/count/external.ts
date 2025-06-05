import { Count } from "@/vis/slate/symbols/count/Count";
import { Form } from "@/vis/slate/symbols/count/Form";
import { type types } from "@/vis/slate/symbols/types";

export const REGISTRY: Record<string, types.Spec<any>> = {
  count: {
    key: "count",
    name: "Count",
    zIndex: 100,
    Form,
    Symbol: Count,
    Preview: Count,
    defaultProps: () => ({
      duration: 1000,
    }),
  },
};
