import { Layout } from "@/layout";
import { State, create } from "@/lineplot/slice";
import { Synnax } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";

export const handleLink = ({ url, placer, client }: HandleLinkProps): boolean => {
  if (!url.includes("lineplot")) return false;

  const path = url.split("/");
  const item = path[path.indexOf("lineplot") + 1];
  void (async () => {
    const plot = await client.workspaces.linePlot.retrieve(item);
    placer(
      create({
        ...(plot.data as unknown as State),
        key: plot.key,
        name: plot.name,
      }),
    );
  })();
};
