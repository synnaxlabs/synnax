import { decodeError, EOF, StreamClosed } from './errors';
import { Runtime, RUNTIME } from './runtime';
const resolveWebsocketProvider = () => {
    if (RUNTIME == Runtime.Node) {
        return require('ws');
    }
    return WebSocket;
};
var MessageType;
(function (MessageType) {
    MessageType["Data"] = "data";
    MessageType["Close"] = "close";
})(MessageType || (MessageType = {}));
var CloseCode;
(function (CloseCode) {
    CloseCode[CloseCode["Normal"] = 1000] = "Normal";
    CloseCode[CloseCode["GoingAway"] = 1001] = "GoingAway";
})(CloseCode || (CloseCode = {}));
export class WebSocketClientStream {
    encoder;
    ws;
    server_closed;
    send_closed;
    receiveDataQueue = [];
    receiveCallbacksQueue = [];
    constructor(encoder, ws) {
        this.encoder = encoder;
        this.ws = ws;
        this.send_closed = false;
        this.listenForMessages();
    }
    send(req) {
        if (this.server_closed) {
            return new EOF();
        }
        if (this.send_closed) {
            throw new StreamClosed();
        }
        this.ws.send(this.encoder.encode({
            type: MessageType.Data,
            payload: req,
        }));
        return undefined;
    }
    async receive() {
        if (this.server_closed) {
            return [undefined, this.server_closed];
        }
        const msg = await this.receiveMsg();
        if (msg.type == MessageType.Close) {
            if (!msg.error) {
                throw new Error('Message error must be defined');
            }
            this.server_closed = decodeError(msg.error);
            return [undefined, this.server_closed];
        }
        return [msg.payload, undefined];
    }
    closeSend() {
        if (this.send_closed || this.server_closed) {
            return undefined;
        }
        const msg = { type: MessageType.Close };
        try {
            this.ws.send(this.encoder.encode(msg));
        }
        finally {
            this.send_closed = true;
        }
        return undefined;
    }
    async receiveMsg() {
        if (this.receiveDataQueue.length !== 0) {
            const msg = this.receiveDataQueue.shift();
            if (msg) {
                return msg;
            }
            else {
                throw new Error('unexpected undefined message');
            }
        }
        return new Promise((resolve, reject) => {
            this.receiveCallbacksQueue.push({ resolve, reject });
        });
    }
    listenForMessages() {
        this.ws.onmessage = (ev) => {
            const msg = this.encoder.decode(ev.data);
            if (this.receiveCallbacksQueue.length > 0) {
                const callback = this.receiveCallbacksQueue.shift();
                if (callback) {
                    callback.resolve(msg);
                }
                else {
                    throw new Error('Unexpected empty callback queue');
                }
            }
            else {
                this.receiveDataQueue.push(msg);
            }
        };
        this.ws.onclose = (ev) => {
            if ([CloseCode.Normal, CloseCode.GoingAway].includes(ev.code)) {
                this.server_closed = new EOF();
            }
            else {
                this.server_closed = new StreamClosed();
            }
        };
    }
}
export class WebSocketClient {
    endpoint;
    encoder;
    constructor(encoder, endpoint) {
        this.endpoint = endpoint.child({ protocol: 'ws' });
        this.encoder = encoder;
    }
    async stream(target) {
        const ResolvedWebSocket = resolveWebsocketProvider();
        const url = this.endpoint.path(`${target}?contentType=${this.encoder.contentType}`);
        const ws = new ResolvedWebSocket(url);
        ws.binaryType = 'arraybuffer';
        return new Promise((resolve, reject) => {
            ws.onopen = () => {
                resolve(new WebSocketClientStream(this.encoder, ws));
            };
            ws.onerror = (ev) => {
                reject(ev);
            };
        });
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoid3MuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi9zcmMvbGliL3dzLnRzIl0sIm5hbWVzIjpbXSwibWFwcGluZ3MiOiJBQUVBLE9BQU8sRUFBRSxXQUFXLEVBQUUsR0FBRyxFQUFnQixZQUFZLEVBQUUsTUFBTSxVQUFVLENBQUM7QUFDeEUsT0FBTyxFQUFFLE9BQU8sRUFBRSxPQUFPLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFJN0MsTUFBTSx3QkFBd0IsR0FBRyxHQUFxQixFQUFFO0lBQ3RELElBQUksT0FBTyxJQUFJLE9BQU8sQ0FBQyxJQUFJLEVBQUU7UUFDM0IsT0FBTyxPQUFPLENBQUMsSUFBSSxDQUFDLENBQUM7S0FDdEI7SUFDRCxPQUFPLFNBQVMsQ0FBQztBQUNuQixDQUFDLENBQUM7QUFFRixJQUFLLFdBR0o7QUFIRCxXQUFLLFdBQVc7SUFDZCw0QkFBYSxDQUFBO0lBQ2IsOEJBQWUsQ0FBQTtBQUNqQixDQUFDLEVBSEksV0FBVyxLQUFYLFdBQVcsUUFHZjtBQVFELElBQUssU0FHSjtBQUhELFdBQUssU0FBUztJQUNaLGdEQUFhLENBQUE7SUFDYixzREFBZ0IsQ0FBQTtBQUNsQixDQUFDLEVBSEksU0FBUyxLQUFULFNBQVMsUUFHYjtBQUVELE1BQU0sT0FBTyxxQkFBcUI7SUFHeEIsT0FBTyxDQUFpQjtJQUN4QixFQUFFLENBQVk7SUFDZCxhQUFhLENBQVM7SUFDdEIsV0FBVyxDQUFVO0lBQ3JCLGdCQUFnQixHQUFrQixFQUFFLENBQUM7SUFDckMscUJBQXFCLEdBR3ZCLEVBQUUsQ0FBQztJQUVULFlBQVksT0FBdUIsRUFBRSxFQUFhO1FBQ2hELElBQUksQ0FBQyxPQUFPLEdBQUcsT0FBTyxDQUFDO1FBQ3ZCLElBQUksQ0FBQyxFQUFFLEdBQUcsRUFBRSxDQUFDO1FBQ2IsSUFBSSxDQUFDLFdBQVcsR0FBRyxLQUFLLENBQUM7UUFDekIsSUFBSSxDQUFDLGlCQUFpQixFQUFFLENBQUM7SUFDM0IsQ0FBQztJQUVELElBQUksQ0FB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