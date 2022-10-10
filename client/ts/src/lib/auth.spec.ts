import { URL } from '@synnaxlabs/freighter';
import test from 'ava';

import { HOST, PORT } from '../setupspecs';

import AuthenticationClient from './auth';
import { AuthError } from './errors';
import Transport from './transport';

test('[auth] - valid credentials', async (t) => {
  const transport = new Transport(new URL({ host: HOST, port: PORT }));
  const client = new AuthenticationClient(transport.httpFactory, {
    username: 'synnax',
    password: 'seldon',
  });
  await client.authenticating;
  t.assert(client.authenticated);
});

test('[auth] - invalid credentials', async (t) => {
  const transport = new Transport(new URL({ host: HOST, port: PORT }));
  const client = new AuthenticationClient(transport.httpFactory, {
    username: 'synnax',
    password: 'wrong',
  });
  try {
    await client.authenticating;
    t.assert(false);
  } catch (e) {
    t.assert(!client.authenticated);
    t.assert(e instanceof AuthError);
    if (e instanceof AuthError) {
      t.is(e.message, '[synnax] - invalid credentials');
    }
  }
});
