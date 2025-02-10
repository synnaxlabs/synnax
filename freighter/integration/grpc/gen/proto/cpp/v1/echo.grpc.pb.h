// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Generated by the gRPC C++ plugin.
// If you make any local change, they will be lost.
// source: v1/echo.proto
#ifndef GRPC_v1_2fecho_2eproto__INCLUDED
#define GRPC_v1_2fecho_2eproto__INCLUDED

#include "v1/echo.pb.h"

#include <functional>
#include <grpcpp/generic/async_generic_service.h>
#include <grpcpp/support/async_stream.h>
#include <grpcpp/support/async_unary_call.h>
#include <grpcpp/support/client_callback.h>
#include <grpcpp/client_context.h>
#include <grpcpp/completion_queue.h>
#include <grpcpp/support/message_allocator.h>
#include <grpcpp/support/method_handler.h>
#include <grpcpp/impl/proto_utils.h>
#include <grpcpp/impl/rpc_method.h>
#include <grpcpp/support/server_callback.h>
#include <grpcpp/impl/server_callback_handlers.h>
#include <grpcpp/server_context.h>
#include <grpcpp/impl/service_type.h>
#include <grpcpp/support/status.h>
#include <grpcpp/support/stub_options.h>
#include <grpcpp/support/sync_stream.h>

namespace integration {
namespace v1 {

class EchoService final {
 public:
  static constexpr char const* service_full_name() {
    return "integration.v1.EchoService";
  }
  class StubInterface {
   public:
    virtual ~StubInterface() {}
    virtual ::grpc::Status Exec(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::integration::v1::Message* response) = 0;
    std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::integration::v1::Message>> AsyncExec(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::integration::v1::Message>>(AsyncExecRaw(context, request, cq));
    }
    std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::integration::v1::Message>> PrepareAsyncExec(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReaderInterface< ::integration::v1::Message>>(PrepareAsyncExecRaw(context, request, cq));
    }
    class async_interface {
     public:
      virtual ~async_interface() {}
      virtual void Exec(::grpc::ClientContext* context, const ::integration::v1::Message* request, ::integration::v1::Message* response, std::function<void(::grpc::Status)>) = 0;
      virtual void Exec(::grpc::ClientContext* context, const ::integration::v1::Message* request, ::integration::v1::Message* response, ::grpc::ClientUnaryReactor* reactor) = 0;
    };
    typedef class async_interface experimental_async_interface;
    virtual class async_interface* async() { return nullptr; }
    class async_interface* experimental_async() { return async(); }
   private:
    virtual ::grpc::ClientAsyncResponseReaderInterface< ::integration::v1::Message>* AsyncExecRaw(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::grpc::CompletionQueue* cq) = 0;
    virtual ::grpc::ClientAsyncResponseReaderInterface< ::integration::v1::Message>* PrepareAsyncExecRaw(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::grpc::CompletionQueue* cq) = 0;
  };
  class Stub final : public StubInterface {
   public:
    Stub(const std::shared_ptr< ::grpc::ChannelInterface>& channel, const ::grpc::StubOptions& options = ::grpc::StubOptions());
    ::grpc::Status Exec(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::integration::v1::Message* response) override;
    std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::integration::v1::Message>> AsyncExec(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::integration::v1::Message>>(AsyncExecRaw(context, request, cq));
    }
    std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::integration::v1::Message>> PrepareAsyncExec(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::grpc::CompletionQueue* cq) {
      return std::unique_ptr< ::grpc::ClientAsyncResponseReader< ::integration::v1::Message>>(PrepareAsyncExecRaw(context, request, cq));
    }
    class async final :
      public StubInterface::async_interface {
     public:
      void Exec(::grpc::ClientContext* context, const ::integration::v1::Message* request, ::integration::v1::Message* response, std::function<void(::grpc::Status)>) override;
      void Exec(::grpc::ClientContext* context, const ::integration::v1::Message* request, ::integration::v1::Message* response, ::grpc::ClientUnaryReactor* reactor) override;
     private:
      friend class Stub;
      explicit async(Stub* stub): stub_(stub) { }
      Stub* stub() { return stub_; }
      Stub* stub_;
    };
    class async* async() override { return &async_stub_; }

