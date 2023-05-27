attribute float a_x;
attribute float a_y;
attribute vec2 a_translate;

uniform vec2 u_scale_viewport;
uniform vec2 u_offset_viewport;
uniform vec2 u_scale;
uniform vec2 u_offset;

void main(void) {
  vec2 transformed = u_scale_viewport * (u_scale * vec2(a_x,a_y) + u_offset) + u_offset_viewport;
  transformed = transformed + a_translate;
  gl_Position = vec4(transformed * vec2(2,2) - vec2(1,1), 0.0, 1.0);
}
