import { addExtension, pack, unpack } from 'msgpackr';
import { ZodSchema } from 'zod';

import { camelKeys, snakeKeys } from './caseconv';

/**
 * CustomTypeEncoder is an interface for a class that needs to transform its
 * value before encoding.
 */
interface CustomTypeEncoder {
  /** The Class the custom encoder is set for */
  // eslint-disable-next-line @typescript-eslint/ban-types
  Class: Function;

  /**
   * The function that transforms the value before encoding;
   * @param instance - The instance of the class to transform.
   * @returns The transformed value.
   */
  write<P>(instance: P): unknown;
}

/**
 * EncoderDecoder is an entity that encodes and decodes messages to and from
 * a binary format.
 */
export interface EncoderDecoder {
  /** The HTTP content type of the encoder */
  contentType: string;

  /**
   * Encodes the given payload into a binary representation.
   * @param payload - The payload to encode.
   * @returns an ArrayBuffer containing the encoded payload.
   */
  encode(payload: unknown): ArrayBuffer;

  /**
   * Decodes the given binary representation into a type checked payload.
   * @param data - The data to decode.
   * @param schema - The schema to decode the data with.
   */
  decode<P>(data: Uint8Array | ArrayBuffer, schema: ZodSchema<P>): P;
}

interface StaticEncoderDecoder {
  registerCustomType(encoder: CustomTypeEncoder): void;
}

/***
 * MsgpackEncoderDecoder is a msgpack implementation of EncoderDecoder.
 */
export class MsgpackEncoderDecoder implements EncoderDecoder {
  contentType = 'application/msgpack';

  encode(payload: unknown): ArrayBuffer {
    return pack(snakeKeys(payload));
  }

  decode<P>(data: Uint8Array, schema: ZodSchema<P>): P {
    return schema.parse(camelKeys(unpack(new Uint8Array(data))));
  }

  static registerCustomType(encoder: CustomTypeEncoder): void {
    addExtension({ type: 0, ...encoder });
  }
}

/**
 * JSONEncoderDecoder is a JSON implementation of EncoderDecoder.
 */
export class JSONEncoderDecoder implements EncoderDecoder {
  contentType = 'application/json';

  encode(payload: unknown): ArrayBuffer {
    return new TextEncoder().encode(JSON.stringify(snakeKeys(payload)));
  }

  decode<P>(data: Uint8Array, schema: ZodSchema<P>): P {
    return schema.parse(camelKeys(JSON.parse(new TextDecoder().decode(data))));
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
