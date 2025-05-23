import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { type Event, type UnlistenFn } from "@tauri-apps/api/event";
import { getCurrentWindow as getTauriWindow, type Theme } from "@tauri-apps/api/window";

interface TauriWindow {
  show: () => Promise<void>;
  hide: () => Promise<void>;
  maximize: () => Promise<void>;
  minimize: () => Promise<void>;
  close: () => Promise<void>;
  theme: () => Promise<Theme | null>;
  onThemeChanged: (callback: (theme: Event<Theme>) => void) => Promise<UnlistenFn>;
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
      theme: async () => null,
      onThemeChanged: async () => () => {},
    };
  }
};
