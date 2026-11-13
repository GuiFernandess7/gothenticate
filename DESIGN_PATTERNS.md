# Design Patterns — Gothenticate API

Checkpoint dos padrões aplicados neste projeto, com a dor que cada um resolve e um pseudocódigo de como atua.

---

## 1. Strategy — Cálculo de Frete

**Onde:** `internal/modules/order/domain/shippingCalculator.go`

**A dor:**
O cálculo de frete muda por transportadora. Sem o padrão, o service teria um `switch` gigante com a lógica de cada transportadora misturada. Adicionar uma nova transportadora significa mexer no código existente — risco de quebrar o que já funciona.

**O que resolve:**
Cada transportadora vira uma struct com seu próprio algoritmo de cálculo. O service não sabe qual é — só chama `Calculate`. Adicionar uma nova transportadora é criar uma nova struct e um novo `case` no factory, sem tocar no resto.

**Pseudocódigo:**
```
interface ShippingCalculator
    Calculate(order) → (cost, error)

struct PostalShipping  → implementa Calculate com lógica dos Correios
struct ExpressShipping → implementa Calculate com lógica do Sedex
struct DroneShipping   → implementa Calculate (com limite de peso)

// Factory: seleciona a estratégia a partir de uma string
GetShippingCalculator("standard_mail") → PostalShipping{}
GetShippingCalculator("local_drone")   → DroneShipping{}

// Service: não sabe qual transportadora é
calculator = GetShippingCalculator(request.Carrier)
cost = calculator.Calculate(orderDetails)
```

---

## 2. Factory Method — Seleção de Estratégia

**Onde:** `GetShippingCalculator` em `shippingCalculator.go` / `stateFromLabel` em `orderStatus.go` e `auth/domain.go`

**A dor:**
Quem precisa de uma `ShippingCalculator` ou de um `OrderStateInterface` não deveria saber como instanciar cada tipo concreto. Sem o factory, o `switch` de instanciação vaza para o handler ou service, acoplando quem usa ao detalhe de como construir.

**O que resolve:**
Centraliza a lógica de criação num único lugar. O resto do código recebe uma interface pronta e não sabe (nem precisa saber) qual struct concreta está por baixo.

**Pseudocódigo:**
```
// Quem chama só conhece a interface
func GetShippingCalculator(tipo string) (ShippingCalculator, error)
    switch tipo
        "standard_mail" → return PostalShipping{}
        "local_drone"   → return DroneShipping{}
        default         → return nil, ErrUnknownCarrier

// Mesmo padrão para estados
func stateFromLabel(label OrderStatus) OrderStateInterface
    switch label
        "pending"  → return &StatusPending{}
        "paid"     → return &StatusPaid{}
        default    → return &StatusPending{}
```

---

## 3. State — Fluxo de Status do Pedido

**Onde:** `internal/modules/order/domain/orderStatus.go`

**A dor:**
O pedido passa por várias etapas (`pending → paid → preparing → sent → received`). Sem o padrão, cada ação teria um `if status == "pending" { ... } else if status == "paid" { ... }` espalhado pelo código. Adicionar um novo estado ou mudar uma regra de transição exige rastrear todos esses `ifs`.

**O que resolve:**
Cada estado vira uma struct que sabe o que pode e o que não pode fazer a partir dele. O handler chama `pedido.Pagar()` — se o estado atual permite, a transição acontece; se não, retorna erro. Nenhum `if` no handler.

**Pseudocódigo:**
```
interface OrderStateInterface
    Pagar(state, order) → error
    Enviar(state, order) → error
    ConfirmarEntrega(state, order) → error
    Cancelar(state, order) → error

struct StatusPending
    Pagar()          → muda estado para StatusPaid, notifica observers ✓
    Enviar()         → ErrInvalidOperation ✗
    ConfirmarEntrega → ErrInvalidOperation ✗
    Cancelar()       → muda estado para StatusCanceled, notifica observers ✓

struct StatusPaid
    Pagar()          → ErrInvalidOperation ✗
    Enviar()         → muda estado para StatusPreparing, notifica observers ✓
    ...

// Handler: zero ifs sobre status
err = orderState.Pagar(&order)
if err != nil → retorna 400 com err.Error()
// quem decidiu se era válido foi o estado, não o handler
```

---

## 4. Observer — Notificações em Mudança de Status

**Onde:** `internal/modules/order/domain/notifiyOrder.go`

**A dor:**
Quando um pedido muda de status, vários sistemas precisam ser notificados: estoque, fiscal, e-mail. Sem o padrão, o service chamaria cada um diretamente — acoplamento forte. Adicionar um novo sistema significa mexer no service.

**O que resolve:**
Cada sistema vira um observer registrado no pedido. Quando uma transição de estado acontece, o State chama `notify()` automaticamente — o service não sabe quem está ouvindo. Adicionar um novo observer é criar uma struct e registrá-la na inicialização.

**Pseudocódigo:**
```
interface OrderObserver
    OnStatusChanged(order)

struct NotificadorEstoque → OnStatusChanged: POST para serviço de estoque
struct NotificadorFiscal  → OnStatusChanged: POST para serviço fiscal
struct NotificadorEmail   → OnStatusChanged: envia e-mail de confirmação

// Registro na inicialização
orderState = NewOrderStatus(
    "pending",
    &NotificadorEstoque{},
    &NotificadorFiscal{},
    &NotificadorEmail{},
)

// Disparo automático dentro do State, transparente para o handler
func (s *StatusPending) Pagar(state, order)
    state.status = &StatusPaid{}
    state.notify(order)   ← itera pelos observers e chama OnStatusChanged
    return nil
```

---

## 5. Decorator (funcional) — Middlewares HTTP

**Onde:** `internal/middleware/`

**A dor:**
Autenticação JWT, rate limiting e headers de segurança precisam rodar em várias rotas. Sem o padrão, cada handler repetiria esse código — ou receberia um service gigante com responsabilidades misturadas.

**O que resolve:**
Cada comportamento extra vira uma função que envolve o handler original sem modificá-lo. Os handlers ficam focados só na lógica de negócio. Remover ou reordenar um middleware não exige tocar nos handlers.

**Pseudocódigo:**
```
// Handler original — só lógica de negócio
func AddOrder(c Context) error
    // valida body, chama service, retorna JSON

// Cada decorator envolve o próximo sem saber o que é
func JWTAuth(next HandlerFunc) HandlerFunc
    return func(c)
        token = c.Header("Authorization")
        if inválido → return 401
        c.Set("user_id", claims.UserID)
        return next(c)   ← passa para o próximo

func RateLimiter(next HandlerFunc) HandlerFunc
    return func(c)
        if limite excedido → return 429
        return next(c)

// Composição na definição das rotas
router.POST("/orders", JWTAuth(RateLimiter(AddOrder)))
// ordem de execução: JWTAuth → RateLimiter → AddOrder
```

---

## Resumo

| Pattern | Onde aparece | Benefício principal |
|---|---|---|
| Strategy | Cálculo de frete | Trocar algoritmo sem mudar o service |
| Factory Method | Instanciação de estratégias e estados | Centraliza criação, desacopla quem usa |
| State | Fluxo de status do pedido e conta de usuário | Elimina `ifs` de transição espalhados |
| Observer | Notificações ao mudar status | Adiciona/remove ouvintes sem tocar no fluxo principal |
| Decorator (funcional) | Middlewares JWT, rate limit, segurança | Comportamento transversal sem poluir handlers |