// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/os/Controls/Mac.css";

import { type ReactElement } from "react";

import { type Button } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type InternalControlsProps } from "@/os/Controls/types";

export const Icon = {
  Close: (
    <svg
      width="124"
      height="124"
      viewBox="0 0 124 124"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect
        x="26.8701"
        y="88.3883"
        width="87"
        height="11"
        transform="rotate(-45 26.8701 88.3883)"
        fill="#7E0508"
      />
      <rect
        x="26.8701"
        y="34.6482"
        width="11"
        height="87"
        transform="rotate(-45 26.8701 34.6482)"
        fill="#7E0508"
      />
    </svg>
  ),
  Minimize: (
    <svg
      width="87"
      height="11"
      viewBox="0 0 87 11"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect x="15" width="60" height="11" fill="#985712" />
    </svg>
  ),
  Maximize: (
    <svg
      width="143"
      height="143"
      viewBox="0 0 143 143"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M108.242 32.8833C108.797 32.8817 109.247 33.3317 109.245 33.8862L109.092 87.5891C109.09 88.4786 108.014 88.9223 107.385 88.2933L53.8351 34.7432C53.2061 34.1142 53.6499 33.0387 54.5394 33.0361L108.242 32.8833Z"
        fill="#0B650D"
      />
      <path
        d="M33.8862 109.245C33.3317 109.247 32.8818 108.797 32.8833 108.242L33.0361 54.5394C33.0387 53.6499 34.1142 53.2061 34.7432 53.8351L88.2934 107.385C88.9223 108.014 88.4786 109.09 87.5891 109.092L33.8862 109.245Z"
        fill="#0B650D"
      />
    </svg>
  ),
};

export const MacOS = ({
  disabled = [],
  className,
  focused = true,
  onMinimize,
  onMaximize,
  onFullscreen,
  onClose,
  ...rest
}: InternalControlsProps): ReactElement => (
  <Flex.Box
    gap={1.5}
    x
    className={CSS(
      CSS.B("macos-controls"),
      !focused && CSS.BM("macos-controls", "blurred"),
      className,
    )}
    {...rest}
  >
    <TrafficLight
      onClick={onClose}
      className={CSS.BM("macos-control", "close")}
      disabled={disabled.includes("close")}
    >
      {Icon.Close}
    </TrafficLight>
    <TrafficLight
      onClick={onMinimize}
      className={CSS.BM("macos-control", "minimize")}
      disabled={disabled.includes("minimize")}
    >
      {Icon.Minimize}
    </TrafficLight>
    <TrafficLight
      onClick={onFullscreen}
      className={CSS.BM("macos-control", "maximize")}
      disabled={disabled.includes("maximize")}
    >
      {Icon.Maximize}
    </TrafficLight>
  </Flex.Box>
);

interface TrafficLightProps extends Button.ButtonProps {}

const TrafficLight = ({
  className,
  disabled,
  color: _,
  ...rest
}: TrafficLightProps): ReactElement => (
  <button
    className={CSS(CSS.B("macos-control"), CSS.disabled(disabled), className)}
    tabIndex={-1}
    disabled={disabled}
    {...rest}
  />
);
