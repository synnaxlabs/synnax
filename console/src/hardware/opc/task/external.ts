import { ReadTask, configureReadLayout } from "@/hardware/opc/task/ReadTask";
import { Layout } from "@/layout";

export * from "@/hardware/opc/task/ReadTask";
export * from "@/hardware/opc/task/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [configureReadLayout.type]: ReadTask,
};
