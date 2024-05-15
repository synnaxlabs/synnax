import { connectWindowLayout } from "@/cluster/Connect";
import { Configure } from "@/hardware/opc/device/Configure";
import { Layout } from "@/layout";

export * from "@/hardware/opc/device/Configure";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [connectWindowLayout.type]: Configure,
};
