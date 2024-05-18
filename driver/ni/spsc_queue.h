#include <atomic>
#include <memory>
#include <optional>


// A thread safe, lock free single producer single consumer queue (!!! NOT MPMC !!!)
template <typename T>
class SPSCQueue {
    public:

    SPSCQueue() :  head(new Node), tail(head.load(std::memory_order_relaxed)) {}

    ~SPSCQueue() {
        while (Node* old_head = head.load(std::memory_order_relaxed)) {
            head.store(old_head->next, std::memory_order_relaxed);
            delete old_head;
        }
    }

    // producer
    void enqueue(const T& item){
        Node* new_node = new Node(item);
        tail.load(std::memory_order_relaxed)->next = new_node;
        tail.store(new_node, std::memory_order_relaxed);
    }

    // consumer
    std::optional<T> dequeue(){

        Node* old_head = head.load(std::memory_order_relaxed);
        Node* next = old_head->next;
        if (!next) return std::nullopt;

        std::optional<T> result = next->data;
        head.store(next, std::memory_order_relaxed);
        delete old_head;
        return result;
    }

    private:
    typedef struct Node {
        std::optional<T> data;
        std::atomic<Node*> next;
        Node() : next(nullptr) {}
        Node(const T& data) : data(data), next(nullptr) {}
    } Node;

    std::atomic<Node*> head;
    std::atomic<Node*> tail;

};