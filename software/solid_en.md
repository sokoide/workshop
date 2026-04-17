# SOLID Principles (Learning Design Principles with Go)

SOLID is a set of five design principles for building code that is easier to change and test. In Go, SOLID is usually achieved through **small interfaces** and **clear responsibility boundaries**, rather than class inheritance.

---

## S: Single Responsibility Principle

A type or function should have only one reason to change.

### Why this matters

When responsibilities are mixed, one change can impact unrelated concerns.

Example: `ProcessOrder` handles pricing, DB persistence, and email notification in one function.

* Even if you only change tax rules, you must validate impacts on notification behavior in the same function.
* To test DB-failure behavior, you may still need SMTP mocks.
* A release for email template updates can accidentally introduce bugs in pricing logic.

In short, your change surface becomes hard to reason about, increasing both change cost and regression risk.

Put differently, S aims to **keep one change localized to one place** so unrelated features are less likely to break.

### Bad Example

```go
// Handles pricing, persistence, and notification in one place.
func ProcessOrder(o Order, db *sql.DB, smtp SMTPClient) error {
	// pricing
	// save to DB
	// send email
	return nil
}
```text

```go
// Example where tax rule changes require checking notification formatting too.
func ProcessOrder(o Order, db *sql.DB, smtp SMTPClient) error {
	taxRate := 0.10
	taxLabel := "standard tax"
	if o.HasReducedTaxItem {
		taxRate = 0.08
		taxLabel = "reduced tax"
	}
	o.Total = int(float64(o.Subtotal) * (1 + taxRate))

	if _, err := db.Exec("INSERT INTO orders(total) VALUES(?)", o.Total); err != nil {
		return err
	}

	// Notification text depends on pricing details.
	body := fmt.Sprintf("Order total: %d (%s)", o.Total, taxLabel)
	return smtp.Send(o.Email, body)
}
```text

In this shape, tax rule changes also require checking/adjusting notification text behavior (`taxLabel`).
You may not always edit notification code directly, but the impact-analysis scope is still wider.

### Improved Example

```go
// domain/order_pricer.go
type PriceResult struct {
	Total    int
	TaxLabel string
}

type OrderPricer interface {
	Price(o Order) PriceResult
}

// domain/order_repository.go
type OrderRepository interface {
	Save(ctx context.Context, o Order) error
}

// domain/notifier.go (Output Port)
type Notifier interface {
	SendOrderCreated(ctx context.Context, to, body string) error
}

// usecase/create_order.go
type OrderUseCase struct {
	pricer   OrderPricer
	repo     OrderRepository
	notifier Notifier
}

func NewOrderUseCase(
	pricer OrderPricer,
	repo OrderRepository,
	notifier Notifier,
) *OrderUseCase {
	return &OrderUseCase{
		pricer:   pricer,
		repo:     repo,
		notifier: notifier,
	}
}

func (uc *OrderUseCase) Execute(ctx context.Context, o Order) error {
	price := uc.pricer.Price(o) // Domain rule
	o.Total = price.Total
	if err := uc.repo.Save(ctx, o); err != nil {
		return err
	}
	// Same notification format as the bad example.
	body := fmt.Sprintf("Order total: %d (%s)", price.Total, price.TaxLabel)
	return uc.notifier.SendOrderCreated(ctx, o.Email, body) // via Domain Port
}

// infra/notifier/mail_notifier.go
type MailNotifier struct {
	client SMTPClient
}

func (n MailNotifier) SendOrderCreated(ctx context.Context, to, body string) error {
	return n.client.Send(ctx, to, body)
}
```text

With separated responsibilities, pricing logic can be tested independently and notification implementations can be swapped independently.
Tax-rule changes (10% -> 12%, reduced-tax condition changes, etc.) only touch `OrderPricer`; `MailNotifier` remains unchanged.

### S in Clean Architecture

* **Domain:** pricing rules (Entity/Domain Service)
* **UseCase:** order-creation flow (price -> save -> notify)
* **Infra Adapters:** DB repository implementation, mail notifier implementation
* **Framework:** HTTP handlers / CLI invoke UseCase

The key point is that fix locations are separated by change reason.

* Tax-rule change: Domain only
* DB product change: Infra Adapters only
* API input/output format change: Framework only
* Email provider change (SMTP -> SES): Infra Adapters only

---

## O: Open/Closed Principle

Software entities should be open for extension, closed for modification.

### Why this matters

If you keep adding cases to existing branching logic like `switch paymentType`, every new payment method forces edits to existing code paths.

* Regression-test scope grows for every new method.
* A mistaken edit in an existing branch can break another payment path.

### Bad Example

