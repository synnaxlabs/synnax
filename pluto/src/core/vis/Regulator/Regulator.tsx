// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentPropsWithoutRef, ReactElement } from "react";

import { Color, CrudeColor } from "@/core/color";
import { CSS } from "@/core/css";

import "@/core/vis/Regulator/Regulator.css";

export interface RegulatorProps extends Omit<ComponentPropsWithoutRef<"svg">, "color"> {
  color?: CrudeColor;
  label?: string;
}

export const Regulator = ({
  color,
  className,
  ...props
}: RegulatorProps): ReactElement => {
  return (
    <svg className={CSS.B("regulator")} viewBox="0 0 104 112" {...props}>
      <path
        stroke={Color.cssString(color)}
        className={CSS.BE("regulator", "body")}
        d="M52 85.5L4.88003 62.4112C3.55123 61.7601 2 62.7274 2 64.2072V107.735C2 109.229 3.57798 110.195 4.90865 109.517L52 85.5ZM52 85.5L99.12 62.4112C100.449 61.7601 102 62.7274 102 64.2072V107.735C102 109.229 100.422 110.195 99.0913 109.517L52 85.5Z"
      />
      <path
        stroke={Color.cssString(color)}
        className={CSS.BE("regulator", "body")}
        d="M21.0088 4.69999L50.1265 82.6327C50.9063 84.7197 54 84.1607 54 81.9327V4C54 2.89543 53.1046 2 52 2H22.8823C21.4869 2 20.5204 3.39286 21.0088 4.69999Z"
      />
    </svg>
  );
};
