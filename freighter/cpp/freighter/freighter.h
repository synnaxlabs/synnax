// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// Std.
#include <memory>
#include <map>
#include <string>
#include <utility>
#include <iostream>

namespace freighter {
const std::string TYPE_NIL = "nil";
const std::string TYPE_UNKNOWN = "unknown";

class Error : std::exception {
public:
    std::string type;
    std::string data;

    Error(const std::string &type, const std::string &data) noexcept: type(type), data(data) {
    }

    Error(const std::string &err) {
        type = TYPE_UNKNOWN;
        data = err;
        std::cout << err << std::endl;
        size_t pos = err.find("---");
        if (pos != std::string::npos) {
            type = err.substr(0, pos);
            data = err.substr(pos + 3);
        }
    }


    [[nodiscard]] const char *what() const noexcept override {
        return data.c_str();
    }

    [[nodiscard]] bool ok() const {
        return type == TYPE_NIL;
    }

    [[nodiscard]] std::string message() const {
        return type + ":" + data;
    }

    Error(const Error &other) noexcept {
        type = other.type;
        data = other.data;
    }

    explicit operator bool() const {
        return !ok();
    }
};


static_assert(std::is_nothrow_copy_constructible<Error>::value,
              "Error must be nothrow copy constructible");


const Error NIL = Error{TYPE_NIL, ""};

const Error STREAM_CLOSED = Error{"freighter.stream_closed", "Stream closed"};

const Error EOF_ = Error{"freighter.eof", "EOF"};


/// @brief A simple URL builder.
struct URL {
    /// @brief The IP address of the target.
    std::string ip;
    /// @brief The port of the target.
    std::uint16_t port;
    /// @brief Supplementary path information.
    std::string path;

    /// @brief Creates a URL with the given IP, port, and path.
    URL(const std::string &ip, std::uint16_t port, const std::string &path);

    /// @brief Parses the given address into a URL.
    /// @throws std::invalid_argument if the address is not a valid URL.
    explicit URL(const std::string &address);

    /// @brief Creates a child URL by appending the given path to the current path.
    /// @returns the child URL. It is guaranteed to have a single slash between the current path and child path,
    /// and have a trailing slash.
    [[nodiscard]] URL child(const std::string &child_path) const;

    /// @brief Converts the URL to a string.
    /// @returns the URL as a string.
    [[nodiscard]] std::string toString() const;
};


/// @brief A Context object that can be used to inject metadata into an outbound request or process metadata from
/// an inbound response.
class Context {
public:
    /// @brief The protocol used to send the request. Should be set by the underlying transport implementation.
    std::string protocol;
    /// @brief The target passed to UnaryClient::send or StreamClient::stream along with any base target configured
    /// in the underlying transport.
    std::string target;

    /// @brief Constructs the context with an empty set of parameters.
    Context(std::string protocol, std::string target) : protocol(std::move(protocol)), target(std::move(target)) {
        params = std::unordered_map<std::string, std::string>();
    }

    /// @brief Copy constructor
    Context(const Context &other) {
        protocol = other.protocol;
        target = other.target;
        for (auto &param: other.params) {
            params[param.first] = param.second;
        }
    }

    /// @brief Copy assignment
    Context &operator=(const Context &other) {
        protocol = other.protocol;
        target = other.target;
        for (auto &param: other.params) {
            params[param.first] = param.second;
        }
        return *this;
    }

    /// @brief Gets the parameter with the given key.
    std::string get(const std::string &key) {
        return params[key];
    }

    /// @brief Sets the given parameter to the given value.
    void set(const std::string &key, const std::string &value) {
        params[key] = value;
    }

    // TODO: This should probably be private no? But I want to be able to iterate through params so we can set meta data on the outbound middleware.

    std::unordered_map<std::string, std::string> params;
};

/// @brief Interface for middleware that can be used to parse/attach metadata to a request, handle errors, or
/// otherwise alter the request or its lifecycle.
class Middleware {
public:
    /// @brief Sets the next middleware in the chain.
    /// @param n the next middleware.
    virtual Middleware *setNext(Middleware *n) = 0;

    /// @brief executes the middleware.
    /// @param context the context for the outgoing request. The context for the inbound response can be accessed
    /// by calling the next middleware in the chain.
    /// @returns a pair containing the context for the inbound response and an error.
    virtual std::pair<Context, freighter::Error> operator()(Context context) = 0;
};

/// @brief A middleware implementation that simply passes the request to the next middleware in the chain. This
/// is useful as a base class for middleware that only needs to modify the request or response.
/// @implements Middleware
class PassthroughMiddleware : public Middleware {
public:
    /// @brief Constructs the middleware with a nullptr next middleware.
    PassthroughMiddleware() : next(nullptr) {}

