import { label, ontology } from "@synnaxlabs/client";
import { Form, Label, Synnax } from "@synnaxlabs/pluto";
import { compare, deep, unique } from "@synnaxlabs/x";
import { z } from "zod";

const labelFormSchema = z.object({
  labels: z.array(z.string()),
});

interface LabelsProps {
  rangeKey: string;
}

export const Labels = ({ rangeKey }: LabelsProps) => {
  const otgID = new ontology.ID({ key: rangeKey, type: "range" });
  const client = Synnax.use();
  const formCtx = Form.useSynced<typeof labelFormSchema, label.Label[]>({
    name: "Labels",
    key: ["range", "labels"],
    schema: labelFormSchema,
    values: { labels: [] },
    queryFn: async ({ client }) => {
      const labels = await client.labels.retrieveFor(otgID);
      console.log(labels);
      return { labels: labels.map((l) => l.key) };
    },
    openObservable: async (client) => await client.labels.trackLabelsOf(otgID),
    applyObservable: async ({ changes, ctx }) => {
      const existing = ctx.get<string[]>("labels").value;
      const next = unique(changes.map((c) => c.key));
      if (compare.unorderedPrimitiveArrays(existing, next) === compare.EQUAL) return;
      ctx.set("labels", next);
    },
    applyChanges: async ({ client, values, prev }) => {
      const next = unique(values.labels);
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
