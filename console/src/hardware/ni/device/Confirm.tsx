import { type ReactElement, useState } from "react";

import { Align, Button, Form, Synnax, Text } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";

import { GroupConfig, type PhysicalPlan } from "@/hardware/ni/device/types";

//   const ok = methods.validate("softwarePlan");
//   if (!ok) return;
//   const groups = methods.get<PhysicalPlan>({ path: "channels" }).value.groups;
//   if (client == null) return;

//   const rack = await client.hardware.racks.retrieve(device.rack);
//   const output = new Map<string, number>();
//   await Promise.all(
//     groups.map(async (g) => {
//       const rawIdx = g.channels.find((c) => c.isIndex);
//       if (rawIdx == null) return;
//       const idx = await client.channels.create({
//         name: rawIdx.name,
//         isIndex: true,
//         dataType: rawIdx?.dataType,
//       });
//       const rawDataChannels = g.channels.filter(
//         (c) => !c.isIndex && c.synnaxChannel == null,
//       );
//       const data = await client.channels.create(
//         rawDataChannels.map((c) => ({
//           name: c.name,
//           dataType: c.dataType,
//           index: idx.key,
//         })),
//       );
//       data.forEach((c, i): void => {
//         rawDataChannels[i].synnaxChannel = c.key;
//       });
//       rawIdx.synnaxChannel = idx.key;
//       g.channels.forEach((c) => {
//         output.set(c.key, c.synnaxChannel);
//       });
//     }),
//   );

//   const tasks = methods.get<NITask[]>({ path: "softwarePlan.tasks" }).value;
//   if (client == null) return;

//   tasks.forEach((t) => {
//     t.config.channels.forEach((c) => {
//       c.channel = output.get(c.key) as string;
//     });
//   });

//   const t = tasks[0];
//   await rack.createTask({
//     name: t.name,
//     type: t.type,
//     config: t.config,
//   });
// }

export const Confirm = (): ReactElement => {
  const client = Synnax.use();
  const formCtx = Form.useContext();

  const [step, setStep] = useState("");

  const { mutate, isPending } = useMutation({
    mutationKey: [client?.key],
    mutationFn: async () => {
      if (client == null) return;
      const groups = formCtx.get<GroupConfig[]>({ path: "groups" }).value;
      for (const group of groups) {
        const rawIdx = group.channels.find((c) => c.isIndex);
        setStep(`Creating index for ${group.name}`);
        if (rawIdx == null) return;
        const idx = await client.channels.create({
          name: rawIdx.name,
          isIndex: true,
          dataType: rawIdx.dataType,
        });
        const rawDataChannels = group.channels.filter(
          (c) => !c.isIndex && c.synnaxChannel == null,
        );
        setStep(`Creating data channels for ${group.name}`);
        await client.channels.create(
          rawDataChannels.map((c) => ({
            name: c.name,
            dataType: c.dataType,
            index: idx.key,
          })),
        );
      }
    },
  });

  return (
    <Align.Center>
      <Text.Text level="h1">Ready to configure?</Text.Text>
      <Text.Text level="p">
        Hitting confirm will make permanent changes to the channels in your Synnax
        cluster. To edit information in the previous steps, you'll need to reconfigure
        the device. Hit confirm to proceed.
      </Text.Text>
      <Button.Button loading={isPending} disabled={isPending} onClick={() => mutate()}>
        Confirm
      </Button.Button>
      {isPending && <Text.Text level="p">{step}</Text.Text>}
    </Align.Center>
  );
};