    /// @implements Middleware::setNext
    Middleware *setNext(Middleware *n) override {
        next = n;
        return this;
    }

    /// @implements Middleware::operator()
    std::pair<Context, freighter::Error> operator()(Context context) override { return next->operator()(context); }

private:
    /// @brief the next middleware in the chain.
    Middleware *next;
};

/// @brief A middleware implementation that simply returns the context and a nullptr error. This is useful
/// as a finalizer for a middleware chain.
class Finalizer : public Middleware {
public:
    /// @brief no-op. Ignores the next middleware.
    /// @implements Middleware::setNext
    Middleware *setNext(Middleware *n) override { return this; }

    /// @implements Middleware::operator()
    std::pair<Context, freighter::Error> operator()(Context context) override {
        return {context, freighter::NIL};
    }
};

/// @brief A collector that can be used to configure and execute a chain of middleware with a finalizer. This is
/// useful as a base class for implementing UnaryClient or StreamClient.
/// @see UnaryClient
/// @see StreamClient
class MiddlewareCollector {
private:
    /// @brief The middlewares in the chain.
    std::vector<freighter::Middleware *> middlewares;
public:
    /// @brief Adds a middleware to the chain. Middleware is executed in the order it is added i.e. the last
    /// middleware added will be executed as the final middleware before the finalizer.
    /// @implements UnaryClient::use
    /// @implements StreamClient::use
    void use(freighter::Middleware *middleware) { middlewares.push_back(middleware); }

    /// @brief Executes the middleware chain.
    /// @param finalizer - the last middleware in the chain. This finalizer should NOT call the next middleware in
    /// the chain, as it will be a nullptr. It should instead execute the request and handle the response.
    std::pair<freighter::Context, freighter::Error> exec(
            const freighter::Context &context,
            freighter::Middleware *finalizer
    ) {
        if (middlewares.empty()) return finalizer->operator()(context);
        for (int i = 0; i < middlewares.size(); i++) {
            auto mw = middlewares[i];
            if (i == middlewares.size() - 1) mw->setNext(finalizer);
            else mw->setNext(middlewares[i + 1]);
        }
        return middlewares[0]->operator()(context);
    }
};


/// @brief The client side interface for a simple request-response transport between two entities.
/// @tparam response_t the expected response type.
/// @tparam request_t the request type.
template<typename response_t, typename request_t>
class UnaryClient {
public:
    /// @brief binds the middleware to the given transport. Middleware is executed in the order it is added
    /// i.e. the last middleware added will be executed as the final middleware before the request is sent.
    virtual void use(Middleware *middleware) = 0;

    /// @brief Sends the given request to the target and blocks until a response is received.
    /// @param target the target to send the request to.
    /// @param request the request to send.
    /// @returns a pair containing the response and an error.
    virtual std::pair<response_t, Error> send(const std::string &target, request_t &request) = 0;
};

/// @brief An interface for a bidirectional stream between two entities.
/// @tparam response_t the expected response type.
/// @tparam request_t the request type.
/// @tparam err_t the error type.
template<typename response_t, typename request_t>
class Stream {
public:
    /// @brief Receives a response from the stream. It's not safe to call receive concurrently with itself.
    /// @returns a pair containing the response and an error.
    virtual std::pair<response_t, Error> receive() = 0;

    /// @brief Sends a request to the stream. It is not safe to call send concurrently with itself or closeSend.
    /// @param request - the request to send.
    virtual Error send(request_t &request) = 0;

    /// @brief Closes the sending end of the stream, signaling to the server that no more requests will be send,
    /// and (if desired) allowing the server to close the receiving end of the stream.
    virtual Error closeSend() = 0;
};

/// @brief The client side interface for opening bidirectional streams between two entities.
template<typename response_t, typename request_t>
class StreamClient {
public:
    /// @brief binds the middleware to the given transport. Middleware is executed in the order it is added
    /// i.e. the last middleware added will be executed as the final middleware before the stream is opened.
    virtual void use(Middleware *middleware) = 0;

    /// @brief Opens a stream to the given target.
    /// @see Stream.
    /// @param target the target to open the stream to.
    /// @returns a pointer to an object implementing the Stream interface.
    virtual std::pair<Stream<response_t, request_t> *, freighter::Error> stream(const std::string &target) = 0;
};

}
