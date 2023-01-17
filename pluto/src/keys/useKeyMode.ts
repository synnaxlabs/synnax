import { KeyboardKey } from "./types";
import { useKeysHeld } from "./useKeys";

export const useKeyMode = <M>(modes: Array<[KeyboardKey, M]>, defaultMode: M): M => {
  const { keys } = useKeysHeld(modes.map(([key]) => key));
  const mode = modes.find(([key]) => keys.includes(key));
  return mode == null ? defaultMode : mode[1];
};
