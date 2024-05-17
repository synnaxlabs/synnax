import { Configure, configureWindowLayout } from "@/hardware/opc/device/Configure";
import { Layout } from "@/layout";

export * from "@/hardware/opc/device/Configure";
export * from "@/hardware/opc/device/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [configureWindowLayout.type]: Configure,
};
