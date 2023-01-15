export const ERROR_NOT_COMPILED = new Error("webgl program executed without compiling");
export const errorUnsupported = (msg: string): Error =>
  new Error(`unsupported webgl feature: ${msg}`);
export const errorCompile = (msg: string): Error =>
  new Error(`failed to compile webgl program: ${msg}`);
export const ERROR_BAD_SHADER = new Error("null shader encountered");
