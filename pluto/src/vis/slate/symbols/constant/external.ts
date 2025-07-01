import { type Config, Constant } from "@/vis/slate/symbols/constant/Constant";
import { Form } from "@/vis/slate/symbols/constant/Form";
import { type types } from "@/vis/slate/symbols/types";

const CONSTANT_SPEC: types.Spec<Config> = {
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
};

export const SYMBOLS: Record<string, types.Spec<any>> = {
  [CONSTANT_SPEC.key]: CONSTANT_SPEC,
};
