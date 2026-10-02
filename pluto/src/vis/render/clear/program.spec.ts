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

const { GL_ENUMS } = canvasTest;
const PROG = { name: "program" };
const BUFFER = { name: "buffer" };
const VAO = { name: "vao" };
const POSITION_LOC = 3;

const create = (returns: Record<string, () => unknown> = {}) => {
  const { gl, calls } = canvasTest.createGL({
    createProgram: () => PROG,
    createBuffer: () => BUFFER,
    createVertexArray: () => VAO,
    getAttribLocation: () => POSITION_LOC,
    ...returns,
  });
  const prog = new Program({ gl } as unknown as Context);
  return { prog, calls };
};

describe("clear.Program", () => {
  describe("constructor", () => {
    it("should store the positions and attribute layout in its vertex array", () => {
      const { calls } = create();
      const setup = calls.slice(calls.findIndex((c) => c.op === "createBuffer"));
      expect(setup).toEqual([
        { op: "createBuffer", args: [] },
        { op: "createVertexArray", args: [] },
        { op: "bindVertexArray", args: [VAO] },
        { op: "bindBuffer", args: [GL_ENUMS.ARRAY_BUFFER, BUFFER] },
        {
          op: "bufferData",
          args: [GL_ENUMS.ARRAY_BUFFER, new Float32Array(6), GL_ENUMS.STATIC_DRAW],
        },
        { op: "getAttribLocation", args: [PROG, "a_position"] },
        { op: "enableVertexAttribArray", args: [POSITION_LOC] },
        {
          op: "vertexAttribPointer",
          args: [POSITION_LOC, 2, GL_ENUMS.FLOAT, false, 0, 0],
        },
        { op: "bindVertexArray", args: [null] },
      ]);
    });

    it("should throw when the buffer cannot be created", () => {
      expect(() => create({ createBuffer: () => null })).toThrow(
        "failed to create buffer",
      );
    });

    it("should throw when the vertex array cannot be created", () => {
      expect(() => create({ createVertexArray: () => null })).toThrow(
        "failed to create vertex array object",
      );
    });
  });

  describe("exec", () => {
    it("should draw one triangle from its vertex array and then unbind it", () => {
      const { prog, calls } = create();
      calls.length = 0;
      prog.exec();
      expect(calls).toEqual([
        { op: "bindVertexArray", args: [VAO] },
        { op: "useProgram", args: [PROG] },
        { op: "drawArrays", args: [GL_ENUMS.TRIANGLES, 0, 3] },
        { op: "bindVertexArray", args: [null] },
      ]);
    });

    it("should not create a vertex array when drawing", () => {
      const { prog, calls } = create();
      calls.length = 0;
      for (let i = 0; i < 100; i++) prog.exec();
      expect(calls.filter((c) => c.op === "createVertexArray")).toHaveLength(0);
      expect(calls.filter((c) => c.op === "drawArrays")).toHaveLength(100);
    });
  });
});
