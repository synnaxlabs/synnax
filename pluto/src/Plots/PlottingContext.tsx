import {
  createContext,
  PropsWithChildren,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import "./PlotContext.css";
import BasePlottingContext from "./engine/Canvas";

export const PlottingContext = createContext<{
  context?: BasePlottingContext;
  randomNumber?: number;
}>({});

export const usePlottingContext = () => {
  return useContext(PlottingContext).context;
};

export const PlottingContextProvider = ({
  children,
}: PropsWithChildren<any>) => {
  const [plottingContext, setPlottingContext] = useState<{
    context?: BasePlottingContext;
    randomNumber?: number;
  }>({});
  const setCanvas = useCallback((canvas: HTMLCanvasElement) => {
    canvas &&
      setPlottingContext({ context: new BasePlottingContext({ canvas }) });
  }, []);

  return (
    <PlottingContext.Provider value={plottingContext}>
      <canvas id="plot-context" ref={setCanvas}></canvas>
      {children}
    </PlottingContext.Provider>
  );
};
