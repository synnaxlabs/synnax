import { useStateRef } from "@/hooks/useStateRef";
import { Deep } from "@synnaxlabs/x";
import { createContext, MutableRefObject, PropsWithChildren, Ref, useCallback, useContext, useEffect, useMemo, useState } from "react";

export interface Hauled {
  key: string;
  type: string;
}

export interface HaulContextValue {
  startDrag: (entities: Hauled[]) => void;
  endDrag: () => void;
  dragging: Hauled[];
}


const HaulContext = createContext<HaulContextValue | null>(null);

export const useHaulContext = (): HaulContextValue => {
  const ctx = useContext(HaulContext);
  if (ctx == null) throw new Error("HaulContext not available");
  return ctx
}

export const useOptionalHaulContext = (): HaulContextValue | null => {
  return useContext(HaulContext);
}

export interface HaulProviderProps extends PropsWithChildren {
}

interface HaulState {
  dragging: Hauled[];
}

const ZERO_HAUL_STATE = { dragging: [] }

export const HaulProvider = ({ children }: HaulProviderProps): JSX.Element => {
  const ctx = useOptionalHaulContext()
  const [state, setState] = useState<HaulState>(Deep.copy(ZERO_HAUL_STATE));

  const startDrag = useCallback((entities: Hauled[]) => {
    setState((p) => ({ ...p, dragging: entities }))
  }, [setState])

  const endDrag = useCallback(() => {
    setState((p) => ({ ...p, dragging: [] }))
  }, [setState])

  return <HaulContext.Provider value={ctx ?? {
    dragging: state.dragging,
    startDrag: startDrag,
    endDrag: endDrag,
  }}>{children}</HaulContext.Provider>
}

export interface UseHaulStateReturn extends HaulContextValue {

}

export const useHaulState = (): UseHaulStateReturn => {
  return useHaulContext();
}

export interface UseHaulRefReturn extends Omit<HaulContextValue, "dragging"> {
  dragging: MutableRefObject<Hauled[]>
}

export const useHaulRef = (): UseHaulRefReturn => {
  const [ref, setRef] = useStateRef<Hauled[]>([]);
  const { startDrag, endDrag, dragging } = useHaulContext();
  useEffect(() => setRef(dragging), [setRef, dragging])

  return useMemo(() => ({
    dragging: ref,
    startDrag,
    endDrag
  }), [ref, startDrag, endDrag])
}
