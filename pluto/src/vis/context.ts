import { GLContext } from "./gl/renderer";

import { Client } from "@/telem/client";
import { Theme } from "@/theming";

export interface VisContext {
  theme: Theme;
  client: Client;
  gl: GLContext;
}
