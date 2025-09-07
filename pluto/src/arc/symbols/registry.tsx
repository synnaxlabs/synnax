import { Icon } from "@/icon";
import { Constant } from "@/arc/symbols/constant";
import { Operator } from "@/arc/symbols/operator";
import { Range } from "@/arc/symbols/range";
import { Select } from "@/arc/symbols/select";
import { Sink } from "@/arc/symbols/sink";
import { Source } from "@/arc/symbols/source";
import { StableFor } from "@/arc/symbols/stable";
import { Status } from "@/arc/symbols/status";
import { Time } from "@/arc/symbols/time";
import { type Spec } from "@/arc/symbols/types/spec";

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
