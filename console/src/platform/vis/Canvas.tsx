// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/vis/Canvas.css";

import { Canvas as Base } from "@synnaxlabs/pluto";
import { runtime } from "@synnaxlabs/x";
import type { PropsWithChildren, ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Session } from "@/session";

// WebKitGTK crashes when a worker draws on a canvas transferred from the page.
const HANDOFF: Base.Handoff =
  Session.Runtime.ENGINE === "tauri" && runtime.getOS() === "Linux"
    ? "bitmap"
    : "canvas";

export const Canvas = ({ children }: PropsWithChildren): ReactElement => (
  <Base.Canvas className={CSS.BE("vis", "canvas")} handoff={HANDOFF}>
    {children}
  </Base.Canvas>
);
