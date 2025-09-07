import { caseconv, type location } from "@synnaxlabs/x";
import {
  Handle,
  type HandleProps,
  Position,
  useUpdateNodeInternals,
} from "@xyflow/react";

import { CSS } from "@/css";

export interface CoreProps extends Omit<HandleProps, "position"> {
  location: location.Outer;
}

export const locationToRFPosition = (location: location.Outer) =>
  Position[caseconv.capitalize(location) as keyof typeof Position];

export const Core = ({ location, className, ...props }: CoreProps) => {
  try {
    useUpdateNodeInternals();
  } catch {
    return null;
  }
  const position = locationToRFPosition(location);
  return (
    <Handle
      className={CSS(CSS.BE("slate", "handle"), className)}
      position={position}
      {...props}
    />
  );
};
