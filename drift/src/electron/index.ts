import { isRenderer } from "./util";
import MainRuntime from "./main";
import RendererRuntime from "./renderer";

export const ElectronRuntime = isRenderer() ? RendererRuntime : MainRuntime;
