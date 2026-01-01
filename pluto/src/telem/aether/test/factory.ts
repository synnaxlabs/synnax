// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";

import { type CreateOptions, type Factory } from "@/telem/aether/factory";
import { type Spec, type Telem } from "@/telem/aether/telem";

const instances = new Map<string, Telem>();

export const registerInstance = (
  key: string,
  instance: Telem,
): destructor.Destructor => {
  instances.set(key, instance);
  return () => instances.delete(key);
};

export const TEST_SINK_TYPE = "test-sink";
export const TEST_SOURCE_TYPE = "test-source";

export class TestFactory implements Factory {
  type = "test";

  create(spec: Spec, _options?: CreateOptions): Telem | null {
    if (spec.type !== TEST_SINK_TYPE && spec.type !== TEST_SOURCE_TYPE) return null;
    if (spec.props?.testId != null) {
      const instance = instances.get(spec.props.testId);
      if (instance) return instance;
    }
    return null;
  }
}
