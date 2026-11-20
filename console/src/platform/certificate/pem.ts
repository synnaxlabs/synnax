// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const ARMOR = "-----BEGIN ";

const LEAF = /-----BEGIN CERTIFICATE-----([^-]+)-----END CERTIFICATE-----/;

const BASE64_LINE = /.{1,64}/g;

const decoder = new TextDecoder();

const toBase64 = (bytes: Uint8Array): string => {
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary);
};

/**
 * Reads a certificate file as PEM. A DER file becomes one PEM block.
 * @throws {Error} if the file is empty.
 */
export const readCertificate = (bytes: Uint8Array, name: string): string => {
  if (bytes.length === 0) throw new Error(`${name} is empty.`);
  const text = decoder.decode(bytes);
  if (text.includes(ARMOR)) return text;
  const lines = toBase64(bytes).match(BASE64_LINE) ?? [];
  return `-----BEGIN CERTIFICATE-----\n${lines.join("\n")}\n-----END CERTIFICATE-----\n`;
};

/**
 * Reads a private key file as PEM.
 * @throws {Error} if the file is not PEM.
 */
export const readPrivateKey = (bytes: Uint8Array, name: string): string => {
  const text = decoder.decode(bytes);
  if (!text.includes(ARMOR)) throw new Error(`${name} is not a PEM file.`);
  return text;
};

/**
 * Returns the DER bytes of the leaf, the first certificate of a PEM chain.
 * @throws {Error} if the chain holds no certificate.
 */
export const leafDER = (pem: string): Uint8Array<ArrayBuffer> => {
  const body = LEAF.exec(pem)?.[1];
  if (body == null) throw new Error("The certificate holds no PEM block.");
  return Uint8Array.from(atob(body.replace(/\s/g, "")), (c) => c.charCodeAt(0));
};
