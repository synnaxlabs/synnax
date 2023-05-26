import { GLRenderContext } from "./gl/renderer";

import { Client } from "@/telem/client";
import { Theme } from "@/theming";

export interface VisBuilderContext {
  theme: Theme;
  client: Client;
}

export interface VisRenderContext {
  gl: GLRenderContext;
}
