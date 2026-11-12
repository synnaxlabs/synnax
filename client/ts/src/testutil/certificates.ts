// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** A self-signed P-256 certificate valid until 2126, with its private key. */
export const CERTIFICATE = {
  pem: `-----BEGIN CERTIFICATE-----
MIIBnDCCAUKgAwIBAgIUTlojETDboAh4nmfahlzAemoZ4g4wCgYIKoZIzj0EAwIw
MzEbMBkGA1UEAwwSU3lubmF4IFRlc3QgQ2xpZW50MRQwEgYDVQQKDAtTeW5uYXgg
TGFiczAgFw0yNjExMDQwMjUyMTBaGA8yMTI2MTAxMTAyNTIxMFowMzEbMBkGA1UE
AwwSU3lubmF4IFRlc3QgQ2xpZW50MRQwEgYDVQQKDAtTeW5uYXggTGFiczBZMBMG
ByqGSM49AgEGCCqGSM49AwEHA0IABAttV7/ZZaP/ck5ZSEibyz8Bq/FOha2BqRFx
KcHLLjh+GByF19Xf9tZdf+9TfDlOPwyGPp+zArW6WOjeHdzzPv6jMjAwMB0GA1Ud
DgQWBBQnJ8GnAtkFwtBOUq0b4+pWyuXWQTAPBgNVHRMBAf8EBTADAQH/MAoGCCqG
SM49BAMCA0gAMEUCIQDucqkUixw8A44mdBmPFZIhYtq3tzNMzPF5zPD9ONNZPwIg
I3WFxQbCO86E+bgOOHH1NSjmhCFtORY5jSOGw6yL3m4=
-----END CERTIFICATE-----
`,
  privateKeyPem: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgjJtOV99NCzRnmMoB
wui9Lv+9a8aqZ+LwPX2PyGfVEa+hRANCAAQLbVe/2WWj/3JOWUhIm8s/AavxToWt
gakRcSnByy44fhgchdfV3/bWXX/vU3w5Tj8Mhj6fswK1uljo3h3c8z7+
-----END PRIVATE KEY-----
`,
};
