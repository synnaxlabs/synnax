// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/schematic/OrientationControl.css";

import { type location } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { Align } from "@/align";
import { Button as CoreButton } from "@/button";
import { CSS } from "@/css";
import { type Input } from "@/input";

export interface OrientationValue {
  inner: location.Outer;
  outer: location.Outer;
}

export interface SelectOrientationProps
  extends Input.Control<OrientationValue>,
    Omit<Align.SpaceProps, "value" | "onChange"> {
  hideOuter?: boolean;
  hideInner?: boolean;
}

export const SelectOrientation = ({
  value,
  hideOuter = false,
  hideInner,
  onChange,
}: SelectOrientationProps): ReactElement => {
  const { outer } = value;
  const handleChange = (next: Partial<OrientationValue>) => () =>
    onChange({ ...value, ...next });

  if (hideOuter)
    return (
      <InternalOrientation
        className={CSS.B("orientation-control")}
        value={value}
        onChange={onChange}
      />
    );

  return (
    <Align.Space
      className={CSS.B("orientation-control")}
      align="center"
      justify="center"
      size={0.5}
    >
      <Button selected={outer === "top"} onClick={handleChange({ outer: "top" })} />
      <Align.Space direction="x" align="center" justify="center" size={0.5}>
        <Button selected={outer === "left"} onClick={handleChange({ outer: "left" })} />
        <InternalOrientation hideInner={hideInner} value={value} onChange={onChange} />
        <Button
          selected={outer === "right"}
          onClick={handleChange({ outer: "right" })}
        />
      </Align.Space>
      <Button
        selected={outer === "bottom"}
        onClick={handleChange({ outer: "bottom" })}
      />
    </Align.Space>
  );
};

const InternalOrientation = ({
  value,
  onChange,
  className,
  hideInner = false,
  ...rest
}: SelectOrientationProps): ReactElement => {
  const { inner } = value;
  const handleChange = (next: Partial<OrientationValue>) => () =>
    onChange({ ...value, ...next });
  let showStyle: CSSProperties = {};
  if (hideInner)
    showStyle = { opacity: 0, userSelect: "none", width: "1.5rem", height: "1.5rem" };
  return (
    <Align.Space
      className={CSS(className, CSS.B("value"))}
      direction="y"
      align="center"
      justify="center"
      empty
      {...rest}
    >
      <Button
        style={showStyle}
        disabled={hideInner}
        className={CSS(CSS.dir("y"))}
        selected={inner === "top"}
        onClick={handleChange({ inner: "top" })}
      />
      <Align.Space direction="x" align="center" justify="center">
        <Button
          style={showStyle}
          disabled={hideInner}
          selected={inner === "left"}
          onClick={handleChange({ inner: "left" })}
        />
        <Button
          style={showStyle}
          disabled={hideInner}
          selected={inner === "right"}
          onClick={handleChange({ inner: "right" })}
        />
      </Align.Space>
      <Button
        style={showStyle}
        disabled={hideInner}
        className={CSS(CSS.dir("y"))}
        selected={inner === "bottom"}
        onClick={handleChange({ inner: "bottom" })}
      />
    </Align.Space>
  );
};

export interface ButtonProps extends Omit<CoreButton.IconProps, "children"> {
  selected: boolean;
}

export const Button = ({ selected, className, ...rest }: ButtonProps): ReactElement => (
  <CoreButton.Icon
    variant="text"
    className={CSS(className, CSS.selected(selected))}
    {...rest}
  >
    <div className="symbol"></div>
  </CoreButton.Icon>
);
