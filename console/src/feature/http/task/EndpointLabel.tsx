// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/pluto";

import { Task } from "@/platform/task";

export interface EndpointLabelProps {
  epKey: string;
}

/** Names an endpoint by its method and path; an empty path reads as a new endpoint. */
export const EndpointLabel = ({ epKey }: EndpointLabelProps) => {
  const path = `config.endpoints.${epKey}`;
  const method = PForm.useFieldValue<string>(`${path}.method`);
  const epPath = PForm.useFieldValue<string>(`${path}.path`);
  return (
    <Task.Views.ItemLabel kind={method} color={epPath === "" ? 8 : 10}>
      {epPath === "" ? "New endpoint" : epPath}
    </Task.Views.ItemLabel>
  );
};
