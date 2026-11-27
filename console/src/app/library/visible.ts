// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, enumeration, icd, label } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

export const useLabelsVisible = (): boolean =>
  Access.useUpdateGranted(label.TYPE_RESOURCE_ID);

export const useColorsVisible = (): boolean =>
  Access.useUpdateGranted(color.TYPE_RESOURCE_ID);

export const useEnumerationsVisible = (): boolean =>
  Access.useUpdateGranted(enumeration.TYPE_RESOURCE_ID);

export const useICDsVisible = (): boolean =>
  Access.useRetrieveGranted(icd.TYPE_RESOURCE_ID);

/** Reports whether the user can use any section of the Library. */
export const useVisible = (): boolean => {
  const labels = useLabelsVisible();
  const colors = useColorsVisible();
  const enumerations = useEnumerationsVisible();
  const icds = useICDsVisible();
  return labels || colors || enumerations || icds;
};
