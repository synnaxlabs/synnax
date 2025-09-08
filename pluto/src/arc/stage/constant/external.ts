import { Constant } from "@/arc/stage/constant/Constant";
import { Form } from "@/arc/stage/constant/Form";
import { type types } from "@/arc/stage/types";

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
