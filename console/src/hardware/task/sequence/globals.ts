import { type Variable } from "@/code/phantom";

export const GLOBALS: Variable[] = [
  {
    key: "set",
    name: "set",
    value: `
    --- Set the value of a channel
    --- @param channel string The name of the channel to set the value of
    --- @param value string The value to set the channel to
    --- @return void
    function(channel, value)
    end`,
  },
  {
    key: "elapsed_time",
    name: "elapsed_time",
    value: "0.0",
    docs: "The elapsed time of the task in seconds",
  },
];
