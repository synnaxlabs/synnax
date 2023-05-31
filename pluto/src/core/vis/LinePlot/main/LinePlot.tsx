import {
  CSSProperties,
  PropsWithChildren,
  ReactElement,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

import { Box, OuterLocation, locToDir, swapDir } from "@synnaxlabs/x";

import { Bob } from "@/core/bob/main";
import { CSS } from "@/core/css";
import { useResize } from "@/core/hooks";
import {
  LinePlot as WorkerLinePlot,
  LinePlotState as WorkerLinePlotState,
} from "@/core/vis/LinePlot/worker";
import { UseViewportHandler, Viewport } from "@/core/vis/viewport";

import "@/core/vis/LinePlot/main/LinePlot.css";

export interface LinePlotCProps
  extends PropsWithChildren,
    Pick<WorkerLinePlotState, "clearOverscan"> {}

export interface LinePlotContextValue {
  setAxis: (loc: OuterLocation, key: string) => void;
  deleteAxis: (key: string) => void;
}

const LinePlotContext = createContext<LinePlotContextValue>({
  setAxis: () => {},
  deleteAxis: () => {},
});

export const useAxisPosition = (loc: OuterLocation, key: string): CSSProperties => {
  const { setAxis: addAxis, deleteAxis } = useContext(LinePlotContext);
  useEffect(() => {
    addAxis(loc, key);
    return () => deleteAxis(key);
  }, [addAxis, deleteAxis, loc]);
  const dir = swapDir(locToDir(loc));
  if (dir === "x")
    return {
      gridColumnStart: "plot-start",
      gridColumnEnd: "plot-end",
      gridRowStart: `axis-start-${key}`,
      gridRowEnd: `axis-end-${key}`,
    };
  return {
    gridRowStart: "plot-start",
    gridRowEnd: "plot-end",
    gridColumnStart: `axis-start-${key}`,
    gridColumnEnd: `axis-end-${key}`,
  };
};

type AxisState = Array<[OuterLocation, string]>;

export const LinePlotC = ({ children, ...props }: LinePlotCProps): ReactElement => {
  const [axes, setAxes] = useState<AxisState>([]);
  const {
    path,
    state: [, setState],
  } = Bob.useComponent<WorkerLinePlotState>(
    WorkerLinePlot.TYPE,
    {
      plot: Box.ZERO,
      container: Box.ZERO,
      viewport: Box.DECIMAL,
      clearOverscan: { x: 15, y: 15 },
      ...props,
    },
    "line-plot"
  );

  const onViewportChange = useCallback<UseViewportHandler>(
    ({ box }) => setState((prev) => ({ ...prev, viewport: box })),
    []
  );

  const { ref: viewportRef, ...viewportProps } = Viewport.use({
    onChange: onViewportChange,
  });

  const handlePlottingRegionResize = useCallback((box: Box) => {
    setState((prev) => ({ ...prev, plot: box }));
  }, []);

  const handleContainerResize = useCallback(
    (box: Box) => setState((prev) => ({ ...prev, container: box })),
    []
  );

  const containerResizeRef = useResize(handleContainerResize, { debounce: 100 });

  const plottingRegionResizeRef = useResize(handlePlottingRegionResize, {
    debounce: 100,
  });

  const setAxis = useCallback(
    (loc: OuterLocation, key: string) =>
      setAxes((prev) => [...prev.filter(([, k]) => k !== key), [loc, key]]),
    []
  );

  const deleteAxis = useCallback(
    (key: string) => setAxes((prev) => prev.filter(([, k]) => k !== key)),
    []
  );

  const grid = buildPlotGrid(axes);

  const viewportRefCallback = useCallback(
    (el: HTMLDivElement | null) => {
      viewportRef.current = el;
      plottingRegionResizeRef(el);
    },
    [viewportRef, plottingRegionResizeRef]
  );

  return (
    <div className={CSS.B("line-plot")} style={grid} ref={containerResizeRef}>
      <Bob.Composite path={path}>
        <LinePlotContext.Provider value={{ setAxis, deleteAxis }}>
          {children}
          <Viewport.Mask
            style={{ gridArea: "plot-start / plot-start / plot-end / plot-end" }}
            {...viewportProps}
            ref={viewportRefCallback}
          />
        </LinePlotContext.Provider>
      </Bob.Composite>
    </div>
  );
};

export const AXIS_WIDTH = 15;

const buildPlotGrid = (axisCounts: AxisState): CSSProperties => {
  const grid = new CSSGridBuilder();
  const filterAxisLoc = (loc: OuterLocation): AxisState =>
    axisCounts.filter(([l]) => l === loc);
  filterAxisLoc("top").forEach(([loc, key]) => {
    grid.addRow({
      startLabel: `axis-start-${key}`,
      endLabel: `axis-end-${key}`,
      size: AXIS_WIDTH,
    });
  });
  grid.addRow({ startLabel: "plot-start", endLabel: "plot-end", size: "auto" });
  filterAxisLoc("bottom").forEach(([loc, key]) => {
    grid.addRow({
      startLabel: `axis-start-${key}`,
      endLabel: `axis-end-${key}`,
      size: AXIS_WIDTH,
    });
  });
  filterAxisLoc("left").forEach(([loc, key]) => {
    grid.addColumn({
      startLabel: `axis-start-${key}`,
      endLabel: `axis-end-${key}`,
      size: AXIS_WIDTH,
    });
  });
  grid.addColumn({ startLabel: "plot-start", endLabel: "plot-end", size: "auto" });
  filterAxisLoc("right").forEach(([loc, key]) => {
    grid.addColumn({
      startLabel: `axis-start-${key}`,
      endLabel: `axis-end-${key}`,
      size: AXIS_WIDTH,
    });
  });
  return grid.build();
};

export interface CSSGridEntry {
  startLabel: string;
  endLabel: string;
  size: number | string;
}

export class CSSGridBuilder {
  rows: CSSGridEntry[] = [];
  columns: CSSGridEntry[] = [];

  addRow(entry: CSSGridEntry): this {
    this.rows.push(entry);
    return this;
  }

  addColumn(entry: CSSGridEntry): this {
    this.columns.push(entry);
    return this;
  }

  build(): CSSProperties {
    return {
      display: "grid",
      gridTemplateRows: this.rows
        .map((r, i) => {
          let t = i === 0 ? "[" : "";
          t += `${r.startLabel}] ${r.size}${typeof r.size === "number" ? "px" : ""} [${
            r.endLabel
          }`;
          t += i === this.rows.length - 1 ? "]" : "";
          return t;
        })
        .join(" "),
      gridTemplateColumns: this.columns
        .map((c, i) => {
          let t = i === 0 ? "[" : "";
          t += `${c.startLabel}] ${c.size}${typeof c.size === "number" ? "px" : ""} [${
            c.endLabel
          }`;
          t += i === this.columns.length - 1 ? "]" : "";
          return t;
        })
        .join(" "),
    };
  }
}
