import { LinePlotCore, LinePlotCoreProps } from "./LinePlotCore";

export type LinePlotProps = LinePlotCoreProps;

export const LinePlot = (props: LinePlotProps) => {
  return <LinePlotCore.UPlot {...props} />;
};
