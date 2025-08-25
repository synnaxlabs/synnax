import { Form } from "@/vis/slate/symbols/sink/Form";
import { type Config, Symbol } from "@/vis/slate/symbols/sink/Sink";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SPEC: Spec<Config> = {
  key: "telem.sink",
  name: "Telemetry Sink",
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
