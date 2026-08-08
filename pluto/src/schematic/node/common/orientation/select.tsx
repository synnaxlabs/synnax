// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/common/orientation/select.css";

import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Button as BaseButton } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Input } from "@/input";

export interface Value {
  inner: location.Outer;
  outer: location.Location;
}

export interface SelectProps
  extends Input.Control<Value>, Omit<Flex.BoxProps, "value" | "onChange"> {
  hideOuter?: boolean;
  showOuterCenter?: boolean;
  hideInner?: boolean;
}

export const Select = ({
  value,
  hideOuter = false,
  showOuterCenter = false,
  hideInner,
  onChange,
}: SelectProps): ReactElement => {
  const { outer } = value;
  const handleChange = (next: Partial<Value>) => () => onChange({ ...value, ...next });

  const className = CSS.B("select", "orientation");
  if (hideOuter)
    return <Internal className={className} value={value} onChange={onChange} />;

  return (
    <Flex.Box className={className} align="center" justify="center" gap="tiny">
      <Button selected={outer === "top"} onClick={handleChange({ outer: "top" })} />
      <Flex.Box x align="center" justify="center" gap="tiny">
        <Button selected={outer === "left"} onClick={handleChange({ outer: "left" })} />
        <Internal
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

const Internal = ({
  value,
  onChange,
  className,
  hideInner = false,
  showOuterCenter = false,
  ...rest
}: SelectProps): ReactElement => {
  const { inner } = value;
  const handleChange = (next: Partial<Value>) => () => onChange({ ...value, ...next });
  const hiddenInnerClass = hideInner && CSS.M("hidden-inner");
  const content = showOuterCenter ? (
    <Button
      selected={value.outer === "center"}
      onClick={handleChange({ outer: "center" })}
    />
  ) : (
    <>
      <Button
        disabled={hideInner}
        className={CSS(CSS.dir("y"), hiddenInnerClass)}
        selected={inner === "top"}
        onClick={handleChange({ inner: "top" })}
      />
      <Flex.Box x align="center" justify="center">
        <Button
          disabled={hideInner}
          className={CSS(hiddenInnerClass)}
          selected={inner === "left"}
          onClick={handleChange({ inner: "left" })}
        />
        <Button
          disabled={hideInner}
          className={CSS(hiddenInnerClass)}
          selected={inner === "right"}
          onClick={handleChange({ inner: "right" })}
        />
      </Flex.Box>
      <Button
        disabled={hideInner}
        className={CSS(CSS.dir("y"), hiddenInnerClass)}
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

export interface ButtonProps extends Omit<BaseButton.ButtonProps, "children"> {
  selected: boolean;
}

export const Button = ({ selected, className, ...rest }: ButtonProps): ReactElement => (
  <BaseButton.Button
    variant={selected ? "outlined" : "text"}
    className={CSS(className, CSS.B("select-btn"), CSS.selected(selected))}
    size="tiny"
    {...rest}
  >
    <div className="symbol" />
  </BaseButton.Button>
);
