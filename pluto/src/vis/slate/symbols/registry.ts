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

export const REGISTRY: Record<string, Spec<any>> = {
  ...Source.REGISTRY,
  ...Sink.REGISTRY,
  ...Constant.REGISTRY,
  ...Select.REGISTRY,
  ...Status.REGISTRY,
  ...Operator.REGISTRY,
  ...StableFor.REGISTRY,
  ...Time.REGISTRY,
  ...Range.REGISTRY,
  ...Read.REGISTRY,
  ...Count.REGISTRY,
  ...SystemLink.REGISTRY,
  ...Calibration.REGISTRY,
  ...Task.REGISTRY,
  ...Script.REGISTRY,
};
