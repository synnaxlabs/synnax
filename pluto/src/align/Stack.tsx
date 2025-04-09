// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/align/Stack.css";

import { toArray } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { type SpaceProps } from "@/align/Space";
import { CSS } from "@/css";

export interface StackProps extends SpaceProps {}

export const Stack = ({ className, children, ...rest }: StackProps): ReactElement => {
  const arr = toArray(children);
  return (
    <Align.Space x className={CSS.B("stack")} {...rest}>
      <div />
      {arr.map((child, i) => (
        <div key={i}>{child}</div>
      ))}
    </Align.Space>
  );
};
