import { Form } from "@/arc/stage/sink/Form";
import { type Config, Symbol } from "@/arc/stage/sink/Sink";
import { type Spec } from "@/arc/stage/types/spec";

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
