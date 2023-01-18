import { useState } from "react";

import { KeyboardKey } from "./types";
import { useKeysHeld } from "./useKeys";

export const useKeyMode = <M>(modes: Map<KeyboardKey, M>, defaultMode: M): M => {
  const [keys, setKeys] = useState<KeyboardKey[]>([]);
  useKeysHeld(setKeys, ...modes.keys());
  return keys.length > 0 ? (modes.get(keys[0]) as M) : defaultMode;
};
