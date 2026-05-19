#version 300 es
precision lowp float;

uniform lowp vec4 u_color;
out vec4 fragColor;

void main(void) {
  fragColor = u_color;
}
