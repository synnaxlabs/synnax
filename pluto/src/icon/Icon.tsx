import "@/icon/Icon.css";

import { location } from "@synnaxlabs/x";
import { cloneElement, ReactElement, SVGProps } from "react";

import { CSS } from "@/css";

interface BaseIconProps extends SVGProps<SVGSVGElement> {}

export interface IconProps
  extends Partial<Record<location.CornerXYString, ReactElement<BaseIconProps>>> {
  children: ReactElement<BaseIconProps>;
}

const clone = (value: ReactElement<BaseIconProps>, key: location.CornerXYString) =>
  cloneElement(value, { className: CSS(value.props.className, CSS.M(key)) });

export const Icon = ({
  topRight,
  topLeft,
  bottomLeft,
  bottomRight,
  children,
}: IconProps) => {
  return (
    <div className={CSS.B("icon")}>
      {topRight && clone(topRight, "topRight")}
      {topLeft && clone(topLeft, "topLeft")}
      {bottomLeft && clone(bottomLeft, "bottomLeft")}
      {bottomRight && clone(bottomRight, "bottomRight")}
      {children}
    </div>
  );
};
