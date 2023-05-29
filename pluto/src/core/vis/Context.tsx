import {
  useContext,
  createContext,
  useId,
  PropsWithChildren,
  ReactElement,
} from "react";

export interface VisCanvasContextValue {
  parent: string;
  setProps: <P>(path: string, el: string, props: P, root?: string) => void;
}

export const VisContext = createContext<VisCanvasContextValue | null>(null);

export const useVisContext = (): VisCanvasContextValue => {
  const ctx = useContext(VisContext);
  if (ctx == null) throw new Error("useVisContext must be used within a VisProvider");
  return ctx;
};

export interface VisProviderProps {
  path: string;
  keyPath: string;
  supports: (el: string) => boolean;
}

export interface UseVisElementReturn {
  key: string;
}

export const useVisElement = <P extends unknown>(
  el: string,
  props: P
): UseVisElementReturn => {
  const ctx = useVisContext();
  const key = useId();
  ctx.setProps(`${ctx.parent}.${key}`, el, props);
  return { key };
};

export interface ExtendedVisProviderProps extends PropsWithChildren {
  key: string;
}

export const ExtendedVisProvider = ({
  key,
  children,
}: ExtendedVisProviderProps): ReactElement => {
  const ctx = useVisContext();

  return (
    <VisContext.Provider
      value={{
        parent: `${ctx.parent}.${key}`,
        setProps: ctx.setProps,
      }}
    >
      {children}
    </VisContext.Provider>
  );
};
