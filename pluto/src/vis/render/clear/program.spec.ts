// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Program } from "@/vis/render/clear/program";
import { type Context } from "@/vis/render/context";
import { canvasTest } from "@/vis/render/test";

const create = () => {
  const { gl, calls } = canvasTest.createGL();
  const prog = new Program({ gl } as unknown as Context);
  const ops = (op: string) => calls.filter((c) => c.op === op);
  return { prog, calls, ops };
};

describe("clear.Program", () => {
  it("should create one vertex array object across many draws", () => {
    const { prog, ops } = create();
    for (let i = 0; i < 100; i++) prog.exec();
    expect(ops("createVertexArray")).toHaveLength(1);
    expect(ops("drawArrays")).toHaveLength(100);
  });

  it("should restore the default vertex array after each draw", () => {
    const { prog, calls } = create();
    calls.length = 0;
    prog.exec();
    const seq = calls
      .filter((c) => c.op === "bindVertexArray" || c.op === "drawArrays")
      .map((c) =>
        c.op === "drawArrays" ? "draw" : c.args[0] === null ? "unbind" : "bind",
      );
    expect(seq).toEqual(["bind", "draw", "unbind"]);
  });
});
