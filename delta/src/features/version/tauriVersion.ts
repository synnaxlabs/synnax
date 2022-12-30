import { getVersion } from "@tauri-apps/api/app";

/** @returns the tauri application version as exposed by the tauri apps API. */
export const tauriVersion = async (): Promise<string> => await getVersion();
