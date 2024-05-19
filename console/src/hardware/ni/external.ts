import { Layout } from "@/layout";

export * from "@/hardware/ni/pallette";
export * from "@/hardware/ni/task";
export * from "@/hardware/ni/device";
import { Task } from "@/hardware/ni/task";
import { Device } from "@/hardware/ni/device";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...Task.LAYOUTS,
  ...Device.LAYOUTS,
};
