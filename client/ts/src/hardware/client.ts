// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@/hardware/device";
import { type rack } from "@/hardware/rack";
import { type task } from "@/hardware/task";

export class Client {
  readonly tasks: task.Client;
  readonly racks: rack.Client;
  readonly devices: device.Client;

  constructor(tasks: task.Client, racks: rack.Client, devices: device.Client) {
    this.tasks = tasks;
    this.racks = racks;
    this.devices = devices;
  }
}
