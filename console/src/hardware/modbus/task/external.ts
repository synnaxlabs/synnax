import { Read, READ_SELECTABLE } from "@/hardware/modbus/task/Read";
import { READ_TYPE, WRITE_TYPE } from "@/hardware/modbus/task/types";
import { Write, WRITE_SELECTABLE } from "@/hardware/modbus/task/Write";
import { type Layout } from "@/layout";
import { type Selector } from "@/selector";

export * from "@/hardware/modbus/task/palette";
export * from "@/hardware/modbus/task/Read";
export * from "@/hardware/modbus/task/types";
export * from "@/hardware/modbus/task/Write";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [READ_TYPE]: Read,
  [WRITE_TYPE]: Write,
};

export const SELECTABLES: Selector.Selectable[] = [READ_SELECTABLE, WRITE_SELECTABLE];
