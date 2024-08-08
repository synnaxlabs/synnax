import { Edit, EDIT_LAYOUT_TYPE } from "@/label/Edit";
import { Layout } from "@/layout";

export * from "@/label/Edit";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [EDIT_LAYOUT_TYPE]: Edit,
};
