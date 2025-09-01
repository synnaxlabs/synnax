// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/symbol/SelectOrientation.css";

import { type location } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { Button as CoreButton } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Input } from "@/input";

export interface OrientationValue {
  inner: location.Outer;
  outer: location.Location;
}

export interface SelectOrientationProps
  extends Input.Control<OrientationValue>,
    Omit<Flex.BoxProps, "value" | "onChange"> {
  hideOuter?: boolean;
  showOuterCenter?: boolean;
  hideInner?: boolean;
}

export const SelectOrientation = ({
  value,
  hideOuter = false,
  showOuterCenter = false,
  hideInner,
  onChange,
}: SelectOrientationProps): ReactElement => {
  const { outer } = value;
  const handleChange = (next: Partial<OrientationValue>) => () =>
    onChange({ ...value, ...next });

  const className = CSS.B("select", "orientation");
  if (hideOuter)
    return (
      <InternalOrientation className={className} value={value} onChange={onChange} />
    );

  return (
    <Flex.Box className={className} align="center" justify="center" gap="tiny">
      <Button selected={outer === "top"} onClick={handleChange({ outer: "top" })} />
      <Flex.Box x align="center" justify="center" gap="tiny">
        <Button selected={outer === "left"} onClick={handleChange({ outer: "left" })} />
        <InternalOrientation
          hideInner={hideInner}
          value={value}
          onChange={onChange}
          showOuterCenter={showOuterCenter}
        />
        <Button
          selected={outer === "right"}
          onClick={handleChange({ outer: "right" })}
        />
      </Flex.Box>
      <Button
        selected={outer === "bottom"}
        onClick={handleChange({ outer: "bottom" })}
      />
    </Flex.Box>
  );
};

const InternalOrientation = ({
  value,
  onChange,
  className,
  hideInner = false,
  showOuterCenter = false,
  ...rest
}: SelectOrientationProps): ReactElement => {
  const { inner } = value;
  const handleChange = (next: Partial<OrientationValue>) => () =>
    onChange({ ...value, ...next });
  let showStyle: CSSProperties = {};
  if (hideInner)
    showStyle = { opacity: 0, userSelect: "none", width: "1.5rem", height: "1.5rem" };
  let content: ReactElement | null = null;
  if (showOuterCenter)
    content = (
      <Button
        selected={value.outer === "center"}
        onClick={handleChange({ outer: "center" })}
      />
    );
  else
    content = (
      <>
        <Button
          style={showStyle}
          disabled={hideInner}
          className={CSS(CSS.dir("y"))}
          selected={inner === "top"}
          onClick={handleChange({ inner: "top" })}
        />
        <Flex.Box x align="center" justify="center">
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
        </Flex.Box>
        <Button
          style={showStyle}
          disabled={hideInner}
          className={CSS(CSS.dir("y"))}
          selected={inner === "bottom"}
          onClick={handleChange({ inner: "bottom" })}
        />
      </>
    );
  return (
    <Flex.Box
      className={CSS(
        className,
        CSS.B("value"),
        showOuterCenter && CSS.M("show-outer-center"),
      )}
      y
      align="center"
      justify="center"
      empty
      {...rest}
    >
      {content}
    </Flex.Box>
  );
};

export interface ButtonProps extends Omit<CoreButton.ButtonProps, "children"> {
  selected: boolean;
}

export const Button = ({ selected, className, ...rest }: ButtonProps): ReactElement => (
  <CoreButton.Button
    variant="text"
    className={CSS(className, CSS.selected(selected))}
    size="tiny"
    {...rest}
  >
    <div className="symbol" />
  </CoreButton.Button>
);
