// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cloneElement, HTMLAttributes } from "react";

import clsx from "clsx";

import "./Logo.css";

export interface LogoProps
  extends Omit<HTMLAttributes<SVGElement>, "width" | "height"> {
  variant?: "icon" | "title";
  color?: "white" | "black" | "gradient" | "auto";
}

export const Logo = ({
  variant = "icon",
  color = "auto",
  className,
  ...props
}: LogoProps): JSX.Element => {
  const logo = variant === "icon" ? icon : title;
  return cloneElement(logo, {
    className: clsx(`synnax-logo--${color}`, className),
    ...props,
  });
};

const icon = (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 189.34">
    <defs>
      <linearGradient
        id="synnalinear-gradient"
        x1="-2.63"
        y1="56.85"
        x2="167.19"
        y2="157.88"
        gradientUnits="userSpaceOnUse"
      >
        <stop offset="0" stopColor="#10007f" />
        <stop offset="1" stopColor="#0084e9" />
      </linearGradient>
    </defs>
    <g id="Layer_2" data-name="Layer 2">
      <g id="Layer_1-2" data-name="Layer 1">
        <path
          className="cls-1"
          d="M52.61,168.82A14.81,14.81,0,0,1,39.78,146.6L94.22,52.33c5.7-9.88,20-9.88,25.9.42l51.77,89.67a6.88,6.88,0,0,0,2.48,2.49l15.42,8.9a6.78,6.78,0,0,0,9.26-9.27L119.87,7.41a14.8,14.8,0,0,0-25.65,0L2,167.12a14.81,14.81,0,0,0,12.83,22.22H170.39a6.79,6.79,0,0,0,3.39-12.66l-12.05-7a6.83,6.83,0,0,0-3.39-.91Z"
        />
      </g>
    </g>
  </svg>
);

const title = (
  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1050 285">
    <defs>
      <linearGradient
        id="linear-gradient"
        x1="-2.64"
        y1="16561.34"
        x2="167.18"
        y2="16460.3"
        gradientTransform="matrix(1, 0, 0, -1, 0, 16643.76)"
        gradientUnits="userSpaceOnUse"
      >
        <stop offset="0" stopColor="#10007f" />
        <stop offset="1" stopColor="#0084e9" />
      </linearGradient>
    </defs>
    <g id="Layer_2" data-name="Layer 2">
      <path d="M52.6,194.4a14.81,14.81,0,0,1-12.83-22.22L94.21,77.91c5.7-9.88,20-9.88,25.9.42L171.88,168a6.78,6.78,0,0,0,2.48,2.48l15.42,8.91a6.78,6.78,0,0,0,9.26-9.27L119.86,33a14.8,14.8,0,0,0-25.65,0L2,192.69a14.81,14.81,0,0,0,12.83,22.22H170.38a6.78,6.78,0,0,0,3.39-12.65l-12.05-6.95a6.73,6.73,0,0,0-3.39-.91Z" />
      <text transform="translate(208.99 215.58)">Synnax</text>
    </g>
  </svg>
);
