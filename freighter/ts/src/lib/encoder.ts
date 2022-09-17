import { decode, encode } from '@msgpack/msgpack';

import { camelKeys, snakeKeys } from './caseconv';
import { Payload } from './transport';

export interface EncoderDecoder {
  contentType: string;
  encode(data: Payload): Uint8Array;
  decode<A>(data: Uint8Array): A;
}

export class MsgPackEncoderDecoder implements EncoderDecoder {
  contentType = 'application/msgpack';

  encode(payload: unknown): Uint8Array {
    return encode(payload);
  }

  decode<A>(data: Uint8Array): A {
    return decode(data) as unknown as A;
  }
}

export class JSONEncoderDecoder implements EncoderDecoder {
  contentType = 'application/json';

  encode(payload: unknown): Uint8Array {
    return new TextEncoder().encode(JSON.stringify(snakeKeys(payload)));
  }

  decode<A>(data: Uint8Array): A {
    return camelKeys(
      JSON.parse(new TextDecoder().decode(data))
    ) as unknown as A;
  }
}

export const ENCODERS: EncoderDecoder[] = [
  new MsgPackEncoderDecoder(),
  new JSONEncoderDecoder(),
];
