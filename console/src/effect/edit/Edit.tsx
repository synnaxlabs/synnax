import {
  DisconnectedError,
  type effect,
  NotFoundError,
  type slate,
} from "@synnaxlabs/client";
import { Align, Button, Icon, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useCallback, useState } from "react";
import { useStore } from "react-redux";

import { useSelect } from "@/effect/selectors";
import { Layout } from "@/layout";
import { Slate } from "@/slate";
import { translateSlateBackward } from "@/slate/types/translate";
import { type RootState } from "@/store";

export interface LoadedProps {
  effect: effect.Effect;
  layoutKey: string;
}

const Loaded = ({ effect, layoutKey }: LoadedProps): ReactElement => {
  const client = Synnax.use();
  const store = useStore<RootState>();
  const layout = Layout.useSelect(layoutKey);
  const addStatus = Status.useAdder();
  const publishMut = useMutation({
    mutationFn: async () => {
      try {
        if (client == null) throw new DisconnectedError();
        const slate = Slate.select(store.getState(), effect.slate);
        await client.slates.create(translateSlateBackward(slate));
        await client.effects.create({
          ...effect,
          name: layout?.name ?? "",
        });
      } catch (e) {
        console.log(e);
        addStatus(status.fromException(e));
      }
    },
  });

  const validate = useCallback(
    async (graph: slate.Graph) => {
      if (client == null) return null;
      try {
        await client.effects.validate(graph);
        return null;
      } catch (e) {
        return e as Error;
      }
    },
    [client],
  );
  return (
    <Align.Space y grow style={{ height: "100%" }}>
      <Slate.Slate
        layoutKey={effect.slate}
        visible
        validate={validate}
        focused={false}
        onClose={() => {}}
      />
      <Align.Space x style={{ padding: "2rem" }} justify="end" grow>
        <Align.Space
          x
          background={1}
          style={{ padding: "2rem" }}
          bordered
          borderShade={5}
          grow
          rounded={2}
          justify="spaceBetween"
        >
          <EffectState effect={effect} />
          <Button.Button startIcon={<Icon.Play />} onClick={() => publishMut.mutate()}>
            Deploy
          </Button.Button>
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};

const EffectState = ({ effect }: { effect: effect.Effect }) => {
  const client = Synnax.use();
  const [status, setStatus] = useState<effect.Status | null>(null);
  const addStatus = Status.useAdder();
  return (
    <Status.Text variant={status?.variant}>
      {status?.message ?? "Effect has not been deployed yet."}
    </Status.Text>
  );
};

export const Edit: Layout.Renderer = ({ layoutKey }) => {
  const client = Synnax.use();
  const effect = useSelect(layoutKey);
  const res = useQuery({
    queryKey: ["effect", layoutKey],
    queryFn: async () => {
      if (effect != null || client == null) return effect;
      try {
        const effect = await client.effects.retrieve({ key: layoutKey });
        return effect;
      } catch (e) {
        if (NotFoundError.matches(e)) return effect;
        throw e;
      }
    },
  });
  if (res.isLoading) return <Text.Text level="p">Loading...</Text.Text>;
  if (res.isError)
    return (
      <Align.Space y grow style={{ height: "100%" }}>
        <Status.Text.Centered variant="error">{res.error.message}</Status.Text.Centered>
      </Align.Space>
    );
  return <Loaded effect={res.data as effect.Effect} layoutKey={layoutKey} />;
};
