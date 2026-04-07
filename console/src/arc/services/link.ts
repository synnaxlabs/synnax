// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@/arc";
import { translateGraphToConsole } from "@/arc/types/translate";
import { type Link } from "@/link";

export const handleLink: Link.Handler = async ({ client, key, placeLayout }) => {
  const retrieved = await client.arcs.retrieve({ key });
  const { name, text, mode } = retrieved;
  const graph = translateGraphToConsole(retrieved.graph);
  placeLayout(
    Arc.Editor.create({
      name,
      version: "1.0.0",
      key,
      type: "arc",
      remoteCreated: true,
      graph,
      text,
      mode,
    }),
  );
};
