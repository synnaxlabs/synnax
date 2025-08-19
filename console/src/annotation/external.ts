import { Create, CREATE_LAYOUT_TYPE } from "@/annotation/Create";
import { type Layout } from "@/layout";

export * from "@/annotation/Create";
export * from "@/annotation/list/List";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CREATE_LAYOUT_TYPE]: Create,
};
