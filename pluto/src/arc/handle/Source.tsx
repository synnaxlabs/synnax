import { Core, type CoreProps } from "@/arc/handle/Core";
import { CSS } from "@/css";

export interface SourceProps extends Omit<CoreProps, "type"> {}

export const Source = ({ location, ...props }: SourceProps) => (
  <Core type="source" className={CSS.M("source")} location={location} {...props} />
);
