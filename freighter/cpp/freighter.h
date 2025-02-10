// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <iostream>

#include "x/cpp/xerrors/errors.h"

namespace freighter {

const std::string TYPE_UNREACHABLE = "freighter.unreachable";
const std::string TYPE_NIL = "nil";
const std::string TYPE_UNKNOWN = "unknown";

const xerrors::Error STREAM_CLOSED = {TYPE_UNREACHABLE + ".stream_closed", "Stream closed"};
const xerrors::Error EOF_ = {"freighter.eof", "EOF"};
const xerrors::Error UNREACHABLE = {TYPE_UNREACHABLE, "Unreachable"};


/// @brief A simple URL builder.
struct URL {
    /// @brief The IP address of the target.
    std::string ip;
    /// @brief The port of the target.
    std::uint16_t port;
    /// @brief Supplementary path information.
    std::string path;

    URL();

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
    [[nodiscard]] std::string to_string() const;
};

enum TransportVariant {
    UNARY,
    STREAM
};

/// @brief A Context object that can be used to inject metadata into an outbound request or process metadata from
/// an inbound response.
class Context {
public:
    /// @brief unique hash used to retreive sent data.
    int id;
    /// @brief The protocol used to send the request. Should be set by the underlying transport implementation.
    std::string protocol;
    /// @brief The target passed to UnaryClient::send or StreamClient::stream along with any base target configured
    /// in the underlying transport.
    std::string target;
    /// @brief The transport variant that the context is associated with. Can either be
    /// a streaming (STREAM) or unary (UNARY) transport.
    TransportVariant variant;

    /// @brief Constructs the context with an empty set of parameters.
    Context(
        std::string protocol,
        std::string target,
        const TransportVariant variant
    ) : id(0),
        protocol(std::move(protocol)),
        target(std::move(target)), variant(variant) {
        params = std::unordered_map<std::string, std::string>();
    }

    /// @brief Copy constructor
    Context(
        const Context &other
    ) : id(other.id),
        protocol(other.protocol),
        target(other.target),
        variant(other.variant) {
        for (const auto &[k, v]: other.params) params[k] = v;
    }

    /// @brief Copy assignment
    Context &operator=(const Context &other) {
        protocol = other.protocol;
        target = other.target;
        id = other.id;
        variant = other.variant;
        for (auto &[k, v]: other.params) params[k] = v;
        return *this;
    }

    /// @brief Gets the parameter with the given key.
    std::string get(const std::string &key) { return params[key]; }

    bool has(const std::string &key) { return params.find(key) != params.end(); }

    /// @brief Sets the given parameter to the given value.
    void set(const std::string &key, const std::string &value) { params[key] = value; }

    std::unordered_map<std::string, std::string> params;
};

class Next {
public:
    virtual std::pair<Context, xerrors::Error> operator()(Context context) = 0;

    virtual ~Next() = default;
};

/// @brief Interface for middleware that can be used to parse/attach metadata to a
/// request, handle errors, or otherwise alter the request or its lifecycle.
class Middleware {
public:
    /// @brief executes the middleware.
    /// @param context the context for the outgoing request. The context for the
    /// inbound response can be accessed by calling the next middleware in the chain.
    /// @param next a lambda that can be called to execute the next middleware in the
    /// chain.
    /// @returns a pair containing the context for the inbound response and an error.
    virtual std::pair<Context, xerrors::Error> operator()(Context context, Next *next)
    = 0;

    virtual ~Middleware() = default;
};

/// @brief A middleware implementation that simply passes the request to the next
/// middleware in the chain. This is useful as a base class for middleware that only
/// needs to modify the request or response.
/// @implements Middleware
class PassthroughMiddleware : public Middleware {
public:
    /// @brief Constructs the middleware with a nullptr next middleware.
    PassthroughMiddleware() = default;

    /// @implements Middleware::operator()
    std::pair<Context, xerrors::Error> operator()(
        Context context,
        Next *next
    ) override {
        return next->operator()(context);
    }
};

template<typename RS>
struct FinalizerReturn {
    Context context;
    xerrors::Error error;
    RS response;
};

/// @brief A middleware implementation that simply returns the context and a nullptr
/// error. This is useful as a finalizer for a middleware chain.
template<typename RQ, typename RS>
class Finalizer {
public:
    virtual FinalizerReturn<RS> operator()(Context context, RQ &req) {
        return {context, xerrors::NIL};
    }

    virtual ~Finalizer() = default;
};

/// @brief A collector that can be used to configure and execute a chain of middleware
/// with a finalizer. This is useful as a base class for implementing UnaryClient or
/// StreamClient.
/// @see UnaryClient
/// @see StreamClient
template<typename RQ, typename RS>
class MiddlewareCollector {
public:
    MiddlewareCollector() = default;