```go
func Checkout(ctx context.Context, paymentType string, amount int) error {
	switch paymentType {
	case "card":
		return chargeCard(ctx, amount)
	case "paypay":
		return chargePayPay(ctx, amount)
	// Adding ApplePay etc. requires editing this existing function.
	default:
		return errors.New("unsupported payment type")
	}
}
```text

### Example: Adding a Payment Method

```go
type PaymentMethod interface {
	Pay(ctx context.Context, amount int) error
}

type CardPayment struct{}

func (CardPayment) Pay(ctx context.Context, amount int) error { return nil }

type PayPayPayment struct{}

func (PayPayPayment) Pay(ctx context.Context, amount int) error { return nil }

func Checkout(ctx context.Context, pm PaymentMethod, amount int) error {
	return pm.Pay(ctx, amount)
}
```text

A new payment method can be added by implementing `PaymentMethod`, without modifying existing `Checkout` logic.

In Clean Architecture terms, `PaymentMethod` is a UseCase-side port, while implementations like `CardPayment` live in Infra Adapters.

---

## L: Liskov Substitution Principle

Implementations should be replaceable without breaking expected behavior.

### Why this matters

Even with the same interface, behavior mismatch across implementations causes environment-specific bugs.

* Contract mismatch (e.g., one impl returns `nil, nil`, another returns `ErrNotFound`).
* Callers start adding implementation-specific branching, defeating abstraction.

### Bad Example: Same Interface, Broken Contract

```go
type PaymentGateway interface {
	Charge(ctx context.Context, amount int) error
}

type SafeGateway struct{}

func (SafeGateway) Charge(ctx context.Context, amount int) error {
	if amount <= 0 {
		return errors.New("invalid amount")
	}
	return nil
}

type PanicGateway struct{}

func (PanicGateway) Charge(ctx context.Context, amount int) error {
	if amount <= 0 {
		panic("invalid amount") // contract violation
	}
	return nil
}

func Checkout(ctx context.Context, gw PaymentGateway, amount int) error {
	// Caller expects failures to be returned as error.
	return gw.Charge(ctx, amount)
}
```text

`Checkout` is written under the assumption that failures are returned as `error`.
Replacing with `PanicGateway` breaks that assumption, so safe substitution no longer holds.

In Clean Architecture terms, every adapter implementing the same output port must keep the same contract (return shape, error semantics, timeout behavior).

---

## I: Interface Segregation Principle

Prefer small, focused interfaces over large ones.

### Why this matters

Large interfaces create unnecessary dependencies and heavier mock implementations.

* A read-only test may still need dummy implementations for `Create/Update/Delete`.
* Changes spread widely and trigger cascading compile errors.

### Bad Example

```go
type UserService interface {
	Create(ctx context.Context, u User) error
	Update(ctx context.Context, u User) error
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (User, error)
	List(ctx context.Context) ([]User, error)
}
```text

### Improved Example

```go
type UserFinder interface {
	Find(ctx context.Context, id string) (User, error)
}

type UserCreator interface {
	Create(ctx context.Context, u User) error
}
```text

A read-only use case depends only on `UserFinder`, minimizing coupling.

In Clean Architecture, input ports are also easier to reason about when split per use case (e.g., `CreateUserUseCase`, `GetUserUseCase`).

---

## D: Dependency Inversion Principle

High-level modules (UseCase) should not depend directly on low-level modules (DB, external APIs); both should depend on abstractions.

### Why this matters

If a use case depends directly on DB clients or SDKs, technology changes leak into business logic.

* MySQL -> PostgreSQL migration may force broad UseCase changes.
* Unit testing use cases becomes harder (real DB/API dependencies).

### Example

```go
// abstraction defined on the usecase side

type InventoryGateway interface {
	Reserve(ctx context.Context, sku string, qty int) error
}

type CreateOrderUseCase struct {
	inv InventoryGateway
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, sku string, qty int) error {
	return uc.inv.Reserve(ctx, sku, qty)
}
```text

When `infra` implements `InventoryGateway`, UseCase remains isolated from external-technology details.

Keeping the Clean Architecture dependency direction (outer -> inner) protects business rules from technology churn.

---

## Summary: Why Use SOLID

SOLID is not just about writing "clean-looking" code. It is a practical set of rules to reduce:

* change impact scope
* test setup cost
* production risks caused by implementation mismatch

---

## Practical Go Tips for SOLID

* Define interfaces on the **consumer side**, not the implementation side.
* Pass `context.Context` as the first parameter for I/O-bound functions.
* Keep error contracts consistent across implementations (e.g., `ErrNotFound`).
* Start simple, then introduce abstraction when duplication or change pressure appears.
