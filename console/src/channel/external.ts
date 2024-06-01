import { Create,CREATE_LAYOUT_TYPE } from "@/channel/Create";
import { Layout } from "@/layout";

export * from "@/channel/Create";
export * from "@/channel/ontology";
export * from "@/channel/palette";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [CREATE_LAYOUT_TYPE]: Create,
};
