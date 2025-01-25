#version 300 es

// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

in float a_x;
in float a_y;
in vec2 a_translate;

out vec4 v_color;

uniform lowp vec2 u_scale_aggregate;
uniform lowp vec2 u_offset_aggregate;

void main(void) {
  vec2 transformed = u_scale_aggregate * vec2(a_x,a_y) + u_offset_aggregate;
  transformed = transformed + a_translate;
  gl_Position = vec4(transformed, 0.0, 1.0);
}