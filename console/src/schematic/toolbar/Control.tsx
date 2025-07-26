import { Align, Input } from "@synnaxlabs/pluto";
import { control } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { useSelectAuthority } from "@/schematic/selectors";
import { setAuthority } from "@/schematic/slice";

export const Control = ({ layoutKey }: { layoutKey: string }) => {
  const dispatch = useDispatch();
  const authority = useSelectAuthority(layoutKey);

  return (
    <Align.Space x gap="small" style={{ padding: "1.5rem 2rem" }}>
      <Input.Item label="Control Authority">
        <Input.Numeric
          value={authority ?? 0}
          onChange={(v) => dispatch(setAuthority({ key: layoutKey, authority: v }))}
          bounds={control.AUTHORITY_BOUNDS}
        />
      </Input.Item>
    </Align.Space>
  );
};
