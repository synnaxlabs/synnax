import { PageNavLeaf } from "@/components/PageNav";

export const serverCLINav: PageNavLeaf = {
  key: "server-cli",
  name: "Server CLI",
  children: [
    {
      key: "start",
      url: "/reference/server-cli/start",
      name: "Start",
    },
    {
      key: "systemd-service",
      url: "/reference/server-cli/systemd-service",
      name: "Systemd Service",
    },
  ],
};
