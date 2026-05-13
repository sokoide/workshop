# Clean Architecture Essentials: Understanding the Core with a Minimum Example (WS8)

In this workshop, you will experience the design philosophy of Clean Architecture (4-layer) and its "resilience to change" through a "User Greeting App" consisting of only about 100 lines of code.

## 1. Overview

The essence of Clean Architecture lies in **"Control of Dependencies."** The goal is to point dependencies inward so that business rules (Domain) are not contaminated by external concerns (databases, web frameworks, etc.).

### App Features

- Receives an HTTP request (`GET /?id=1`)
- Retrieves the username for the specified ID from a database (in-memory this time)
- Generates and returns a greeting: `Hello, [Name]!`

---

## 2. Directory Structure and Dependency Rules

Dependencies always point **from the outside in**.

```text
software/assets/greeting/
├── domain/       (Layer 1: Innermost) Business rules, Entities, Ports (Interfaces)
├── usecase/      (Layer 2) Use cases (Scenarios)
├── infra/        (Layer 3) Infrastructure implementation details (Adapters)
├── framework/    (Layer 4: Outermost) HTTP Handlers, External libraries
└── main.go       (Composition Root) Assembling all layers (Dependency Injection)
```

---

## 3. Explanation of Each Layer

### 3-1. Domain Layer: The "Core" of the System

The `domain/` directory defines knowledge that is invariant to this system.

- `user.go`: The concept of a "User."
- `repository.go`: The "window" (Port) for how to retrieve a user. Defined as an interface; concrete database operations are not written here.

### 3-2. UseCase Layer: The "Steps" of a Scenario

The `usecase/` directory contains business logic that defines "what to do and in what order."

- `GreetingUseCase` assembles the greeting string through the `UserRepository` "window" without knowing where the data comes from.

### 3-3. Infra Adapter Layer: Technical "Details"

The `infra/` directory implements the "contents" of the interfaces defined in the Domain layer.

- This time we use `MemoryUserRepo` to hold data in memory, but you can swap this with MySQL or an external API without breaking other layers.

### 3-4. Framework Layer: "Contact Point" with the Outside

The `framework/` directory handles communication protocols like HTTP.

- It focuses solely on extracting the ID from the request, passing it to the UseCase, and returning the result as a response.

---

## 4. Workshop: Experience "Resilience to Change"

### Step 1: Start the App

Run the following in your terminal to start the server:

```bash
cd software/assets/greeting
go run main.go
```

Send a request from another terminal to verify the behavior:

```bash
curl "http://localhost:8080?id=1"
# Output: Hello, Alice!
```

### Step 2: Run Unit Tests

A benefit of Clean Architecture is the ability to swap external environments (like DBs) with Mocks and test business logic at high speed.

```bash
go test ./usecase/...
```

Read `usecase/greeting_test.go` to see how the repository is swapped with a fake (Mock).

### Step 3: [Exercise] Change Business Rules

Open `usecase/greeting.go` and try changing the greeting message (e.g., to Japanese: `こんにちは、[Name]さん！`).

- **Key Point**: Observe that you don't need to touch any code in `infra` or `framework` for this change.

---

## 5. Summary

1. **Domain depends on nothing**: Isolate the core knowledge of the business.
2. **UseCase talks through Ports (Interfaces)**: Logic can be completed without knowing implementation details (DB, etc.).
3. **Dependency Injection (DI)**: By snapping each part together in `main.go`, the entire system becomes operational.

This is the first step toward a "Clean" design that is highly maintainable, easy to test, and resilient to change.
