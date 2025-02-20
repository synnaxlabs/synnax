// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/Canvas.css";

import { Canvas as Core } from "@synnaxlabs/pluto";
import { type PropsWithChildren } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";

export const Canvas = ({ children }: PropsWithChildren) => {
  const { focused } = Layout.useSelectFocused();
  return (
    <Core.Canvas
      id={CSS.BE("vis", "canvas")}
      className={CSS(focused != null && CSS.M("focused"))}
    >
      {children}
    </Core.Canvas>
  );
};
