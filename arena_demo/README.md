# Go Hybrid Architecture Demo

这是一个展示 Go 语言 **"混合架构" (Hybrid Architecture)** 的最小化示例。

它演示了如何在同一个 Go 进程中，通过 **Arena (手动内存管理)** 和 **LockOSThread (独占线程)** 来实现 C/C++ 级别的极致性能，同时保留 Go 在业务层的开发效率。

## 目录结构

*   `main.go`: 业务层入口 (HTTP Server, JSON, etc.) -> **Go World**
*   `pkg/core/`: 核心引擎 (Zero GC, Pinned Thread) -> **C World**
*   `pkg/arena/`: 简单的 Arena 内存分配器实现
*   `pkg/fastqueue/`: 连接两个世界的无锁队列 (Bridge)

## 核心原理

1.  **Go World**: 处理 IO 密集型任务（如 HTTP 请求），享受 Runtime 便利。
2.  **C World**: 处理 CPU 密集型任务（如撮合、计算），绕过 GC 和调度器。
3.  **Bridge**: 使用无锁队列传递数据，避免锁竞争。

## 如何运行

直接使用 Go 运行即可（无需重新编译 Go 编译器）：

```bash
go run main.go
```

## 测试

启动后，访问：

```bash
curl "http://localhost:8080/calc?val=50"
```

你将看到结果 `Result: 100`。
整个计算过程在 `core` 包中完成，该过程：
*   **无 GC**: 使用 Arena 分配内存，用完即重置。
*   **无调度**: 运行在独占的 OS 线程上。
