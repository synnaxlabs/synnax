import { describe, expect, test } from "vitest";
import { z } from "zod";

import { JSONEncoderDecoder } from "./encoder";
import { HTTPClientFactory } from "./http";
import URL from "./url";

const ENDPOINT = new URL({
  host: "127.0.0.1",
  port: 8080,
  pathPrefix: "unary",
});

const factory = new HTTPClientFactory(ENDPOINT, new JSONEncoderDecoder());

const MessageSchema = z.object({
  id: z.number().optional(),
  message: z.string().optional(),
});

type Message = z.infer<typeof MessageSchema>;

const getClient = factory.getClient();
const postClient = factory.postClient();

describe("http", () => {
  test("post echo", async () => {
    const [response, error] = await postClient.send<Message, Message>(
      "/echo",
      {
        id: 1,
        message: "hello",
      },
      MessageSchema
    );
    expect(error).toBeUndefined();
    expect(response).toEqual({ id: 2, message: "hello" });
  });

  test("get echo", async () => {
    const [response, error] = await getClient.send<Message, Message>(
      "/echo",
      {
        id: 1,
        message: "hello",
      },
      MessageSchema
    );
    expect(error).toBeUndefined();
    expect(response).toEqual({ id: 2, message: "hello" });
  });

  test("get not found", async () => {
    const [response, error] = await getClient.send<Message, Message>(
      "/not-found",
      {},
      MessageSchema
    );
    expect(error?.message).toEqual("Cannot GET /unary/not-found");
    expect(response).toBeUndefined();
  });

  test("post not found", async () => {
    const [response, error] = await postClient.send<Message, Message>(
      "/not-found",
      {},
      MessageSchema
    );
    expect(error?.message).toEqual("Cannot POST /unary/not-found");
    expect(response).toBeUndefined();
  });

  test("middleware", async () => {
    const client = factory.getClient();
    client.use(async (md, next) => {
      md.params["Test"] = "test";
      return await next(md);
    });
    const [response, error] = await client.send<Message, Message>(
      "/middlewareCheck",
      {},
      MessageSchema
    );
    expect(error).toBeUndefined();
    expect(response?.message).toEqual("");
  });
});
