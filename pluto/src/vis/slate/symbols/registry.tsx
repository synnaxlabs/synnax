import { Icon } from "@/icon";
import { Constant } from "@/vis/slate/symbols/constant";
import { Operator } from "@/vis/slate/symbols/operator";
import { Range } from "@/vis/slate/symbols/range";
import { Script } from "@/vis/slate/symbols/script";
import { Select } from "@/vis/slate/symbols/select";
import { Sink } from "@/vis/slate/symbols/sink";
import { Source } from "@/vis/slate/symbols/source";
import { StableFor } from "@/vis/slate/symbols/stable";
import { Status } from "@/vis/slate/symbols/status";
import { Time } from "@/vis/slate/symbols/time";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const SYMBOLS: Record<string, Spec<any>> = {
  ...Source.SYMBOLS,
  ...Sink.SYMBOLS,
  ...Constant.SYMBOLS,
  ...Select.SYMBOLS,
  ...Status.SYMBOLS,
  ...Operator.SYMBOLS,
  ...StableFor.SYMBOLS,
  ...Time.SYMBOLS,
  ...Range.SYMBOLS,
  ...Script.SYMBOLS,
};

export interface Group {
  key: string;
  name: string;
  icon: Icon.ReactElement;
  symbols: Record<string, Spec<any>>;
}

export const REGISTRY: Record<string, Group> = {
  basic: {
    key: "basic",
    name: "Basic",
    icon: <Icon.Schematic />,
    symbols: {
      ...Constant.SYMBOLS,
      ...Status.SYMBOLS,
    },
  },
  telem: {
    key: "telem",
    name: "Telemetry",
    icon: <Icon.Channel />,
    symbols: {
      ...Source.SYMBOLS,
      ...Sink.SYMBOLS,
    },
  },
  operator: {
    key: "operator",
    name: "Operators",
    icon: <Icon.Add />,
    symbols: Operator.SYMBOLS,
  },
  range: {
    key: "range",
    name: "Ranges",
    icon: <Icon.Range />,
    symbols: Range.SYMBOLS,
  },
  flow_control: {
    key: "flow_control",
    name: "Flow Control",
    icon: <Icon.Select />,
    symbols: {
      ...Select.SYMBOLS,
      ...StableFor.SYMBOLS,
    },
  },
  time: {
    key: "time",
    name: "Time",
    icon: <Icon.Time />,
    symbols: Time.SYMBOLS,
  },
};
