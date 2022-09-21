import { decode, encode } from '@msgpack/msgpack';

import { camelKeys, snakeKeys } from './caseconv';

export interface EncoderDecoder {
  contentType: string;
  encode(payload: unknown): Uint8Array;
  decode(data: Uint8Array): unknown;
}

export class MsgPackEncoderDecoder implements EncoderDecoder {
  contentType = 'application/msgpack';

  encode(payload: unknown): Uint8Array {
    return encode(payload);
  }

  decode(data: Uint8Array): unknown {
    return camelKeys(decode(data));
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
}

export const ENCODERS: EncoderDecoder[] = [
  new MsgPackEncoderDecoder(),
  new JSONEncoderDecoder(),
];
