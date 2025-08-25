import { Constant } from "@/vis/slate/symbols/constant/Constant";
import { Form } from "@/vis/slate/symbols/constant/Form";
import { type types } from "@/vis/slate/symbols/types";

export const SYMBOLS: Record<string, types.Spec<any>> = {
  constant: {
    key: "constant",
    name: "Constant",
    zIndex: 100,
    Form,
    Symbol: Constant,
    Preview: Constant,
    defaultProps: () => ({
      value: 0,
      dataType: "float32",
      units: "",
    }),
  },
};
