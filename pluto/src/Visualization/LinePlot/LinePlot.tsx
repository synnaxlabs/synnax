import Cores, { LinePlotCoreProps } from "./LinePlotCore";

export interface PlotProps extends LinePlotCoreProps {}

export default function Plot(props: PlotProps) {
  return <Cores.UPlot {...props} />;
}
