// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { Color } from "@/color";
import { CSS } from "@/css";

import "@/vis/regulator/Regulator.css";

export interface RegulatorProps extends Omit<ComponentPropsWithoutRef<"svg">, "color"> {
  color?: Color.Crude;
  label?: string;
}

export const Regulator = ({
  color,
  className,
  ...props
}: RegulatorProps): ReactElement => {
  return (
    <svg
      width="104"
      height="52"
      viewBox="0 0 104 52"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <rect x="2" y="1" width="100" height="50" fill="white" />
      <path
        d="M52 25.5L4.88003 2.41121C3.55123 1.7601 2 2.72744 2 4.20719V47.7349C2 49.2287 3.57798 50.1952 4.90865 49.5166L52 25.5ZM52 25.5L99.12 2.41121C100.449 1.7601 102 2.72744 102 4.2072V47.7349C102 49.2287 100.422 50.1952 99.0913 49.5166L52 25.5Z"
        stroke="black"
        stroke-width="3"
      />
      <path
        d="M52 25.5L4.88003 2.41121C3.55123 1.7601 2 2.72744 2 4.20719V47.7349C2 49.2287 3.57798 50.1952 4.90865 49.5166L52 25.5ZM52 25.5L99.12 2.41121C100.449 1.7601 102 2.72744 102 4.2072V47.7349C102 49.2287 100.422 50.1952 99.0913 49.5166L52 25.5Z"
        stroke="black"
        stroke-width="3"
      />
      <g filter="url(#filter0_i_679_12)">
        <path
          d="M52.9701 26C69.7245 52.2615 52.6377 50.7641 42 46.7328"
          stroke="black"
          stroke-width="3"
        />
      </g>
      <g filter="url(#filter1_i_679_12)">
        <path
          d="M52.0299 26C35.2755 -0.261493 52.3623 1.23587 63 5.26724"
          stroke="black"
          stroke-width="3"
        />
      </g>
      <defs>
        <filter
          id="filter0_i_679_12"
          x="41.4684"
          y="25.1932"
          width="20.0297"
          height="29.3068"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend
            mode="normal"
            in="SourceGraphic"
            in2="BackgroundImageFix"
            result="shape"
          />
          <feColorMatrix
            in="SourceAlpha"
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
            result="hardAlpha"
          />
          <feOffset dy="4" />
          <feGaussianBlur stdDeviation="2" />
          <feComposite in2="hardAlpha" operator="arithmetic" k2="-1" k3="1" />
          <feColorMatrix
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0"
          />
          <feBlend mode="normal" in2="shape" result="effect1_innerShadow_679_12" />
        </filter>
        <filter
          id="filter1_i_679_12"
          x="43.5019"
          y="1.49997"
          width="20.0297"
          height="29.3068"
          filterUnits="userSpaceOnUse"
          color-interpolation-filters="sRGB"
        >
          <feFlood flood-opacity="0" result="BackgroundImageFix" />
          <feBlend
            mode="normal"
            in="SourceGraphic"
            in2="BackgroundImageFix"
            result="shape"
          />
          <feColorMatrix
            in="SourceAlpha"
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0"
            result="hardAlpha"
          />
          <feOffset dy="4" />
          <feGaussianBlur stdDeviation="2" />
          <feComposite in2="hardAlpha" operator="arithmetic" k2="-1" k3="1" />
          <feColorMatrix
            type="matrix"
            values="0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0"
          />
          <feBlend mode="normal" in2="shape" result="effect1_innerShadow_679_12" />
        </filter>
      </defs>
    </svg>
  );
};
