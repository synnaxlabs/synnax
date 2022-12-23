import { useState } from "react";

import { getVersion } from "@tauri-apps/api/app";

import { useAsyncEffect } from "./useAsyncEffect";

/**
 * useVersion gets the application version.
 * @returns The application version in the format "vX.X.X".
 */
export const useVersion = (): string => {
  const [v, setV] = useState<string>("");
  useAsyncEffect(async () => setV("v" + (await getVersion())), []);
  return v;
};
