import { Form } from "@/arc/symbols/sink/Form";
import { type Config, Symbol } from "@/arc/symbols/sink/Sink";
import { type Spec } from "@/arc/symbols/types/spec";

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
