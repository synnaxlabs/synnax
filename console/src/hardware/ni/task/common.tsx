import { Device, Form, Synnax } from "@synnaxlabs/pluto";

import { Device as NIDevice } from "@/hardware/ni/device";
import { Properties } from "@/hardware/ni/device/types";
import { Layout } from "@/layout";

export const SelectDevice = () => {
  const client = Synnax.use();
  const placer = Layout.usePlacer();
  const handleDeviceChange = async (v: string) => {
    if (client == null) return;
    const { configured } = await client.hardware.devices.retrieve<Properties>(v);
    if (configured) return;
    placer(NIDevice.createConfigureLayout(v, {}));
  };
  return (
    <Form.Field<string>
      path="config.device"
      label="Device"
      grow
      onChange={handleDeviceChange}
      style={{ width: "100%" }}
    >
      {(p) => (
        <Device.SelectSingle
          allowNone={false}
          grow
          {...p}
          autoSelectOnNone={false}
          searchOptions={{ makes: ["NI"] }}
        />
      )}
    </Form.Field>
  );
};
