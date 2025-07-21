import { DisconnectedError, type effect, NotFoundError } from "@synnaxlabs/client";
import {
  Align,
  Button,
  Icon,
  Status,
  Synnax,
  Text,
  useAsyncEffect,
} from "@synnaxlabs/pluto";
import { status } from "@synnaxlabs/x";
import { useMutation, useQuery } from "@tanstack/react-query";
import { type ReactElement, useState } from "react";
import { useStore } from "react-redux";

import { useSelect } from "@/effect/selectors";
import { type Layout } from "@/layout";
import { Slate } from "@/slate";
import { translateSlateBackward } from "@/slate/types/translate";
import { type RootState } from "@/store";

export interface LoadedProps {
  effect: effect.Effect;
}

const Loaded = ({ effect }: LoadedProps): ReactElement => {
  const client = Synnax.use();
  const store = useStore<RootState>();
  const addStatus = Status.useAdder();
  const publishMut = useMutation({
    mutationFn: async () => {
      try {
        if (client == null) throw new DisconnectedError();
        const slate = Slate.select(store.getState(), effect.slate);
        console.log("slate", slate, translateSlateBackward(slate));
        await client.slates.create(translateSlateBackward(slate));
        await client.effects.create(effect);
      } catch (e) {
        console.log(e);
        addStatus(status.fromException(e));
      }
    },
  });
  return (
    <Align.Space y grow style={{ height: "100%" }}>
      <Slate.Slate
        layoutKey={effect.slate}
        visible
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
  useAsyncEffect(async () => {
    if (client == null) return;
    const observer = await client.effects.openStatusObserver();
    observer.onChange((statuses) => {
      console.log("statuses", statuses, effect.key);
      const status = statuses.find((s) => s.details.effect === effect.key);
      if (status == null) return;
      setStatus(status);
      addStatus({
        key: effect.key,
        variant: status.variant,
        message: status.message,
      });
    });
    return () => observer.close();
  }, [client]);
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
        const effect = await client.effects.retrieve(layoutKey);
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
  return <Loaded effect={res.data as effect.Effect} />;
};
