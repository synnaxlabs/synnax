import test from 'ava';

import { newClient } from '../../setupspecs';
import { QueryError } from '../errors';
import { DataType, Rate } from '../telem';

const client = newClient();

test('Channel - create', async (t) => {
  const channel = await client.channel.create({
    name: 'test',
    nodeId: 1,
    rate: Rate.Hz(1),
    dataType: DataType.Float32,
  });
  t.is(channel.name, 'test');
  t.is(channel.nodeId, 1);
  t.deepEqual(channel.rate, Rate.Hz(1));
  t.deepEqual(channel.dataType, DataType.Float32);
});

test('Channel - retrieve by key', async (t) => {
  const channel = await client.channel.create({
    name: 'test',
    nodeId: 1,
    rate: Rate.Hz(1),
    dataType: DataType.Float32,
  });
  const retrieved = (await client.channel.retrieveByKeys(channel.key))[0];
  t.is(retrieved.name, 'test');
  t.is(retrieved.nodeId, 1);
  t.deepEqual(retrieved.rate, Rate.Hz(1));
  t.deepEqual(retrieved.dataType, DataType.Float32);
});

test('Channel - retrieve by key - not found', async (t) => {
  const err = await t.throwsAsync(async () => {
    await client.channel.retrieveByKeys('1-1000');
  });
  t.true(err instanceof QueryError);
});

test('Channel - retrieve by node id', async (t) => {
  const retrieved = await client.channel.retrieveByNodeId(1);
  t.true(retrieved.length > 0);
  retrieved.forEach((ch) => {
    t.is(ch.nodeId, 1);
  });
});

test('Channel - retrieve by name', async (t) => {
  const retrieved = await client.channel.retrieveByNames("test");
  t.true(retrieved.length > 0);
  retrieved.forEach((ch) => {
    t.is(ch.name, "test");
  });
})