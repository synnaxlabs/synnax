// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type NumericSinkProps,
  NumericSink,
  type AuthoritySourceProps,
  AuthoritySource,
  type AuthoritySinkProps,
  AuthoritySink,
} from "@/control/aether/controller";
import { type telem } from "@/telem/core";

export const useNumericSink = (props: NumericSinkProps): telem.NumericSinkSpec => {
  return {
    type: NumericSink.TYPE,
    props,
    variant: "numeric-sink",
  };
};

export const useAuthorityStatusSource = (
  props: AuthoritySourceProps,
): telem.StatusSourceSpec => {
  return {
    type: AuthoritySource.TYPE,
    props,
    variant: "status-source",
  };
};

export const useAuthorityColorSource = (
  props: AuthoritySourceProps,
): telem.ColorSourceSpec => {
  return {
    type: AuthoritySource.TYPE,
    props,
    variant: "color-source",
  };
};

export const useAuthoritySink = (props: AuthoritySinkProps): telem.BooleanSinkSpec => {
  return {
    type: AuthoritySink.TYPE,
    props,
    variant: "boolean-sink",
  };
};
