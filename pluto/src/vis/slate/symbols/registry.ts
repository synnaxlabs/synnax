import { Annotation } from "@/vis/slate/symbols/annotation";
import { Constant } from "@/vis/slate/symbols/constant";
import { Operator } from "@/vis/slate/symbols/operator";
import { Select } from "@/vis/slate/symbols/select";
import { Sink } from "@/vis/slate/symbols/sink";
import { Source } from "@/vis/slate/symbols/source";
import { type Spec } from "@/vis/slate/symbols/types/spec";
import { Variable } from "@/vis/slate/symbols/variable";

export const REGISTRY: Record<string, Spec<any>> = {
  ...Source.REGISTRY,
  ...Sink.REGISTRY,
  ...Constant.REGISTRY,
  ...Select.REGISTRY,
  ...Annotation.REGISTRY,
  ...Operator.REGISTRY,
  ...Variable.REGISTRY,
};
