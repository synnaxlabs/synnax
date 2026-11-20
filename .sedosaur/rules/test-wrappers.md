+++
description = "Console and Pluto test wrappers and a live Core"
paths = ["console/**", "pluto/**"]
+++

A test wrapper with a non-null client talks to a live Core, so it calls
`disableActEnvironment()`. A test that needs no Core passes `client: null` and keeps
React's act environment. Report a test that talks to a Core with the act environment on,
or one that passes a client it does not need.
