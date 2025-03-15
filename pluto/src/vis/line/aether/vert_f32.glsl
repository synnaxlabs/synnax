#version 300 es

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

in mediump float a_x;
in mediump float a_y;
in mediump vec2 a_translate;

uniform mediump vec2 u_scale_aggregate;
uniform mediump vec2 u_offset_aggregate;

void main(void) {
  gl_Position = vec4((u_scale_aggregate * vec2(a_x, a_y) + u_offset_aggregate) + a_translate, 0.0, 1.0);
}
