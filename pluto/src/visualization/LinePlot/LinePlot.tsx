import { LinePlotCoreComponent, LinePlotCoreProps } from "./LinePlotCore";

export type LinePlotProps = LinePlotCoreProps;

export const LinePlot = (props: LinePlotProps): JSX.Element => (
  <LinePlotCore.UPlot {...props} />
);
