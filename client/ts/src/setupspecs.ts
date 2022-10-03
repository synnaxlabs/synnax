import Synnax, { SynnaxProps } from './lib/client';

export const HOST = 'localhost';
export const PORT = 8080;

export const newClient = (...props: SynnaxProps[]): Synnax => {
  let _props = {};
  if (props.length > 0) _props = props[0];
  return new Synnax({
    host: HOST,
    port: PORT,
    username: 'synnax',
    password: 'seldon',
    ..._props,
  });
};
