// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { Canvas as Core } from "@synnaxlabs/pluto";

import "@/vis/Canvas.css";

export const Canvas = ({ children }: PropsWithChildren): ReactElement => (
  <Core.Canvas className="delta-vis__canvas">{children}</Core.Canvas>
);
