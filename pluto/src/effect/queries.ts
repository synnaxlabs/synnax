import { effect, label, ontology, slate } from "@synnaxlabs/client";
import z from "zod";

import { Flux } from "@/flux";
import { Label } from "@/label";

export interface ListParams extends effect.RetrieveRequest {}

const handleLabelRelationshipSet: Flux.ListenerHandler<
  ontology.Relationship,
  Flux.ListListenerExtraArgs<{}, string, effect.Effect>
> = async ({ changed, onChange, client }) => {
  const isLabel = ontology.matchRelationship(changed, {
    from: { type: effect.ONTOLOGY_TYPE },
    type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
    to: { type: label.ONTOLOGY_TYPE },
  });
  if (isLabel) {
    const label = await client.labels.retrieve({ key: changed.to.key });
    onChange(changed.from.key, (prev) => {
      if (prev == null) return prev;
      return {
        ...prev,
        labels: [...(prev.labels ?? []), label],
      };
    });
  }
};

const DEFAULT_LIST_PARAMS: Partial<effect.RetrieveRequest> = {
  includeLabels: true,
  includeStatus: true,
};

export const useList = Flux.createList<ListParams, effect.Key, effect.Effect>({
  name: "Effects",
  retrieve: async ({ client, params }) =>
    await client.effects.retrieve({
      ...DEFAULT_LIST_PARAMS,
      ...params,
    }),
  retrieveByKey: async ({ client, key }) =>
    await client.effects.retrieve({ key, includeLabels: true }),
  listeners: [
    {
      channel: effect.SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(effect.effectZ, async ({ changed, onChange }) =>
        onChange(changed.key, changed),
      ),
    },
    {
      channel: effect.DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(effect.keyZ, async ({ changed, onDelete }) =>
        onDelete(changed),
      ),
    },
    {
      channel: effect.STATUS_CHANNEL_NAME,
      onChange: Flux.parsedHandler(effect.statusZ, async ({ changed, onChange }) =>
        onChange(changed.details.effect, (prev) => {
          if (prev == null) return prev;
          return { ...prev, status: changed };
        }),
      ),
    },
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(ontology.relationshipZ, async (args) => {
        await handleLabelRelationshipSet(args);
      }),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange }) => {
          if (changed.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE)
            return onChange(changed.from.key, (prev) => {
              if (prev == null) return prev;
              return {
                ...prev,
                labels: (prev.labels ?? []).filter((l) => l.key !== changed.to.key),
              };
            });
        },
      ),
    },
  ],
});

export const formSchema = z.object({
  key: effect.keyZ,
  name: z.string(),
  enabled: z.boolean(),
  slate: slate.keyZ,
  labels: z.array(label.keyZ),
});

export interface FormParams {
  key?: effect.Key;
}

const effectToFormValues = (effect: effect.Effect): z.infer<typeof formSchema> => ({
  key: effect.key,
  name: effect.name,
  enabled: effect.enabled,
  slate: effect.slate,
  labels: (effect.labels ?? []).map((l) => l.key),
});

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  name: "",
  key: "",
  enabled: false,
  slate: "",
  labels: [],
};

export const useForm = Flux.createForm<FormParams, typeof formSchema>({
  name: "Effect",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, params: { key } }) => {
    if (key == null) return null;
    const effect = await client.effects.retrieve({ key, includeLabels: true });
    return effectToFormValues(effect);
  },
  update: async ({ client, value, onChange }) => {
    const { labels, ...effectData } = value;
    const createdEffect = await client.effects.create(effectData);
    await client.labels.label(effect.ontologyID(createdEffect.key), labels, {
      replace: true,
    });
    onChange({
      ...effectToFormValues(createdEffect),
      labels,
    });
  },
  listeners: [
    {
      channel: effect.SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        effect.effectZ,
        async ({ changed, onChange, params: { key } }) => {
          if (key != null && changed.key === key) onChange(effectToFormValues(changed));
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange }) => {
          onChange((prev) => {
            if (prev == null || prev.key == null) return prev;
            const otgID = effect.ontologyID(prev.key);
            const isLabelChange = Label.matchRelationship(changed, otgID);
            if (isLabelChange)
              return {
                ...prev,
                labels: [
                  ...prev.labels.filter((l) => l !== changed.to.key),
                  changed.to.key,
                ],
              };
            return prev;
          });
        },
      ),
    },
    {
      channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
      onChange: Flux.parsedHandler(
        ontology.relationshipZ,
        async ({ changed, onChange, params: { key } }) => {
          if (key == null || !Label.matchRelationship(changed, effect.ontologyID(key)))
            return;
          onChange((prev) => {
            if (prev == null) return prev;
            const nextLabels = prev.labels.filter((l) => l !== changed.to.key);
            return { ...prev, labels: nextLabels };
          });
        },
      ),
    },
  ],
});
