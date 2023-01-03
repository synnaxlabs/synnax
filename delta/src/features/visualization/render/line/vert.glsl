attribute float x;
attribute float y;
attribute float translate;
attribute float a_mod;

uniform vec2 u_scale_root;
uniform vec2 u_scale;
uniform vec2 u_offset_root;
uniform vec2 u_offset;
uniform float aspect;

void main(void) {
  vec2 transform;
  if(a_mod < 0.5) {
    transform = vec2(0.0, 0.0);
  } else if (a_mod < 1.5) {
    transform = vec2(1.0, 0.0);
  } else if (a_mod < 2.5) {
    transform = vec2(0.0, -1.0*aspect*1.1);
  } else if (a_mod < 3.5){
    transform = vec2(-1.0, 0.0);
  } else if (a_mod < 4.5){
    transform = vec2(0.0, 1.0*aspect*1.1);
  }
  vec2 transformed = u_scale_root * (u_scale * vec2(x,y) + u_offset) + u_offset_root;
  transformed = transformed + transform * translate;
  gl_Position = vec4(transformed * vec2(2,2) - vec2(1,1), 0.0, 1.0);
}
