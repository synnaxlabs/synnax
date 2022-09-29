import { UnaryClient } from "@synnaxlabs/freighter";
import {z} from "zod"

export default class Auth {
    client: UnaryClient;

    main() {
        this.client.send("/auth/login", {
            username: "whatever",
            password: "whatever",
        })
    }
}