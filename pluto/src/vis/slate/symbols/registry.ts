import { Constant } from "@/vis/slate/symbols/constant";
import { Operator } from "@/vis/slate/symbols/operator";
import { Range } from "@/vis/slate/symbols/range";
import { Select } from "@/vis/slate/symbols/select";
import { Sink } from "@/vis/slate/symbols/sink";
import { Source } from "@/vis/slate/symbols/source";
import { StableFor } from "@/vis/slate/symbols/stable";
import { Status } from "@/vis/slate/symbols/status";
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
};
