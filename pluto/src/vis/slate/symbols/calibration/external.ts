import { type Config, Symbol } from "@/vis/slate/symbols/calibration/Calibration";
import { Form } from "@/vis/slate/symbols/calibration/Form";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SPEC: Spec<Config> = {
  key: "calibration.query",
  name: "Query Calibrations from Database",
  Form,
  Symbol,
  defaultProps: () => ({
    channel: 0,
  }),
  Preview: Symbol,
  zIndex: 0,
};

export const SYMBOLS = {
  [SPEC.key]: SPEC,
};
