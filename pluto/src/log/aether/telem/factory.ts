// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NoopLogSource } from "@/log/aether/telem/noop";
import { StreamMultiChannelLog } from "@/log/aether/telem/sources";
import { telem } from "@/telem/aether";

export class LogFactory implements telem.Factory {
  type = "log";
  private readonly client: telem.client.Client;

  constructor(cl: telem.client.Client) {
    this.client = cl;
  }

  create(spec: telem.Spec, options?: telem.CreateOptions): telem.Telem | null {
    if (spec.type === NoopLogSource.TYPE) return new NoopLogSource();
    if (spec.type === StreamMultiChannelLog.TYPE)
      return new StreamMultiChannelLog(this.client, spec.props, options);
    return null;
  }
}
