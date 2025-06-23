// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createMockWorkers } from "@synnaxlabs/x";
import { type FC, type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { aether } from "@/aether/aether";
import { type AetherMessage, type MainMessage } from "@/aether/message";

export const createAetherProvider = (
  registry: aether.ComponentRegistry,
): FC<PropsWithChildren> => {
  const [a, b] = createMockWorkers();
  aether.render({ comms: a.route("test"), registry });
  const worker = b.route<MainMessage, AetherMessage>("test");
  const AetherProvider = (props: PropsWithChildren): ReactElement => (
    <Aether.Provider {...props} worker={worker} workerKey="test" />
  );
  return AetherProvider;
};
