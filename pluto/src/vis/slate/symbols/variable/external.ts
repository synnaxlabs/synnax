import { type types } from "@/vis/slate/symbols/types";
import { Form } from "@/vis/slate/symbols/variable/Form";
import { Variable } from "@/vis/slate/symbols/variable/Variable";

export const REGISTRY: Record<string, types.Spec<any>> = {
  variable: {
    key: "variable",
    name: "Variable",
    zIndex: 100,
    Form,
    Symbol: Variable,
    Preview: Variable,
    defaultProps: () => ({
      value: 0,
      dataType: "float32",
    }),
  },
};
