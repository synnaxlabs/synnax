import { array, type Destructor, type MultiSeries } from "@synnaxlabs/x";
import z from "zod";

import { aether, synnax } from "@/ether";
import { type flux } from "@/flux/aether";
import { type ListenerSpec } from "@/flux/aether/listeners";
import { Streamer } from "@/flux/aether/streamer";
import { type ListenerAdder } from "@/flux/aether/types";
import { status } from "@/status/aether";

export type ProviderState = z.input<typeof providerStateZ>;
export const providerStateZ = z.object({});

interface InternalState {
  streamer: Streamer;
}

export interface ContextValue {
  addListener: ListenerAdder | null;
}

export const ZERO_CONTEXT_VALUE: ContextValue = {
  addListener: null,
};

const CONTEXT_KEY = "flux-context";

const set = (ctx: aether.Context, value: ContextValue): void =>
  ctx.set(CONTEXT_KEY, value);

export class Provider extends aether.Composite<typeof providerStateZ, InternalState> {
  static readonly TYPE = "flux.Provider";
  static readonly stateZ = providerStateZ;
  schema = Provider.stateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    if (!ctx.wasSetPreviously(CONTEXT_KEY)) set(ctx, ZERO_CONTEXT_VALUE);
    const client = synnax.use(ctx);
    const handleError = status.useErrorHandler(ctx);
    i.streamer ??= new Streamer(handleError);
    if (client == null) return;
    handleError(
      async () => await i.streamer.updateStreamer(client.openStreamer.bind(client)),
      "Failed to update Flux.Provider streamer",
    );
    const ctxValue: ContextValue = {
      addListener: ({ channel, handler, onOpen }) =>
        i.streamer.addListener(handler, channel, onOpen),
    };
    ctx.set(CONTEXT_KEY, ctxValue);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Provider.TYPE]: Provider,
};

const noopListenerAdder: ListenerAdder = () => () => {};

export const useAddListener = (ctx: aether.Context): ListenerAdder => {
  const value = ctx.get<ContextValue>(CONTEXT_KEY);
  return value.addListener ?? noopListenerAdder;
};

export const mountListeners = (
  add: ListenerAdder,
  handleError: status.ErrorHandler,
  listeners: ListenerSpec<MultiSeries, {}> | flux.ListenerSpec<MultiSeries, {}>[],
): Destructor => {
  const destructors = array.toArray(listeners).map(({ channel, onChange }) =>
    add({
      channel,
      handler: (frame) => {
        handleError(
          async () => await onChange({ changed: frame.get(channel) }),
          `Error in Flux.useListener on channel ${channel}`,
        );
      },
    }),
  );
  return () => destructors.forEach((d) => d());
};

export const useListener = (
  ctx: aether.Context,
  listeners: ListenerSpec<MultiSeries, {}> | flux.ListenerSpec<MultiSeries, {}>[],
  destructor: Destructor | null,
): Destructor => {
  if (destructor != null) return destructor;
  const addListener = useAddListener(ctx);
  const handleError = status.useErrorHandler(ctx);
  return mountListeners(addListener, handleError, listeners);
};
