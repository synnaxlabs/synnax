import { Constant } from "@/vis/slate/symbols/constant";
import { Notification } from "@/vis/slate/symbols/notification";
import { Comparisons } from "@/vis/slate/symbols/operators";
import { Select } from "@/vis/slate/symbols/select";
import { Sink } from "@/vis/slate/symbols/sink";
import { Source } from "@/vis/slate/symbols/source";
import { type Spec } from "@/vis/slate/symbols/types/spec";

export const REGISTRY: Record<string, Spec<any>> = {
  ...Source.REGISTRY,
  ...Sink.REGISTRY,
  ...Comparisons.REGISTRY,
  ...Constant.REGISTRY,
  ...Select.REGISTRY,
  ...Notification.REGISTRY,
};
