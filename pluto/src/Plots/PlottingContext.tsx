import {
  createContext,
  PropsWithChildren,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import "./PlotContext.css";
import PlottingEngine from "./engine/PlottingEngine";

export const PlottingEngineContext = createContext<PlottingEngine | undefined>(
  undefined
);

export const usePlottingEngine = () => useContext(PlottingEngineContext);

export const PlottingEngineProvider = ({
  children,
}: PropsWithChildren<any>) => {
  const [plottingEngine, setPlottingEngine] = useState<
    PlottingEngine | undefined
  >(undefined);
  const setCanvas = useCallback(
    (canvas: HTMLCanvasElement) =>
      canvas && setPlottingEngine(new PlottingEngine({ canvas })),
    []
  );
  return (
    <PlottingEngineContext.Provider value={plottingEngine}>
      <canvas id="plot-context" ref={setCanvas}></canvas>
      {children}
    </PlottingEngineContext.Provider>
  );
};
