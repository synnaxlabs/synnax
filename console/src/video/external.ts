import { Layout } from "@/layout";
import { LAYOUT_TYPE, SELECTABLE, Video } from "@/video/Video";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [LAYOUT_TYPE]: Video,
};

export const SELECTABLES: Layout.Selectable[] = [SELECTABLE];
