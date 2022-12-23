import { DataType, TypedArray } from "../telem";

export const randomTypedArray = (length: number, dataType: DataType): TypedArray => {
  // generate random bytes of the correct length
  const bytes = new Uint8Array(length * dataType.density.valueOf());
  for (let i = 0; i < bytes.byteLength; i++) {
    bytes[i] = Math.floor(Math.random() * 256);
  }
  return new dataType.Array(bytes.buffer);
};
