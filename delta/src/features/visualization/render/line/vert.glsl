attribute float a_x;
attribute float a_y;
attribute vec2 a_translate;

uniform vec2 u_scale_root;
uniform vec2 u_scale;
uniform vec2 u_offset_root;
uniform vec2 u_offset;

void main(void) {
//     transform = vec2(0.0, 1.0*aspect*1.1);
//   } else if (a_mod < 1.5) {
//     transform = vec2(0.0, -1.0*aspect*1.1);
//   } else if (a_mod < 2.5) {
//     transform = vec2(1.0, 0.0);
//   } else if (a_mod < 3.5){
//     transform = vec2(-1.0, 0.0);
//   }
  vec2 transformed = u_scale_root * (u_scale * vec2(a_x,a_y) + u_offset) + u_offset_root;
  transformed = transformed + a_translate;
  gl_Position = vec4(transformed * vec2(2,2) - vec2(1,1), 0.0, 1.0);
}
