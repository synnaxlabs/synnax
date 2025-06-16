import { Icon } from "@/icon";
import { Calibration } from "@/vis/slate/symbols/calibration";
import { Constant } from "@/vis/slate/symbols/constant";
import { Count } from "@/vis/slate/symbols/count";
import { Operator } from "@/vis/slate/symbols/operator";
import { Range } from "@/vis/slate/symbols/range";
import { Read } from "@/vis/slate/symbols/read";
import { Script } from "@/vis/slate/symbols/script";
import { Select } from "@/vis/slate/symbols/select";
import { Sink } from "@/vis/slate/symbols/sink";
import { Source } from "@/vis/slate/symbols/source";
import { StableFor } from "@/vis/slate/symbols/stable";
import { Status } from "@/vis/slate/symbols/status";
import { SystemLink } from "@/vis/slate/symbols/systemlink";
import { Task } from "@/vis/slate/symbols/task";
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
  ...Read.SYMBOLS,
  ...Count.SYMBOLS,
  ...SystemLink.SYMBOLS,
  ...Calibration.SYMBOLS,
  ...Task.SYMBOLS,
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
    icon: <Icon.Add />,
    symbols: {
      ...Constant.SYMBOLS,
      ...Count.SYMBOLS,
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
      ...Read.SYMBOLS,
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
  task: {
    key: "task",
    name: "Task",
    icon: <Icon.Task />,
    symbols: Task.SYMBOLS,
  },
};
