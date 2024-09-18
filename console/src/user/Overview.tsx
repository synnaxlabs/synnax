import { Align, Divider, Form, Text } from "@synnaxlabs/pluto";
import { FC, ReactElement } from "react";
import { useDispatch } from "react-redux";
import { z } from "zod";

import { Layout } from "@/layout";

export const OVERVIEW_LAYOUT_TYPE = "userOverview";

export const overviewLayout: Layout.State = {
  key: OVERVIEW_LAYOUT_TYPE,
  windowKey: OVERVIEW_LAYOUT_TYPE,
  type: OVERVIEW_LAYOUT_TYPE,
  name: "User Overview",
  location: "mosaic",
  icon: "User",
};

export const Overview: Layout.Renderer = (): ReactElement => (
  <Align.Space direction="y">
    <Info />
    <Divider.Divider direction="x" />
    <Permissions />
  </Align.Space>
);

interface InfoProps {
  foo: boolean;
}

const formSchema = z.object({
  username: z.string().min(1, "Username must not be empty"),
  firstName: z.string().min(1, "First name must not be empty"),
  lastName: z.string().min(1, "Last name must not be empty"),
});

const Info: FC<InfoProps> = (): ReactElement => {
  const dispatch = useDispatch();

  const formCtx = Form.useSynced<typeof formSchema>({
    name: "User Info",
    key: [OVERVIEW_LAYOUT_TYPE, "info"],
    schema: formSchema,
    values: {
      username: "",
      firstName: "",
      lastName: "",
    },
    queryFn: async ({ client }) => {
      const user = await client.user.retrieve();
      return {
        username: user.username,
        firstName: user.firstName,
        lastName: user.lastName,
      };
    },
  });

  return (
    <Align.Space direction="y">
      <Text.Text level="p">info username etc.</Text.Text>
    </Align.Space>
  );
};

interface PermissionsProps {}

const Permissions: FC<PermissionsProps> = (): ReactElement => {
  return (
    <Align.Space direction="y">
      <Text.Text level="p">Permissions</Text.Text>
    </Align.Space>
  );
};
