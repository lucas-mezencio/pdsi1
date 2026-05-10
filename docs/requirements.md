# MedNotify - Requisitos e Documentação Técnica

## 1. Visão Geral do Sistema

MedNotify é uma plataforma de notificação de medicamentos projetada para ajudar pacientes a gerenciar sua medicação de forma segura e pontual. O sistema combina uma API REST, messaging event-driven com Watermill, agendamento via Redis e notificações push via Firebase FCM.

**Stack Tecnológico:**
- Go 1.25+
- Watermill (event streaming)
- Redis (agendamento de notificações)
- PostgreSQL (persistência)
- Firebase FCM (push notifications)
- Clean Architecture + CQRS

---

## 2. Diagrama de Arquitetura (PlantUML)

```plantuml
@startuml MedNotify_Architecture
!theme plain

package "Presentation Layer" {
  [REST API] as REST_API
}

package "Application Layer" {
  [Command Handlers] as CMD_HANDLERS
  [Query Handlers] as QRY_HANDLERS
}

package "Domain Layer" {
  entity User
  entity Doctor
  entity Prescription
  entity Dose
  entity Notification
}

package "Infrastructure Layer" {
  [PostgreSQL Repository] as PG_REPO
  [Redis Scheduler] as REDIS_SCHED
  [Firebase FCM] as FCM
  [Watermill Publisher] as WM_PUB
}

REST_API --> CMD_HANDLERS : Command
REST_API --> QRY_HANDLERS : Query
CMD_HANDLERS --> WM_PUB : Publish Event
WM_PUB --> REDIS_SCHED : Schedule Notification
REDIS_SCHED --> FCM : Send Push
CMD_HANDLERS --> PG_REPO : Persist
QRY_HANDLERS --> PG_REPO : Query
PG_REPO --> PostgreSQL

@enduml
```

---

## 3. Diagrama de Entidade-Relacionamento (PlantUML)

```plantuml
@startuml MedNotify_ER
!theme plain

entity "users" as users {
  * id : UUID <<PK>>
  --
  * name : VARCHAR(255)
  * email : VARCHAR(255) <<UNIQUE>>
  * device_token : VARCHAR(512)
  * created_at : TIMESTAMP
  * updated_at : TIMESTAMP
}

entity "doctors" as doctors {
  * id : UUID <<PK>>
  --
  * name : VARCHAR(255)
  * crm : VARCHAR(50) <<UNIQUE>>
  * specialty : VARCHAR(100)
  * created_at : TIMESTAMP
}

entity "prescriptions" as prescriptions {
  * id : UUID <<PK>>
  --
  * user_id : UUID <<FK>>
  * doctor_id : UUID <<FK>>
  * medication_name : VARCHAR(255)
  * dosage : VARCHAR(100)
  * frequency : VARCHAR(100)
  * start_date : DATE
  * end_date : DATE
  * created_at : TIMESTAMP
}

entity "doses" as doses {
  * id : UUID <<PK>>
  --
  * prescription_id : UUID <<FK>>
  * scheduled_time : TIMESTAMP
  * status : ENUM('scheduled','notified','taken','missed')
  * taken_at : TIMESTAMP
  * created_at : TIMESTAMP
}

entity "notifications" as notifications {
  * id : UUID <<PK>>
  --
  * dose_id : UUID <<FK>>
  * sent_at : TIMESTAMP
  * status : ENUM('pending','sent','failed')
  * error_message : TEXT
}

users ||--o{ prescriptions : has
doctors ||--o{ prescriptions : prescribes
prescriptions ||--o{ doses : contains
doses ||--o{ notifications : triggers

@enduml
```

---

## 4. Diagrama de Sequência - Fluxo de Notificação (PlantUML)

```plantuml
@startuml Notification_Flow
!theme plain

actor Patient as P
participant "REST API" as API
participant "Command Handler" as CMD
participant "Watermill" as WM
participant "Redis Scheduler" as REDIS
participant "Firebase FCM" as FCM

P -> API : POST /prescriptions (Create Prescription)
API -> CMD : Handle CreatePrescriptionCommand
CMD -> CMD : Validate Domain
CMD -> WM : Publish PrescriptionCreatedEvent
WM -> REDIS : Schedule Notification at dose_time
REDIS --> REDIS : Wait until scheduled_time

REDIS -> FCM : Send Push Notification
FCM -> P : "Time to take your medication"
P -> API : POST /doses/{id}/take
API -> CMD : Handle MarkDoseTakenCommand
CMD -> CMD : Update dose status to 'taken'
CMD -> WM : Publish DoseTakenEvent

@enduml
```

---

## 5. Diagrama de Estados - Ciclo de Vida da Dose (PlantUML)

