export * from "@/table/slice";
export * from "@/table/Table";
export * from "@/table/Toolbar";
import { type Layout } from "@/layout";
import { LAYOUT_TYPE, Loaded } from "@/table/Table";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Loaded,
};
