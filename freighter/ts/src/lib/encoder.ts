import { addExtension, pack, unpack } from 'msgpackr';

import { camelKeys, snakeKeys } from './caseconv';

interface CustomTypeEncoder {
  // eslint-disable-next-line @typescript-eslint/ban-types
  Class: Function;
  write(instance: unknown): unknown;
}

export interface EncoderDecoder {
  contentType: string;
  encode(payload: unknown): ArrayBuffer;
  decode(data: Uint8Array | ArrayBuffer): unknown;
}

interface StaticEncoderDecoder {
  registerCustomType(encoder: CustomTypeEncoder): void;
}

export class MsgpackEncoderDecoder implements EncoderDecoder {
  contentType = 'application/msgpack';

  encode(payload: unknown): Uint8Array {
    return pack(snakeKeys(payload));
  }

  decode(data: Uint8Array): unknown {
    return camelKeys(unpack(new Uint8Array(data)));
  }

  static registerCustomType(encoder: CustomTypeEncoder): void {
    addExtension({ type: 0, ...encoder });
  }
}

export class JSONEncoderDecoder implements EncoderDecoder {
  contentType = 'application/json';

  encode(payload: unknown): Uint8Array {
    return new TextEncoder().encode(JSON.stringify(snakeKeys(payload)));
  }

  decode(data: Uint8Array): unknown {
    return camelKeys(JSON.parse(new TextDecoder().decode(data)));
  }

  static registerCustomType(): void {
    return;
  }
}

export const ENCODERS: EncoderDecoder[] = [
  new MsgpackEncoderDecoder(),
  new JSONEncoderDecoder(),
];

export const ENCODER_CLASSES: StaticEncoderDecoder[] = [
  MsgpackEncoderDecoder,
  JSONEncoderDecoder,
];

export const registerCustomTypeEncoder = (encoder: CustomTypeEncoder): void => {
  ENCODER_CLASSES.forEach((encoderClass) => {
    encoderClass.registerCustomType(encoder);
  });
};
