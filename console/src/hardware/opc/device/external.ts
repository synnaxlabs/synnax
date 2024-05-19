import { Configure, CONFIGURE_LAYOUT_TYPE } from "@/hardware/opc/device/Configure";
import { Layout } from "@/layout";

export * from "@/hardware/opc/device/Configure";
export * from "@/hardware/opc/device/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CONFIGURE_LAYOUT_TYPE]: Configure,
};
