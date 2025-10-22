// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";

import { policy } from "@/access/policy";
import { role } from "@/access/role";

export class Client {
  readonly policies: policy.Client;
  readonly roles: role.Client;

  constructor(client: UnaryClient) {
    this.policies = new policy.Client(client);
    this.roles = new role.Client(client);
  }
}
