// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Runs Pluto's line shaders on nanosecond timestamps through transform feedback. Posts
// "precision ok", or each position that is not finite or not as expected.

// A 30 s window maps nanoseconds to clip space.
const SCALE = [2 / 30e9, 0.5];
const OFFSET = [-1, -0.75];
const TRANSLATE = [0.01, -0.02];
const X = [1e3, 5e6, 1e10];
const Y = [1, 2, 3];
const TOLERANCE = 1e-5;

const compile = (gl, type, source) => {
  const shader = gl.createShader(type);
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS))
    throw new Error(gl.getShaderInfoLog(shader));
  return shader;
};

const attribute = (gl, prog, name, data, integer) => {
  const loc = gl.getAttribLocation(prog, name);
  gl.bindBuffer(gl.ARRAY_BUFFER, gl.createBuffer());
  gl.bufferData(gl.ARRAY_BUFFER, data, gl.STATIC_DRAW);
  if (integer) gl.vertexAttribIPointer(loc, 1, gl.UNSIGNED_BYTE, 0, 0);
  else gl.vertexAttribPointer(loc, 1, gl.FLOAT, false, 0, 0);
  gl.enableVertexAttribArray(loc);
};

// Returns the clip space positions the vertex shader in file computes.
const positions = async (gl, file, integerY) => {
  const [vert, frag] = await Promise.all(
    [file, "frag.glsl"].map(async (f) => (await fetch(f)).text()),
  );
  const prog = gl.createProgram();
  gl.attachShader(prog, compile(gl, gl.VERTEX_SHADER, vert));
  gl.attachShader(prog, compile(gl, gl.FRAGMENT_SHADER, frag));
  gl.transformFeedbackVaryings(prog, ["gl_Position"], gl.INTERLEAVED_ATTRIBS);
  gl.linkProgram(prog);
  if (!gl.getProgramParameter(prog, gl.LINK_STATUS))
    throw new Error(gl.getProgramInfoLog(prog));
  gl.useProgram(prog);
  gl.bindVertexArray(gl.createVertexArray());
  gl.uniform2f(gl.getUniformLocation(prog, "u_scale_aggregate"), ...SCALE);
  gl.uniform2f(gl.getUniformLocation(prog, "u_offset_aggregate"), ...OFFSET);
  gl.vertexAttrib2f(gl.getAttribLocation(prog, "a_translate"), ...TRANSLATE);
  attribute(gl, prog, "a_x", new Float32Array(X), false);
  attribute(
    gl,
    prog,
    "a_y",
    integerY ? new Uint8Array(Y) : new Float32Array(Y),
    integerY,
  );

  const out = gl.createBuffer();
  gl.bindBuffer(gl.ARRAY_BUFFER, out);
  gl.bufferData(gl.ARRAY_BUFFER, X.length * 16, gl.STATIC_READ);
  gl.bindBuffer(gl.ARRAY_BUFFER, null);
  gl.bindTransformFeedback(gl.TRANSFORM_FEEDBACK, gl.createTransformFeedback());
  gl.bindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, out);
  gl.enable(gl.RASTERIZER_DISCARD);
  gl.beginTransformFeedback(gl.POINTS);
  gl.drawArrays(gl.POINTS, 0, X.length);
  gl.endTransformFeedback();
  gl.disable(gl.RASTERIZER_DISCARD);
  gl.bindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, null);
  gl.bindTransformFeedback(gl.TRANSFORM_FEEDBACK, null);
  const result = new Float32Array(X.length * 4);
  gl.bindBuffer(gl.ARRAY_BUFFER, out);
  gl.getBufferSubData(gl.ARRAY_BUFFER, 0, result);
  return result;
};

const check = async (gl, file, integerY) => {
  const result = await positions(gl, file, integerY);
  const failures = [];
  for (let i = 0; i < X.length; i++) {
    const want = [
      SCALE[0] * X[i] + OFFSET[0] + TRANSLATE[0],
      SCALE[1] * Y[i] + OFFSET[1] + TRANSLATE[1],
    ];
    for (let d = 0; d < 2; d++) {
      const got = result[i * 4 + d];
      if (!Number.isFinite(got) || Math.abs(got - want[d]) > TOLERANCE)
        failures.push(`${file} x=${X[i]} axis ${d}: got ${got}, want ${want[d]}`);
    }
  }
  return failures;
};

(async () => {
  const gl = new OffscreenCanvas(1, 1).getContext("webgl2");
  if (gl == null) return postMessage("no webgl");
  const failures = [
    ...(await check(gl, "vert_f32.glsl", false)),
    ...(await check(gl, "vert_hybrid.glsl", true)),
  ];
  postMessage(failures.length === 0 ? "precision ok" : failures.join("; "));
})().catch((e) => postMessage(`error ${e.message}`));
