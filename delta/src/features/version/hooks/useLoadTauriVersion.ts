import { useDispatch } from "react-redux";

import { setVersion } from "../store";
import { tauriVersion } from "../tauriVersion";

import { useAsyncEffect } from "@/hooks";

export const useLoadTauriVersion = (): void => {
  const d = useDispatch();
  useAsyncEffect(async () => {
    const version = await tauriVersion();
    d(setVersion(version));
  }, []);
};
