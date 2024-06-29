import { Layout } from "@/layout";
import { GET_STARTED_LAYOUT_TYPE } from "@/layout/slice";
import { GetStarted } from "@/layouts/GetStarted";
import { Main, MAIN_TYPE } from "@/layouts/Main";
import { Mosaic, MOSAIC_TYPE } from "@/layouts/Mosaic";
import { Selector, SELECTOR_TYPE } from "@/layouts/Selector";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [MAIN_TYPE]: Main,
  [SELECTOR_TYPE]: Selector,
  [MOSAIC_TYPE]: Mosaic,
  [GET_STARTED_LAYOUT_TYPE]: GetStarted,
};
