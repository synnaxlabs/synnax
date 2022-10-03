import test from 'ava';
import { z } from 'zod';

import { JSONEncoderDecoder } from './encoder';
import { HTTPClientFactory } from './http';
import URL from './url';

const ENDPOINT = new URL({
  host: '127.0.0.1',
  port: 8080,
  pathPrefix: 'unary',
});

const factory = new HTTPClientFactory(ENDPOINT, new JSONEncoderDecoder());

const MessageSchema = z.object({
  id: z.number().optional(),
  message: z.string().optional(),
});

type Message = z.infer<typeof MessageSchema>;

const getClient = factory.getClient();
const postClient = factory.postClient();

test('[http] - post echo', async (t) => {
  const [response, error] = await postClient.send<Message, Message>(
    '/echo',
    {
      id: 1,
      message: 'hello',
    },
    MessageSchema
  );
  t.is(error, undefined);
  t.deepEqual(response, { id: 2, message: 'hello' });
});

test('[http] - get echo', async (t) => {
  const [response, error] = await getClient.send<Message, Message>(
    '/echo',
    {
      id: 1,
      message: 'hello',
    },
    MessageSchema
  );
  t.is(error, undefined);
  t.deepEqual(response, { id: 2, message: 'hello' });
});

test('[http] - get not found', async (t) => {
  const [response, error] = await getClient.send<Message, Message>(
    '/not-found',
    {},
    MessageSchema
  );
  t.is(error?.message, 'Cannot GET /http/not-found/');
  t.is(response, undefined);
});

test('[http] - post not found', async (t) => {
  const [response, error] = await postClient.send<Message, Message>(
    '/not-found',
    {},
    MessageSchema
  );
  t.is(error?.message, 'Cannot POST /http/not-found/');
  t.is(response, undefined);
});
