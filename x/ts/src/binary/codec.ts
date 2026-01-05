// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { caseconv } from "@/caseconv";
import { narrow } from "@/narrow";

/**
 * Codec is an entity that encodes and decodes messages to and from a
 * binary format.
 */
export interface Codec {
  /** The HTTP content type of the encoder */
  contentType: string;

  /**
   * Encodes the given payload into a binary representation.
   *
   * @param payload - The payload to encode.
   * @returns An ArrayBuffer containing the encoded payload.
   */
  encode: (payload: unknown) => Uint8Array;

  /**
   * Decodes the given binary representation into a type checked payload.
   *
   * @param data - The data to decode.
   * @param schema - The schema to decode the data with.
   */
  decode: <P extends z.ZodType>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
  ) => z.infer<P>;
}

/** JSONCodec is a JSON implementation of Codec. */
export class JSONCodec implements Codec {
  contentType = "application/json";
  private readonly decoder: TextDecoder;
  private readonly encoder: TextEncoder;

  constructor() {
    this.decoder = new TextDecoder();
    this.encoder = new TextEncoder();
  }

  encode(payload: unknown): Uint8Array {
    return this.encoder.encode(this.encodeString(payload));
  }

  decode<P extends z.ZodType>(data: Uint8Array | ArrayBuffer, schema?: P): z.infer<P> {
    return this.decodeString(this.decoder.decode(data), schema);
  }

  decodeString<P extends z.ZodType>(data: string, schema?: P): z.infer<P> {
    const parsed = JSON.parse(data);
    const unpacked = caseconv.snakeToCamel(parsed);
    return schema != null ? schema.parse(unpacked) : (unpacked as z.infer<P>);
  }

  encodeString(payload: unknown): string {
    const caseConverted = caseconv.camelToSnake(payload);
    return JSON.stringify(caseConverted, (_, v) => {
      if (ArrayBuffer.isView(v)) return Array.from(v as Uint8Array);
      if (typeof v === "bigint") return v.toString();
      return v;
    });
  }
}

/**
 * CSVCodec is a CSV implementation of Codec.
 */
export class CSVCodec implements Codec {
  contentType = "text/csv";

  encode(payload: unknown): Uint8Array {
    const csvString = this.encodeString(payload);
    return new TextEncoder().encode(csvString);
  }

  decode<P extends z.ZodType>(data: Uint8Array | ArrayBuffer, schema?: P): z.infer<P> {
    const csvString = new TextDecoder().decode(data);
    return this.decodeString(csvString, schema);
  }

  encodeString(payload: unknown): string {
    if (!Array.isArray(payload) || payload.length === 0 || !narrow.isObject(payload[0]))
      throw new Error("Payload must be an array of objects");

    const keys = Object.keys(payload[0]);
    const csvRows = [keys.join(",")];

    payload.forEach((item) => {
      const values = keys.map((key) => JSON.stringify(item[key] ?? ""));
      csvRows.push(values.join(","));
    });

    return csvRows.join("\n");
  }

  decodeString<P extends z.ZodType>(data: string, schema?: P): z.infer<P> {
    const [headerLine, ...lines] = data
      .trim()
      .split("\n")
      .map((line) => line.trim());
    if (headerLine.length === 0)
      return schema != null ? schema.parse({}) : ({} as z.infer<P>);
    const headers = headerLine.split(",").map((header) => header.trim());
    const result: { [key: string]: unknown[] } = {};

    headers.forEach((header) => {
      result[header] = [];
    });

    lines.forEach((line) => {
      const values = line.split(",").map((value) => value.trim());
      headers.forEach((header, index) => {
        const v = this.parseValue(values[index]);
        if (v == null) return;
        result[header].push(v);
      });
    });

    return schema != null ? schema.parse(result) : (result as z.infer<P>);
  }

  private parseValue(value?: string): unknown {
    if (value == null || value.length === 0) return null;
    const num = Number(value);
    if (!isNaN(num)) return num;
    if (value.startsWith('"') && value.endsWith('"')) return value.slice(1, -1);
    return value;
  }
}

export class TextCodec implements Codec {
  contentType = "text/plain";

  encode(payload: unknown): Uint8Array {
    if (typeof payload !== "string")
      throw new Error("TextCodec.encode payload must be a string");
    return new TextEncoder().encode(payload);
  }

  decode<P extends z.ZodType>(data: Uint8Array | ArrayBuffer, schema?: P): z.infer<P> {
    const text = new TextDecoder().decode(data);
    return schema != null ? schema.parse(text) : (text as z.infer<P>);
  }
}

export const JSON_CODEC = new JSONCodec();
export const CSV_CODEC = new CSVCodec();
export const TEXT_CODEC = new TextCodec();

export const ENCODERS: Codec[] = [JSON_CODEC];