   private:
    std::shared_ptr< ::grpc::ChannelInterface> channel_;
    class async async_stub_{this};
    ::grpc::ClientAsyncResponseReader< ::integration::v1::Message>* AsyncExecRaw(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::grpc::CompletionQueue* cq) override;
    ::grpc::ClientAsyncResponseReader< ::integration::v1::Message>* PrepareAsyncExecRaw(::grpc::ClientContext* context, const ::integration::v1::Message& request, ::grpc::CompletionQueue* cq) override;
    const ::grpc::internal::RpcMethod rpcmethod_Exec_;
  };
  static std::unique_ptr<Stub> NewStub(const std::shared_ptr< ::grpc::ChannelInterface>& channel, const ::grpc::StubOptions& options = ::grpc::StubOptions());

  class Service : public ::grpc::Service {
   public:
    Service();
    virtual ~Service();
    virtual ::grpc::Status Exec(::grpc::ServerContext* context, const ::integration::v1::Message* request, ::integration::v1::Message* response);
  };
  template <class BaseClass>
  class WithAsyncMethod_Exec : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service* /*service*/) {}
   public:
    WithAsyncMethod_Exec() {
      ::grpc::Service::MarkMethodAsync(0);
    }
    ~WithAsyncMethod_Exec() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status Exec(::grpc::ServerContext* /*context*/, const ::integration::v1::Message* /*request*/, ::integration::v1::Message* /*response*/) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    void RequestExec(::grpc::ServerContext* context, ::integration::v1::Message* request, ::grpc::ServerAsyncResponseWriter< ::integration::v1::Message>* response, ::grpc::CompletionQueue* new_call_cq, ::grpc::ServerCompletionQueue* notification_cq, void *tag) {
      ::grpc::Service::RequestAsyncUnary(0, context, request, response, new_call_cq, notification_cq, tag);
    }
  };
  typedef WithAsyncMethod_Exec<Service > AsyncService;
  template <class BaseClass>
  class WithCallbackMethod_Exec : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service* /*service*/) {}
   public:
    WithCallbackMethod_Exec() {
      ::grpc::Service::MarkMethodCallback(0,
          new ::grpc::internal::CallbackUnaryHandler< ::integration::v1::Message, ::integration::v1::Message>(
            [this](
                   ::grpc::CallbackServerContext* context, const ::integration::v1::Message* request, ::integration::v1::Message* response) { return this->Exec(context, request, response); }));}
    void SetMessageAllocatorFor_Exec(
        ::grpc::MessageAllocator< ::integration::v1::Message, ::integration::v1::Message>* allocator) {
      ::grpc::internal::MethodHandler* const handler = ::grpc::Service::GetHandler(0);
      static_cast<::grpc::internal::CallbackUnaryHandler< ::integration::v1::Message, ::integration::v1::Message>*>(handler)
              ->SetMessageAllocator(allocator);
    }
    ~WithCallbackMethod_Exec() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status Exec(::grpc::ServerContext* /*context*/, const ::integration::v1::Message* /*request*/, ::integration::v1::Message* /*response*/) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    virtual ::grpc::ServerUnaryReactor* Exec(
      ::grpc::CallbackServerContext* /*context*/, const ::integration::v1::Message* /*request*/, ::integration::v1::Message* /*response*/)  { return nullptr; }
  };
  typedef WithCallbackMethod_Exec<Service > CallbackService;
  typedef CallbackService ExperimentalCallbackService;
  template <class BaseClass>
  class WithGenericMethod_Exec : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service* /*service*/) {}
   public:
    WithGenericMethod_Exec() {
      ::grpc::Service::MarkMethodGeneric(0);
    }
    ~WithGenericMethod_Exec() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status Exec(::grpc::ServerContext* /*context*/, const ::integration::v1::Message* /*request*/, ::integration::v1::Message* /*response*/) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
  };
  template <class BaseClass>
  class WithRawMethod_Exec : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service* /*service*/) {}
   public:
    WithRawMethod_Exec() {
      ::grpc::Service::MarkMethodRaw(0);
    }
    ~WithRawMethod_Exec() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status Exec(::grpc::ServerContext* /*context*/, const ::integration::v1::Message* /*request*/, ::integration::v1::Message* /*response*/) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    void RequestExec(::grpc::ServerContext* context, ::grpc::ByteBuffer* request, ::grpc::ServerAsyncResponseWriter< ::grpc::ByteBuffer>* response, ::grpc::CompletionQueue* new_call_cq, ::grpc::ServerCompletionQueue* notification_cq, void *tag) {
      ::grpc::Service::RequestAsyncUnary(0, context, request, response, new_call_cq, notification_cq, tag);
    }
  };
  template <class BaseClass>
  class WithRawCallbackMethod_Exec : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service* /*service*/) {}
   public:
    WithRawCallbackMethod_Exec() {
      ::grpc::Service::MarkMethodRawCallback(0,
          new ::grpc::internal::CallbackUnaryHandler< ::grpc::ByteBuffer, ::grpc::ByteBuffer>(
            [this](
                   ::grpc::CallbackServerContext* context, const ::grpc::ByteBuffer* request, ::grpc::ByteBuffer* response) { return this->Exec(context, request, response); }));
    }
    ~WithRawCallbackMethod_Exec() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable synchronous version of this method
    ::grpc::Status Exec(::grpc::ServerContext* /*context*/, const ::integration::v1::Message* /*request*/, ::integration::v1::Message* /*response*/) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    virtual ::grpc::ServerUnaryReactor* Exec(
      ::grpc::CallbackServerContext* /*context*/, const ::grpc::ByteBuffer* /*request*/, ::grpc::ByteBuffer* /*response*/)  { return nullptr; }
  };
  template <class BaseClass>
  class WithStreamedUnaryMethod_Exec : public BaseClass {
   private:
    void BaseClassMustBeDerivedFromService(const Service* /*service*/) {}
   public:
    WithStreamedUnaryMethod_Exec() {
      ::grpc::Service::MarkMethodStreamed(0,
        new ::grpc::internal::StreamedUnaryHandler<
          ::integration::v1::Message, ::integration::v1::Message>(
            [this](::grpc::ServerContext* context,
                   ::grpc::ServerUnaryStreamer<
                     ::integration::v1::Message, ::integration::v1::Message>* streamer) {
                       return this->StreamedExec(context,
                         streamer);
                  }));
    }
    ~WithStreamedUnaryMethod_Exec() override {
      BaseClassMustBeDerivedFromService(this);
    }
    // disable regular version of this method
    ::grpc::Status Exec(::grpc::ServerContext* /*context*/, const ::integration::v1::Message* /*request*/, ::integration::v1::Message* /*response*/) override {
      abort();
      return ::grpc::Status(::grpc::StatusCode::UNIMPLEMENTED, "");
    }
    // replace default version of method with streamed unary
    virtual ::grpc::Status StreamedExec(::grpc::ServerContext* context, ::grpc::ServerUnaryStreamer< ::integration::v1::Message,::integration::v1::Message>* server_unary_streamer) = 0;
  };
  typedef WithStreamedUnaryMethod_Exec<Service > StreamedUnaryService;
  typedef Service SplitStreamedService;
  typedef WithStreamedUnaryMethod_Exec<Service > StreamedService;
};

}  // namespace v1
}  // namespace integration


#endif  // GRPC_v1_2fecho_2eproto__INCLUDED
