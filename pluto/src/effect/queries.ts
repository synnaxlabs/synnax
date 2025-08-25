import { effect, label, ontology } from "@synnaxlabs/client";
import z from "zod";

import { Flux } from "@/flux";
import { Label } from "@/label";

export const FLUX_STORE_KEY = "effects";

export interface FluxStore extends Flux.UnaryStore<effect.Key, effect.Effect> {}

interface SubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  relationships: Flux.UnaryStore<string, ontology.Relationship>;
  labels: Label.FluxStore;
}

const SET_EFFECT_LISTENER: Flux.ChannelListener<SubStore, typeof effect.effectZ> = {
  channel: effect.SET_CHANNEL_NAME,
  schema: effect.effectZ,
  onChange: ({ store, changed }) => store.effects.set(changed.key, changed),
};

const DELETE_EFFECT_LISTENER: Flux.ChannelListener<SubStore, typeof effect.keyZ> = {
  channel: effect.DELETE_CHANNEL_NAME,
  schema: effect.keyZ,
  onChange: ({ store, changed }) => store.effects.delete(changed),
};

const STATUS_LISTENER: Flux.ChannelListener<SubStore, typeof effect.statusZ> = {
  channel: effect.STATUS_CHANNEL_NAME,
  schema: effect.statusZ,
  onChange: ({ store, changed }) =>
    store.effects.set(changed.details.effect, (prev) => {
      if (prev == null) return prev;
      return { ...prev, status: changed };
    }),
};

export const STORE_CONFIG: Flux.UnaryStoreConfig<SubStore, effect.Key, effect.Effect> =
  {
    listeners: [SET_EFFECT_LISTENER, DELETE_EFFECT_LISTENER, STATUS_LISTENER],
  };

export interface RetrieveArgs {
  key: effect.Key;
}

export const retrieve = Flux.createRetrieve<RetrieveArgs, effect.Effect, SubStore>({
  name: "Effect",
  retrieve: async ({ client, params: { key }, store }) => {
    let e = store.effects.get(key);
    if (e == null) {
      e = await client.effects.retrieve({ key, includeLabels: true });
      store.effects.set(e.key, e);
      if (e.labels) e.labels.forEach((l) => store.labels.set(l.key, l));
    }
    return e;
  },
  mountListeners: ({ store, onChange, params: { key } }) =>
    store.effects.onSet((e) => {
      if (e.key === key) onChange(e);
    }, key),
});

export interface ListParams extends effect.RetrieveRequest {}

const DEFAULT_LIST_PARAMS: Partial<effect.RetrieveRequest> = {
  includeLabels: true,
  includeStatus: true,
};

export const useList = Flux.createList<ListParams, effect.Key, effect.Effect, SubStore>(
  {
    name: "Effects",
    retrieve: async ({ client, params, store }) => {
      const effects = await client.effects.retrieve({
        ...DEFAULT_LIST_PARAMS,
        ...params,
      });
      store.effects.set(effects);
      // Store labels as well
      effects.forEach((e) => {
        if (e.labels) e.labels.forEach((l) => store.labels.set(l.key, l));
      });
      return effects;
    },
    retrieveByKey: async ({ client, key, store }) => {
      const effect = await client.effects.retrieve({ key, includeLabels: true });
      store.effects.set(effect.key, effect);
      if (effect.labels) effect.labels.forEach((l) => store.labels.set(l.key, l));

      return effect;
    },
    mountListeners: ({ store, onChange, onDelete }) => [
      store.effects.onSet((effect) => onChange(effect.key, effect)),
      store.effects.onDelete((key) => onDelete(key)),
    ],
  },
);

export const formSchema = z.object({
  key: effect.keyZ.optional(),
  name: z.string(),
  enabled: z.boolean(),
  slate: z.string(),
  labels: z.array(label.keyZ),
});

export interface FormParams {
  key?: effect.Key;
}

const effectToFormValues = (e: effect.Effect): z.infer<typeof formSchema> => ({
  key: e.key,
  name: e.name,
  enabled: e.enabled,
  slate: e.slate,
  labels: (e.labels ?? []).map((l) => l.key),
});

const ZERO_FORM_VALUES: z.infer<typeof formSchema> = {
  name: "",
  enabled: false,
  slate: "",
  labels: [],
};

export const useForm = Flux.createForm<FormParams, typeof formSchema, SubStore>({
  name: "Effect",
  schema: formSchema,
  initialValues: ZERO_FORM_VALUES,
  retrieve: async ({ client, params: { key }, store, reset }) => {
    if (key == null) return;
    const e = await client.effects.retrieve({ key, includeLabels: true });
    store.effects.set(e.key, e);
    if (e.labels) e.labels.forEach((l) => store.labels.set(l.key, l));

    reset(effectToFormValues(e));
  },
  update: async ({ client, value, store, reset }) => {
    const { labels, ...effectData } = value();
    const createdEffect = await client.effects.create(effectData);
    store.effects.set(createdEffect.key, createdEffect);

    if (labels.length > 0)
      await client.labels.label(effect.ontologyID(createdEffect.key), labels, {
        replace: true,
      });

    reset({
      ...effectToFormValues(createdEffect),
      labels,
    });
  },
  mountListeners: ({ store, params: { key }, reset, set, get }) => {
    if (key == null) return [];
    return [
      store.effects.onSet((e) => {
        if (e.key !== key) return;
        reset(effectToFormValues(e));
      }, key),
      store.relationships.onSet((rel) => {
        const otgID = effect.ontologyID(key);
        const isLabelChange = Label.matchRelationship(rel, otgID);
        if (isLabelChange) {
          const currentLabels = get<label.Key[]>("labels").value;
          set("labels", [...currentLabels.filter((l) => l !== rel.to.key), rel.to.key]);
        }
      }),
      store.relationships.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        if (!Label.matchRelationship(rel, effect.ontologyID(key))) return;
        const currentLabels = get<label.Key[]>("labels").value;
        set(
          "labels",
          currentLabels.filter((l) => l !== rel.to.key),
        );
      }),
    ];
  },
});

export interface DeleteParams {
  keys: effect.Key | effect.Key[];
}

export const useDelete = Flux.createUpdate<DeleteParams, void, SubStore>({
  name: "Effect",
  update: async ({ client, params: { keys }, store }) => {
    await client.effects.delete(keys);
    const keyArray = Array.isArray(keys) ? keys : [keys];
    keyArray.forEach((key) => store.effects.delete(key));
  },
});
