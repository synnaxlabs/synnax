import { ReadTask, readTaskLayout } from "@/hardware/opc/task/ReadTask";
import { Layout } from "@/layout";

export * from "@/hardware/opc/task/ReadTask";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [readTaskLayout.type]: ReadTask,
};
