// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";

import { insecureCredentialsZ, tokenResponseZ } from "@/auth/auth";
import { Payload } from "@/user/payload";

const REGISTER_ENDPOINT = "/auth/register";

export class Client {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async register(username: string, password: string): Promise<Payload> {
    const { user: usr } = await sendRequired<
      typeof insecureCredentialsZ,
      typeof tokenResponseZ
    >(
      this.client,
      REGISTER_ENDPOINT,
      { username: username, password: password },
      insecureCredentialsZ,
      tokenResponseZ,
    );

    return usr;
  }
}
