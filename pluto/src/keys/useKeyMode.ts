import { KeyboardKey } from "./types";
import { useKeysHeld } from "./useKeys";

export const useKeyMode = <M>(modes: Map<KeyboardKey, M>, defaultMode: M): M => {
  const { keys, any } = useKeysHeld(...modes.keys());
  return any ? (modes.get(keys[0]) as M) : defaultMode;
};
