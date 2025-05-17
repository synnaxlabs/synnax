import { CSS } from "@/css";
import { Core, type CoreProps } from "@/vis/slate/handle/Core";

export interface SourceProps extends Omit<CoreProps, "type"> {}

export const Source = ({ location, ...props }: SourceProps) => (
  <Core type="source" className={CSS.M("source")} location={location} {...props} />
);