    MiddlewareCollector(const MiddlewareCollector &other) {
        middlewares = other.middlewares;
    }

    /// @brief Adds a middleware to the chain. Middleware is executed in the order it
    /// is added i.e. the last middleware added will be executed as the final middleware
    /// before the finalizer.
    /// @implements UnaryClient::use
    /// @implements StreamClient::use
    void use(std::shared_ptr<freighter::Middleware> middleware) {
        middlewares.push_back(std::move(middleware));
    }

    /// @brief Executes the middleware chain.
    /// @param context - contains context information for the request that can be modified
    /// by the middleware prior to it's executing. Middleware key-value pairs shouild
    /// be sent to the server by the finalizer.
    /// @param finalizer - A finalizer that represents the last middleware in the chain,
    /// and is responsible for executing the request.
    /// @param req - the request to execute.
    std::pair<RS, xerrors::Error> exec(
        const freighter::Context &context,
        freighter::Finalizer<RQ, RS> *finalizer,
        RQ &req
    ) const {
        class NextImpl : public Next {
            int index;
            const MiddlewareCollector &collector;
            freighter::Finalizer<RQ, RS> *finalizer;
            RQ req;

        public:
            RS res;

            NextImpl(
                const MiddlewareCollector &collector,
                freighter::Finalizer<RQ, RS> *finalizer,
                RQ &req
            ) : index(0), collector(collector), finalizer(finalizer), req(req) {
            }

            std::pair<Context, xerrors::Error> operator()(
                freighter::Context context
            ) override {
                if (index >= collector.middlewares.size()) {
                    auto f_res = finalizer->operator()(context, req);
                    res = std::move(f_res.response);
                    return {f_res.context, f_res.error};
                }
                auto mw = collector.middlewares[index].get();
                index++;
                return mw->operator()(context, this);
            }
        };
        auto mw = std::make_unique<NextImpl>(*this, finalizer, req);
        auto err = mw->operator()(context).second;
        return {std::move(mw->res), err};
    }

private:
    /// @brief The middlewares in the chain.
    std::vector<std::shared_ptr<freighter::Middleware> > middlewares{};
};

/// @brief The client side interface for a simple request-response transport between two
/// entities.
/// @tparam RS the expected response type.
/// @tparam RQ the request type.
template<typename RQ, typename RS>
class UnaryClient {
public:
    /// @brief binds the middleware to the given transport. Middleware is executed in
    /// the order it is added i.e. the last middleware added will be executed as the
    /// final middleware before the request is sent.
    virtual void use(std::shared_ptr<Middleware> middleware) = 0;

    /// @brief Sends the given request to the target and blocks until a response is
    /// received.
    /// @param target the target to send the request to.
    /// @param request the request to send.
    /// @returns a pair containing the response and an error.
    virtual std::pair<RS, xerrors::Error> send(const std::string &target, RQ &request) = 0;

    virtual ~UnaryClient() = default;
};

/// @brief An interface for a bidirectional stream between two entities.
/// @tparam RS the expected response type.
/// @tparam RQ the request type.
template<typename RQ, typename RS>
class Stream {
public:
    /// @brief Receives a response from the stream. It's not safe to call receive
    /// concurrently with itself.
    /// @returns a pair containing the response and an error.
    virtual std::pair<RS, xerrors::Error> receive() = 0;

    /// @brief Sends a request to the stream. It is not safe to call send concurrently
    /// with itself or close_send.
    /// @param request - the request to send.
    virtual xerrors::Error send(RQ &request) const = 0;

    /// @brief Closes the sending end of the stream, signaling to the server that no
    /// more requests will be send, and (if desired) allowing the server to close the
    /// receiving end of the stream.
    virtual void close_send() = 0;

    virtual ~Stream() = default;
};

/// @brief The client side interface for opening bidirectional streams between two entities.
template<typename RQ, typename RS>
class StreamClient {
public:
    /// @brief binds the middleware to the given transport. Middleware is executed in
    /// the order it is added i.e. the last middleware added will be executed as the
    /// final middleware before the stream is opened.
    virtual void use(std::shared_ptr<Middleware> middleware) = 0;

    /// @brief Opens a stream to the given target.
    /// @see Stream.
    /// @param target the target to open the stream to.
    /// @returns a pointer to an object implementing the Stream interface.
    virtual std::pair<std::unique_ptr<Stream<RQ, RS> >, xerrors::Error>
    stream(const std::string &target) = 0;

    virtual ~StreamClient() = default;
};
}
