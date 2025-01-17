// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, ranger } from "@synnaxlabs/client";
import { Form, Synnax } from "@synnaxlabs/pluto";
import { compare, unique } from "@synnaxlabs/x";
import { z } from "zod";

import { Label } from "@/label";

const labelFormSchema = z.object({
  labels: z.array(z.string()),
});

interface LabelsProps {
  rangeKey: string;
}

export const Labels = ({ rangeKey }: LabelsProps) => {
  const otgID = ranger.ontologyID(rangeKey);
  const client = Synnax.use();
  const formCtx = Form.useSynced<typeof labelFormSchema, label.Label[]>({
    name: "Labels",
    key: ["range", "labels", rangeKey],
    schema: labelFormSchema,
    values: { labels: [] },
    queryFn: async ({ client }) => {
      const labels = await client.labels.retrieveFor(otgID);
      return { labels: labels.map((l) => l.key) };
    },
    openObservable: async (client) => await client.labels.trackLabelsOf(otgID),
    applyObservable: async ({ changes, ctx }) => {
      const existing = ctx.get<string[]>("labels").value;
      const next = unique.unique(changes.map((c) => c.key));
      if (compare.unorderedPrimitiveArrays(existing, next) === compare.EQUAL) return;
      ctx.set("labels", next);
    },
    applyChanges: async ({ client, values, prev }) => {
      const next = unique.unique(values.labels);
      if (
        client == null ||
        compare.unorderedPrimitiveArrays(prev as string[], next) === compare.EQUAL
      )
        return;
      await client.labels.label(otgID, values.labels, { replace: true });
    },
  });

  return (
    <Form.Form {...formCtx}>
      <Form.Field<string> required={false} path="labels">
        {(p) => (
          <Label.SelectMultiple
            searcher={client?.labels}
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
