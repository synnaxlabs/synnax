// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { z } from "zod";

import { caseconv } from "@/caseconv";
import { csv } from "@/csv";
import { narrow } from "@/narrow";
import { zod } from "@/zod";

/** Codec is an entity that encodes and decodes messages to and from a binary format. */
export interface Codec {
  /** The HTTP content type of the encoder */
  contentType: string;

  /**
   * Encodes the given payload into a binary representation.
   * @returns An ArrayBuffer containing the encoded payload.
   */
  encode: (payload: unknown, schema?: z.ZodType) => Uint8Array<ArrayBuffer>;

  /**
   * Decodes the given binary representation into a type checked payload.
   * @param opts - Options forwarded to schema parsing, e.g. a label naming the value in
   * parse error messages.
   */
  decode: <P extends z.ZodType>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
    opts?: zod.ParseOptions,
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

  encode(payload: unknown, schema?: z.ZodType): Uint8Array<ArrayBuffer> {
    return this.encoder.encode(this.encodeString(payload, schema));
  }

  decode<P extends z.ZodType>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
    opts?: zod.ParseOptions,
  ): z.infer<P> {
    return this.decodeString(this.decoder.decode(data), schema, opts);
  }

  decodeString<P extends z.ZodType>(
    data: string,
    schema?: P,
    opts?: zod.ParseOptions,
  ): z.infer<P> {
    const parsed = JSON.parse(data);
    const unpacked = caseconv.snakeToCamel(parsed, { schema });
    return schema != null
      ? zod.parse(schema, unpacked, opts)
      : (unpacked as z.infer<P>);
  }

  encodeString(payload: unknown, schema?: z.ZodType): string {
    const parsed = schema != null ? zod.parse(schema, payload) : payload;
    const caseConverted = caseconv.camelToSnake(parsed ?? {}, { schema });
    return JSON.stringify(caseConverted, (_, v) => {
      if (ArrayBuffer.isView(v)) return Array.from(v as Uint8Array);
      if (typeof v === "bigint") return v.toString();
      return v;
    });
  }
}

// An empty cell decodes to null so that every column keeps one entry per row.
const parseCSVField = (field: string | undefined): unknown => {
  if (field == null || field.length === 0) return null;
  const num = Number(field);
  return isNaN(num) || field.trim().length === 0 ? field : num;
};

/**
 * CSVCodec is a CSV implementation of Codec. It encodes an array of objects into rows,
 * and decodes rows into a record of columns keyed by the header row.
 */
export class CSVCodec implements Codec {
  contentType = "text/csv";

  encode(payload: unknown): Uint8Array<ArrayBuffer> {
    if (!Array.isArray(payload) || payload.length === 0 || !narrow.isObject(payload[0]))
      throw new Error("Payload must be an array of objects");
    const keys = Object.keys(payload[0]);
    const rows = [keys, ...payload.map((item) => keys.map((key) => item[key]))];
    const text = rows.map((row) => row.map(csv.formatValue).join(",")).join("\n");
    return new TextEncoder().encode(text);
  }

  decode<P extends z.ZodType>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
    opts?: zod.ParseOptions,
  ): z.infer<P> {
    return this.decodeString(new TextDecoder().decode(data), schema, opts);
  }

  decodeString<P extends z.ZodType>(
    data: string,
    schema?: P,
    opts?: zod.ParseOptions,
  ): z.infer<P> {
    const [headers = [], ...rows] = csv.parseBlock(data, ",");
    const result: Record<string, unknown[]> = {};
    headers.forEach((header, i) => {
      result[header] = rows.map((row) => parseCSVField(row[i]));
    });
    return schema != null
      ? zod.parse(schema, result, { label: opts?.label ?? "CSV data" })
      : (result as z.infer<P>);
  }
}

export class TextCodec implements Codec {
  contentType = "text/plain";

  encode(payload: unknown): Uint8Array<ArrayBuffer> {
    if (typeof payload !== "string")
      throw new Error("TextCodec.encode payload must be a string");
    return new TextEncoder().encode(payload);
  }

  decode<P extends z.ZodType>(
    data: Uint8Array | ArrayBuffer,
    schema?: P,
    opts?: zod.ParseOptions,
  ): z.infer<P> {
    const text = new TextDecoder().decode(data);
    return schema != null
      ? zod.parse(schema, text, { label: opts?.label ?? "text data" })
      : (text as z.infer<P>);
  }
}

export const JSON_CODEC = new JSONCodec();
export const CSV_CODEC = new CSVCodec();
export const TEXT_CODEC = new TextCodec();
