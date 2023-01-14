import { GeneralError, UnexpectedError } from "@synnaxlabs/client";

export const ERROR_NOT_COMPILED = new UnexpectedError(
  "webgl program executed without compiling"
);
export const errorUnsupported = (msg: string): Error =>
  new GeneralError(`unsupported webgl feature: ${msg}`);
export const errorCompile = (msg: string): Error =>
  new GeneralError(`failed to compile webgl program: ${msg}`);
export const ERROR_BAD_SHADER = new UnexpectedError("null shader encountered");
