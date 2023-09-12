// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { addExtension, pack, unpack } from "msgpackr";
import { type ZodSchema, type z } from "zod";

import { Case } from "@/case";

/**
 * CustomTypeEncoder is an interface for a class that needs to transform its
 * value before encoding.
 */
interface CustomTypeEncoder {
  /** The Class the custom encoder is set for */
  Class: Function;

  /**
   * The function that transforms the value before encoding;
   *
   * @param instance - The instance of the class to transform.
   * @returns The transformed value.
   */
  write: <P>(instance: P) => unknown;
}

/**
 * EncoderDecoder is an entity that encodes and decodes messages to and from a
 * binary format.
 */
export interface EncoderDecoder {
  /** The HTTP content type of the encoder */
  contentType: string;

  /**
   * Encodes the given payload into a binary representation.
   *
   * @param payload - The payload to encode.
   * @returns An ArrayBuffer containing the encoded payload.
   */
  encode: (payload: unknown) => ArrayBuffer;

  /**
   * Decodes the given binary representation into a type checked payload.
   *
   * @param data - The data to decode.
   * @param schema - The schema to decode the data with.
   */
  decode: <P>(data: Uint8Array | ArrayBuffer, schema?: ZodSchema<P>) => P;
}

interface StaticEncoderDecoder {
  registerCustomType: (encoder: CustomTypeEncoder) => void;
}

/** MsgpackEncoderDecoder is a msgpack implementation of EncoderDecoder. */
export class MsgpackEncoderDecoder implements EncoderDecoder {
  contentType = "application/msgpack";

  encode(payload: unknown): ArrayBuffer {
    return pack(Case.toSnake(payload));
  }

  decode<P extends z.ZodTypeAny>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
  ): z.output<P> {
    const unpacked = Case.toCamel(unpack(new Uint8Array(data)));
    return schema != null ? schema.parse(unpacked) : (unpacked as P);
  }

  static registerCustomType(encoder: CustomTypeEncoder): void {
    addExtension({ type: 0, ...encoder });
  }
}

/** JSONEncoderDecoder is a JSON implementation of EncoderDecoder. */
export class JSONEncoderDecoder implements EncoderDecoder {
  contentType = "application/json";

  encode(payload: unknown): ArrayBuffer {
    const json = JSON.stringify(Case.toSnake(payload), (_, v) => {
      if (ArrayBuffer.isView(v)) return Array.from(v as Uint8Array);
      return v;
    });
    return new TextEncoder().encode(json);
  }

  decode<P extends z.ZodTypeAny>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
  ): z.output<P> {
    const unpacked = Case.toCamel(JSON.parse(new TextDecoder().decode(data)));
    return schema != null ? schema.parse(unpacked) : (unpacked as P);
  }

  static registerCustomType(): void {}
}

export const ENCODERS: EncoderDecoder[] = [
  new MsgpackEncoderDecoder(),
  new JSONEncoderDecoder(),
];

export const ENCODER_CLASSES: StaticEncoderDecoder[] = [
  MsgpackEncoderDecoder,
  JSONEncoderDecoder,
];

export const registerCustomTypeEncoder = (encoder: CustomTypeEncoder): void =>
  ENCODER_CLASSES.forEach((encoderClass) => {
    encoderClass.registerCustomType(encoder);
  });
