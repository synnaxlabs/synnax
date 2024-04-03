// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { pack, unpack } from "msgpackr";
import { type ZodSchema, type z } from "zod";

import { Case } from "@/case";
import { isObject } from "@/identity";

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
}

/** JSONEncoderDecoder is a JSON implementation of EncoderDecoder. */
export class JSONEncoderDecoder implements EncoderDecoder {
  contentType = "application/json";
  readonly decoder: TextDecoder;

  constructor() {
    this.decoder = new TextDecoder();
  }

  encode(payload: unknown): ArrayBuffer {
    const json = JSON.stringify(Case.toSnake(payload), (_, v) => {
      if (ArrayBuffer.isView(v)) return Array.from(v as Uint8Array);
      if (isObject(v) && "encode_value" in v) {
        if (typeof v.value === "bigint") return v.value.toString();
        return v.value;
      }
      if (typeof v === "bigint") return v.toString();
      return v;
    });
    return new TextEncoder().encode(json);
  }

  decode<P extends z.ZodTypeAny>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
  ): z.output<P> {
    const unpacked = Case.toCamel(JSON.parse(this.decoder.decode(data)));
    return schema != null ? schema.parse(unpacked) : (unpacked as P);
  }

  static registerCustomType(): void {}
}

export const ENCODERS: EncoderDecoder[] = [
  new MsgpackEncoderDecoder(),
  new JSONEncoderDecoder(),
];
