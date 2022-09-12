"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const ava_1 = __importDefault(require("ava"));
const encoder_1 = require("./encoder");
const endpoint_1 = __importDefault(require("./endpoint"));
const errors_1 = require("./errors");
const ws_1 = require("./ws");
const ENDPOINT = new endpoint_1.default({
    host: '127.0.0.1',
    port: 8080,
});
class MyCustomError extends errors_1.BaseTypedError {
    constructor(message, code) {
        super(message, 'integration.error');
        this.code = code;
    }
}
const encodeTestError = (err) => {
    if (!(err instanceof MyCustomError)) {
        throw new Error('Unexpected error type');
    }
    return `${err.code},${err.message}`;
};
const decodeTestError = (encoded) => {
    const [code, message] = encoded.split(',');
    return new MyCustomError(message, parseInt(code, 10));
};
(0, errors_1.registerError)({
    type: 'integration.error',
    encode: encodeTestError,
    decode: decodeTestError,
});
(0, ava_1.default)('basic exchange', async (t) => {
    // Should exchange ten echo messages that increment the ID.
    const client = new ws_1.WebSocketClient(new encoder_1.MsgPackEncoderDecoder(), ENDPOINT);
    const stream = await client.stream('ws/echo');
    for (let i = 0; i < 10; i++) {
        stream.send({ id: i, message: 'hello' });
        const [response, error] = await stream.receive();
        t.is(error, undefined);
        t.is(response === null || response === void 0 ? void 0 : response.id, i + 1);
        t.is(response === null || response === void 0 ? void 0 : response.message, 'hello');
    }
    stream.closeSend();
    const [response, error] = await stream.receive();
    t.deepEqual(error, new errors_1.EOF());
    t.is(response, undefined);
});
(0, ava_1.default)('receive message after close', async (t) => {
    // Should exchange ten echo messages that increment the ID.
    const client = new ws_1.WebSocketClient(new encoder_1.MsgPackEncoderDecoder(), ENDPOINT);
    const stream = await client.stream('ws/sendMessageAfterClientClose');
    await stream.closeSend();
    let [response, error] = await stream.receive();
    t.is(error, undefined);
    t.is(response === null || response === void 0 ? void 0 : response.id, 0);
    t.is(response === null || response === void 0 ? void 0 : response.message, 'Close Acknowledged');
    [response, error] = await stream.receive();
    t.deepEqual(error, new errors_1.EOF());
});
(0, ava_1.default)('receive error', async (t) => {
    // Should exchange ten echo messages that increment the ID.
    const client = new ws_1.WebSocketClient(new encoder_1.MsgPackEncoderDecoder(), ENDPOINT);
    const stream = await client.stream('ws/receiveAndExitWithErr');
    stream.send({ id: 0, message: 'hello' });
    const [response, error] = await stream.receive();
    t.deepEqual(error, new MyCustomError('unexpected error', 1));
    t.is(response, undefined);
});
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoid3Muc3BlYy5qcyIsInNvdXJjZVJvb3QiOiIiLCJzb3VyY2VzIjpbIi4uLy4uLy4uL3NyYy9saWIvd3Muc3BlYy50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7OztBQUFBLDhDQUF1QjtBQUV2Qix1Q0FBa0Q7QUFDbEQsMERBQWtDO0FBQ2xDLHFDQUEwRTtBQUMxRSw2QkFBdUM7QUFFdkMsTUFBTSxRQUFRLEdBQUcsSUFBSSxrQkFBUSxDQUFDO0lBQzVCLElBQUksRUFBRSxXQUFXO0lBQ2pCLElBQUksRUFBRSxJQUFJO0NBQ1gsQ0FBQyxDQUFDO0FBT0gsTUFBTSxhQUFjLFNBQVEsdUJBQWM7SUFHeEMsWUFBWSxPQUFlLEVBQUUsSUFBWTtRQUN2QyxLQUFLLENBQUMsT0FBTyxFQUFFLG1CQUFtQixDQUFDLENBQUM7UUFDcEMsSUFBSSxDQUFDLElBQUksR0FBRyxJQUFJLENBQUM7SUFDbkIsQ0FBQztDQUNGO0FBRUQsTUFBTSxlQUFlLEdBQUcsQ0FBQyxHQUFlLEVBQVUsRUFBRTtJQUNsRCxJQUFJLENBQUMsQ0FBQyxHQUFHLFlBQVksYUFBYSxDQUFDLEVBQUU7UUFDbkMsTUFBTSxJQUFJLEtBQUssQ0FBQyx1QkFBdUIsQ0FBQyxDQUFDO0tBQzFDO0lBQ0QsT0FBTyxHQUFHLEdBQUcsQ0FBQyxJQUFJLElBQUksR0FBRyxDQUFDLE9BQU8sRUFBRSxDQUFDO0FBQ3RDLENBQUMsQ0FBQztBQUVGLE1BQU0sZUFBZSxHQUFHLENBQUMsT0FBZSxFQUFjLEVBQUU7SUFDdEQsTUFBTSxDQUFDLElBQUksRUFBRSxPQUFPLENBQUMsR0FBRyxPQUFPLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBQzNDLE9BQU8sSUFBSSxhQUFhLENBQUMsT0FBTyxFQUFFLFFBQVEsQ0FBQyxJQUFJLEVBQUUsRUFBRSxDQUFDLENBQUMsQ0FBQztBQUN4RCxDQUFDLENBQUM7QUFFRixJQUFBLHNCQUFhLEVBQUM7SUFDWixJQUFJLEVBQUUsbUJBQW1CO0lBQ3pCLE1BQU0sRUFB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