```plantuml
@startuml Dose_State_Diagram
!theme plain

[*] --> Scheduled : Prescription created

Scheduled --> Notified : Notification sent
Notified --> Taken : Patient confirms
Notified --> Missed : Timeout (30 min)
Missed --> [*] : Grace period expired
Taken --> [*] : Dose completed

note right of Scheduled
  Dose time is approaching
end note

note right of Notified
  Push notification sent
  Waiting for patient confirmation
end note

note right of Missed
  Patient did not confirm
  Notify caregiver (future)
end note

@enduml
```

---

## 6. Diagrama de Classes - Clean Architecture (PlantUML)

```plantuml
@startuml Clean_Architecture_Classes
!theme plain

package "domain" {
  class User {
    + ID: UUID
    + Name: string
    + Email: string
    + DeviceToken: string
  }

  class Doctor {
    + ID: UUID
    + Name: string
    + CRM: string
    + Specialty: string
  }

  class Prescription {
    + ID: UUID
    + UserID: UUID
    + DoctorID: UUID
    + MedicationName: string
    + Dosage: string
    + Frequency: string
    + StartDate: time.Time
    + EndDate: time.Time
  }

  class Dose {
    + ID: UUID
    + PrescriptionID: UUID
    + ScheduledTime: time.Time
    + Status: DoseStatus
    + MarkAsTaken() error
  }

  enum DoseStatus {
    Scheduled
    Notified
    Taken
    Missed
  }
}

package "application" {
  interface "CommandHandler" as ICommandHandler {
    + Handle(ctx context.Context, cmd interface{}) error
  }

  interface "QueryHandler" as IQueryHandler {
    + Handle(ctx context.Context, query interface{}) interface{}
  }

  interface "Notifier" as INotifier {
    + Send(ctx context.Context, dose Dose) error
  }

  interface "Repository" as IRepository {
    + Save(ctx context.Context, entity interface{}) error
    + FindByID(ctx context.Context, id UUID) (interface{}, error)
  }
}

package "infrastructure" {
  class "PostgreSQLRepository" implements IRepository {
    - db: *sql.DB
  }

  class "FirebaseNotifier" implements INotifier {
    - client: *messaging.Client
  }

  class "RedisScheduler" {
    - client: *redis.Client
    + Schedule(ctx context.Context, dose Dose) error
  }
}

Dose --> DoseStatus
IRepository <|.. PostgreSQLRepository
INotifier <|.. FirebaseNotifier

@enduml
```

---

## 7. Marcação de Requisitos

| ID | Requisito | Status |
|----|-----------|--------|
| REQ-001 | API REST para CRUD de usuários | `implementado` |
| REQ-002 | API REST para CRUD de médicos | `implementado` |
| REQ-003 | API REST para CRUD de prescrições | `implementado` |
| REQ-004 | Eventos Watermill para domain events | `implementado` |
| REQ-005 | Agendamento Redis para notificações | `implementado` |
| REQ-006 | Integração Firebase FCM para push | `implementado` |
| REQ-007 | Padrão CQRS (Command/Query separation) | `implementado` |
| REQ-008 | Clean Architecture (domain/application/infrastructure) | `implementado` |
| REQ-009 | Testes unitários com cobertura > 70% | `não implementado` |
| REQ-010 | Testes de integração com build tag `integration` | `não implementado` |
| REQ-011 | Documentação PlantUML (este arquivo) | `implementado` |
| REQ-012 | LGPD Privacy by Design (minimização de dados) | `não documentado` |
| REQ-013 | Interface mobile para confirmação de dose | `fora de escopo (frontend)` |

---

## 8. Limites de Escopo

- **Frontend (Mobile/Web):** Fora de escopo para este projeto backend
- **Analytics/Relatórios:** Não escopo atual
- **Multi-idioma:** Não aplicável (projeto brasileiro)
- **Offline-first:** Não aplicável (sistema cloud)

---

## 9. Próximos Passos (Action Items)

- [ ] Implementar testes unitários (REQ-009)
- [ ] Implementar testes de integração (REQ-010)
- [ ] Documentar conformidade LGPD (REQ-012)

---

## 10. Status da Entrega

| Critério de Aceitação | Status |
|-----------------------|--------|
| Documento único em /docs | ✅ Concluído |
| Requisitos Funcionais e Não-Funcionais | ✅ Seção 7 |
| Diagrama de Classes | ✅ Seção 6 |
| Diagrama ER | ✅ Seção 3 |
| Diagrama de Sequência | ✅ Seção 4 |
| Diagrama de Estados | ✅ Seção 5 |
| Diagrama de Arquitetura (solution-level) | ✅ Seção 2 |
| Requisitos marcados (implementado/não impl./não doc./fora de escopo) | ✅ Seção 7 |
| PR para remote repo | ⚠️ Bloqueado — workspace sem git history |

**Aguardando:** Inicialização do repositório git para viabilizar PR aos artefatos de documentação.

---

*Documento gerado em: 2026-05-10*
*Versão: 1.1*