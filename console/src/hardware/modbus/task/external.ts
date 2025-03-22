import { Read, READ_SELECTABLE } from "@/hardware/modbus/task/Read";
import { READ_TYPE } from "@/hardware/modbus/task/types";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/hardware/modbus/task/palette";
export * from "@/hardware/modbus/task/Read";
export * from "@/hardware/modbus/task/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: Read,
};

export const SELECTABLES: Selector.Selectable[] = [READ_SELECTABLE];
