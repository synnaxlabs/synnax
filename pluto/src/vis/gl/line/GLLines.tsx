// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";

import { Box, XY } from "@synnaxlabs/x";

import { useGLContext } from "../GLCanvas/GLCanvas";

import { GLLine, LineRenderRequest } from "./renderer";

export interface GLLinesProps {
  box: Box;
  lines: GLLine[];
  clearOverscan?: XY;
}

export const GLLines = ({ box, lines, clearOverscan }: GLLinesProps): null => {
  const ctx = useGLContext();
  useEffect(() => {
    if (ctx == null) return;
    const base = ctx.registry.get<LineRenderRequest>("line");
    const scissor = ctx.scissor(base, clearOverscan);
    scissor.render(ctx, { box, lines });
    return () => scissor.clear(ctx, box);
  }, [box, lines]);
  return null;
};
