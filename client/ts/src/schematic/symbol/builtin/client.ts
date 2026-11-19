// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type Builtin, builtinZ } from "@/schematic/symbol/types.gen";

const retrieveReqZ = z.object({});
const retrieveResZ = z.object({ builtins: builtinZ.array() });

/** Reads the symbols that the Core ships. */
export class Client {
  private readonly unary: UnaryClient;

  constructor(unary: UnaryClient) {
    this.unary = unary;
  }

  /**
   * Returns the symbols that the Core ships, with their states and commands. Pluto
   * holds their drawings.
   */
  async retrieve(): Promise<Builtin[]> {
    const res = await this.unary.send(
      "/schematic/symbol/builtin/retrieve",
      {},
      retrieveReqZ,
      retrieveResZ,
    );
    return res.builtins;
  }
}
