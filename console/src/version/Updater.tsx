import { check } from "@tauri-apps/plugin-updater";
import { relaunch } from "@tauri-apps/plugin-process";
import { useEffect } from "react";
import { TimeSpan } from "@synnaxlabs/x";
import { Status } from "@synnaxlabs/pluto";

// export const useCheckForUpdates = () => {
//   const addStatus = Status.useAggregator();
//   useEffect(() => {
//     setInterval(async () => {
//       const update = await check();
//       addStatus({
//         key: "update",
//         description: "update",
//       });
//     }, TimeSpan.seconds(5).milliseconds);
//   }, []);
// };
