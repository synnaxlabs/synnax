import test from 'ava';
import { z } from 'zod';

import { JSONEncoderDecoder } from './encoder';
import { HTTPClientFactory } from './http';
import URL from './url';

const ENDPOINT = new URL({
  host: '127.0.0.1',
  port: 8080,
  pathPrefix: 'http',
});

const factory = new HTTPClientFactory(ENDPOINT, new JSONEncoderDecoder());

const MessageSchema = z.object({
  id: z.number().optional(),
  message: z.string().optional(),
});

const getClient = factory.get(MessageSchema, MessageSchema);
const postClient = factory.post(MessageSchema, MessageSchema);

test('[http] - post echo', async (t) => {
  const [response, error] = await postClient.send('/echo', {
    id: 1,
    message: 'hello',
  });
  t.is(error, undefined);
  t.deepEqual(response, { id: 2, message: 'hello' });
});

test('[http] - get echo', async (t) => {
  const [response, error] = await getClient.send('/echo', {
    id: 1,
    message: 'hello',
  });
  t.is(error, undefined);
  t.deepEqual(response, { id: 2, message: 'hello' });
});

test('[http] - get not found', async (t) => {
  const [response, error] = await getClient.send('/not-found', {});
  t.is(error?.message, 'Request failed with status code 404');
  t.is(response, undefined);
});

test('[http] - post not found', async (t) => {
  const [response, error] = await postClient.send('/not-found', {});
  t.is(error?.message, 'Request failed with status code 404');
  t.is(response, undefined);
});
