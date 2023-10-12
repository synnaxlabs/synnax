// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <map>

/// Local headers.

// std.
#include <string>
#include <utility>


namespace Freighter {
    class URL {
    private:

    public:
        std::string ip;
        std::uint16_t port;
        std::string path;

        URL(std::string ip, std::uint16_t port, const std::string &path = "");

        explicit URL(const std::string &address);

        URL child(const std::string &child_path);

        std::string toString();
    };


    class Context {
    public:
        std::string protocol;
        std::string target;

        Context(std::string protocol, std::string target) : protocol(std::move(protocol)), target(std::move(target)) {
            params = std::make_unique<std::unordered_map<std::string, std::string>>();
        }

        // Copy constructor
        Context(const Context &other) {
            protocol = other.protocol;
            target = other.target;
            params = std::make_unique<std::unordered_map<std::string, std::string>>();
            for (auto &param: *other.params) {
                (*params)[param.first] = param.second;
            }
        }

        // Copy assignment
        Context &operator=(const Context &other) {
            protocol = other.protocol;
            target = other.target;
            for (auto &param: *other.params) {
                (*params)[param.first] = param.second;
            }
            return *this;
        }

        std::string get(const std::string &key) {
            return (*params)[key];
        }

        void set(const std::string &key, const std::string &value) {
            (*params)[key] = value;
        }


    private:
        std::unique_ptr<std::unordered_map<std::string, std::string>> params;
    };

    class Middleware {
    public:
        virtual Middleware *setNext(Middleware *n) = 0;

        virtual std::pair<Context, std::exception *> operator()(Context context) = 0;
    };

    class BaseMiddleware : public Middleware {
    private:
        Middleware *next;
    public:
        Middleware *setNext(Middleware *n) override {
            next = n;
            return this;
        }

        std::pair<Context, std::exception *> operator()(Context context) override {
            return next->operator()(context);
        }
    };

    class MiddlewareCollector {
    private:
        std::vector<Freighter::Middleware *> middlewares;
    public:
        void use(Freighter::Middleware *middleware) {
            middlewares.push_back(middleware);
        }

        std::pair<Freighter::Context, std::exception *> exec(
                Freighter::Context context,
                Freighter::Middleware *finalizer
        ) {
            if (middlewares.empty()) return finalizer->operator()(context);
            for (int i = 0; i < middlewares.size(); i++) {
                auto &mw = middlewares[i];
                if (i == middlewares.size() - 1) mw->setNext(finalizer);
                else mw->setNext(middlewares[i + 1]);
            }
            return middlewares[0]->operator()(context);
        }
    };


    template<typename response_t, typename request_t, typename err_t>
    class UnaryClient {
    public:
        virtual void use(Middleware *middleware) = 0;

        virtual std::pair<response_t, err_t> send(const std::string &target, request_t &request) = 0;
    };

    template<typename response_t, typename request_t, typename err_t>
    class Stream {
    public:
        virtual err_t send(request_t &request) = 0;

        virtual std::pair<response_t, err_t> receive() = 0;

        virtual err_t closeSend() = 0;
    };

    template<typename response_t, typename request_t, typename err_t>
    class StreamClient {
    public:
        virtual void use(Middleware *middleware) = 0;

        virtual Stream<response_t, request_t, err_t> *stream(const std::string &target) = 0;
    };
}
