import { useDispatch } from "react-redux";

import { setVersion } from "../store";
import { tauriVersion } from "../tauriVersion";

import { useAsyncEffect } from "@/hooks";

export const useLoadTauriVersion = (): void => {
  const d = useDispatch();
  useAsyncEffect(async () => {
    d(setVersion(await tauriVersion()));
  }, []);
};
