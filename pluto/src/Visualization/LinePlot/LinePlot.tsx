import { LinePlotCore, LinePlotCoreProps } from "./LinePlotCore";

export interface LinePlotProps extends LinePlotCoreProps {}

export const LinePlot = (props: LinePlotProps) => {
  return <LinePlotCore.UPlot {...props} />;
};
