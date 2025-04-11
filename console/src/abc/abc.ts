import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { getCurrentWindow as getTauriWindow } from "@tauri-apps/api/window";

interface TauriWindow {
  show: () => Promise<void>;
  hide: () => Promise<void>;
  maximize: () => Promise<void>;
  minimize: () => Promise<void>;
  close: () => Promise<void>;
  label: string;
}

export const getCurrentWindow = (): TauriWindow => {
  try {
    const window = getTauriWindow();
    return window;
  } catch (_) {
    return {
      show: async () => {},
      hide: async () => {},
      maximize: async () => {},
      minimize: async () => {},
      close: async () => {},
      label: MAIN_WINDOW,
    };
  }
};
