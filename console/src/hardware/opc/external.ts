import { Layout } from "@/layout";

export * from "@/hardware/opc/device";
export * from "@/hardware/opc/palette";
export * from "@/hardware/opc/task";
import { Device } from "@/hardware/opc/device";
import { Task } from "@/hardware/opc/task";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...Device.LAYOUTS,
  ...Task.LAYOUTS,
};
