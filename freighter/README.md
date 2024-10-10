<br/>
<p align="center">
    <a href="https://synnaxlabs.com/">
        <img src="../docs/media/logo/title-white.svg" width="70%"/>
    </a>
</p>

# Freighter

Freighter defines a protocol agnostic transport interface that allows libraries
in a variety of languages to communicate without needing to know protocol, routing,
or encoding details.

It's interface is similar to [gRPC](https://grpc.io/), but it can be implemented by
HTTP, WebSockets, WebRTC, UDP, TCP, etc. For more information on its design, see the
[Freighter RFC](/docs/tech/rfc/0006-220809-freighter.md).

The implementations for specific languages are contained in subdirectories of
this directory.
