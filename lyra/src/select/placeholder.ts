// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@synnaxlabs/x";

import { Input } from "@/input";

/**
 * @returns what an empty trigger reads. Beside a visible label the text prompts the
 * user to act; with no label it is the field's only name, so it reads as one.
 */
export const usePlaceholder = (resourceName: string): string =>
  Input.useLabelled() ? `Select ${resourceName}` : caseconv.capitalize(resourceName);
