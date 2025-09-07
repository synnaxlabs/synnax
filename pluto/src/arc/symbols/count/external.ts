import { Count } from "@/arc/symbols/count/Count";
import { Form } from "@/arc/symbols/count/Form";
import { type types } from "@/arc/symbols/types";

export const SYMBOLS: Record<string, types.Spec<any>> = {
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
