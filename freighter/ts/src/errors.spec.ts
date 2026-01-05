// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { EOF, FreighterError, StreamClosed, Unreachable } from "@/errors";

describe("errors", () => {
  test("encoding and decoding freighter errors", () => {
    [new EOF(), new StreamClosed(), new Unreachable()].forEach((error) => {
      const encoded = errors.encode(error);
      expect(encoded.type.startsWith(FreighterError.TYPE)).toBeTruthy();
      expect(encoded.data).toEqual(error.message);
      const decoded = errors.decode(encoded);
      expect(error.matches(decoded)).toBeTruthy();
    });
  });
});
