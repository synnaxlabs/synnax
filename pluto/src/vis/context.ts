import { RenderContext } from "../core/vis/render/RenderContext";

import { Theme } from "@/core/theming";
import { Client } from "@/telem/client";

export interface VisBuilderContext {
  theme: Theme;
  client: Client;
}

export interface VisRenderContext {
  gl: RenderContext;
}
