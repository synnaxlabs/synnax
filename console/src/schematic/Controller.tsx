import { Control, User } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { setControlStatus } from "@/schematic/slice";

export interface ControllerProps extends Omit<
  Control.ControllerProps,
  "name" | "onStatusChange"
> {
  resourceKey: string;
}

export const Controller = ({ resourceKey, ...rest }: ControllerProps) => {
  const name = Layout.useSelectRequiredName(resourceKey);
  const dispatch = useDispatch();
  const { data: user } = User.useRetrieve({}, { addStatusOnFailure: false });
  const username = user?.username ?? "";
  const controlName = username.length > 0 ? `${name} (${username})` : name;
  const handleControlStatusChange = useCallback(
    (next: Control.Status) =>
      dispatch(setControlStatus({ key: resourceKey, control: next })),
    [dispatch, resourceKey],
  );
  return (
    <Control.Controller
      onStatusChange={handleControlStatusChange}
      name={controlName}
      {...rest}
    />
  );
};
