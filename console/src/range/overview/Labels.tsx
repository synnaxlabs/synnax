// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Ranger } from "@synnaxlabs/pluto";

import { Label } from "@/label";

interface LabelsProps {
  rangeKey: string;
}

export const Labels = ({ rangeKey }: LabelsProps) => {
  const formCtx = Ranger.useSyncedLabelsForm({
    key: rangeKey,
    values: { labels: [] },
  });
  return (
    <Form.Form {...formCtx}>
      <Form.Field<string[]> required={false} path="labels">
        {({ variant: _, ...p }) => (
          <Label.SelectMultiple
            entryRenderKey="name"
            dropdownVariant="floating"
            zIndex={100}
            location="bottom"
            style={{ width: "fit-content" }}
            {...p}
          />
        )}
      </Form.Field>
    </Form.Form>
  );
};
