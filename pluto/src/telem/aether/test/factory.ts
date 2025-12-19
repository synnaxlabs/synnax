// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CreateOptions, type Factory } from "@/telem/aether/factory";
import { type Spec, type Telem } from "@/telem/aether/telem";

// Global registry for test instances - allows specs to reference pre-created instances
const instances = new Map<string, Telem>();

export const registerTestInstance = (id: string, instance: Telem): void => {
  instances.set(id, instance);
};

export const unregisterTestInstance = (id: string): void => {
  instances.delete(id);
};

export const TEST_SINK_TYPE = "test-sink";
export const TEST_SOURCE_TYPE = "test-source";

export class TestFactory implements Factory {
  type = "test";

  create(spec: Spec, _options?: CreateOptions): Telem | null {
    if (spec.props?.testId) {
      const instance = instances.get(spec.props.testId);
      if (instance) return instance;
    }
    return null;
  }
}
