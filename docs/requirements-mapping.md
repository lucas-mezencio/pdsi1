# CareConnect — Mapeamento de Requisitos (PDSI2)

**Data:** 2026-05-10
**Projeto:** CareConnect / MedNotify — Backend API (Go)
**Contexto:** Cruzamento entre requisitos entregues no semestre passado (E1, E2, E3, apresentação HTML) e o código atual no branch `docs/diagrams`.

---

## 1. Resumo Executivo

O backend do CareConnect foi desenvolvido em Go com Clean Architecture + CQRS, usando PostgreSQL, Redis (Watermill + scheduling) e Firebase FCM. Os documentos entregue no semestre passado (E1/E2/E3) listam requisitos que nem sempre batem com a implementação atual — alguns foram implementados com nomes diferentes, outros foram omitidos, e uma parcela significativa dos gaps críticos identificados no sprint scope (MEZ-29) permanece sem implementação.

Este documento consolida todos os requisitos, mapeia cada um ao código existente, e classifica os gaps encontrados.

---

## 2. Legenda de Status

| Símbolo | Significado |
|---------|-------------|
| ✅ Implementado | Existe no código e funciona |
| ❌ Não Implementado | Sem código correspondente |
| ⚠️ Parcialmente Implementado | Implementado mas com lacunas |
| 📄 Documentado | Presente em algum documento do semestre passado |
| 🔲 Não Documentado | Requisito existente sem documentação |
| 🚫 Fora de Escopo | Descartado intencionalmente |

---

## 3. Tabela de Requisitos Consolidados

### 3.1 Requisitos Funcionais (Originários de E1.pdf)

| ID | Descrição | Origem | Código | Status Código | Status Docs | Localização no Código / Docs |
|----|-----------|--------|--------|--------------|-------------|---------------|
| RF01 | Cadastro de medicamentos | E1 | ✅ | ✅ Implementado | 📄 | `domain/prescription/medicament.go`, `domain/prescription/prescription.go` — medicaments são parte de prescriptions |
| RF02 | Cadastro de pacientes | E1 | ✅ | ✅ Implementado | 📄 | `domain/user/user.go` — usuários com `role=ELDERLY` |
| RF03 | Cadastro de médicos | E1 | ✅ | ✅ Implementado | 📄 | `domain/doctor/doctor.go` |
| RF04 | Cadastro de farmacêuticos | E1 | ❌ | ❌ Não Implementado | 📄 | Sem código — nenhuma entidade ou endpoint relacionado |
| RF05 | Cadastro de farmácias | E1 | ❌ | ❌ Não Implementado | 📄 | Sem código — nenhuma entidade ou endpoint relacionado |
| RF06 | Cadastro de unidades de saúde | E1 | ❌ | ❌ Não Implementado | 📄 | Sem código — nenhuma entidade ou endpoint relacionado |
| RF07 | Cadastro de familiares | E1 | ✅ | ✅ Implementado | 📄 | `domain/user/` + `invitation_commands.go` — Cuidadores (CAREGIVER) com sistema de convite via token |
| RF08 | Cadastro de regras de agendamento | E1 | ✅ | ✅ Implementado | 📄 | `medicament.go` — campos `frequency` (HH:MM), `times[]` (HH:MM), `doses` (int) |
| RF09 | Envio de lembretes de medicamentos | E1 | ✅ | ✅ Implementado | 📄 | `infrastructure/scheduler/worker.go` + `notification/firebase_sender.go` — Redis → Watermill → FCM |
| RF10 | Registro de tomada de medicamentos | E1 | ✅ | ✅ Implementado | 📄 | `domain/prescription/dose_record.go` + `dose_record_commands.go` — confirm/miss |
| RF11 | Relatórios de adherence | E1 | ❌ | ❌ Não Implementado | 📄 | Sem código. Sistema apenas registra `DoseRecord.Status` — não há agregação/relatório |
| RF12 | Notificar médicos sobre adherence dos pacientes | E1 | ❌ | ❌ Não Implementado | 📄 | Sem código. Caregivers são notificados, mas médicos não recebem alerts |

---

### 3.2 Requisitos Não-Funcionais (Originários de E1.pdf)

| ID | Descrição | Origem | Código | Status Código | Observação |
|----|-----------|--------|--------|--------------|------------|
| RNF01 | Disponibilidade 24/7 | E1 | — | 🚫 Não aplicável | Requisito de infraestrutura/deploy — backend não implementa disponibilidade por si só |
| RNF02 | Acessível via dispositivos móveis | E1 | — | 🚫 Fora de escopo | Mobile app é projeto separado (care-connect React Native) |
| RNF03 | Processar requisições em < 5 segundos | E1 | ✅ | ⚠️ Parcial | API Go com Redis/PostgreSQL — performance não medida formalmente |
| RNF04 | Armazenar dados de forma segura | E1 | ✅ | ⚠️ Parcial | PostgreSQL com UUIDs, campos sensíveis — sem criptografia em repouso documentada |
| RNF05 | Proteger dados sensíveis conforme LGPD | E1 | ⚠️ | 🔲 Não Documentado | Sem código de minimização/anonimização. `firebase_id` é opcional e não indexado null-only. `firebase_token` armazenado em texto plano. Nenhum endpoint de export/deletion de dados do usuário (LGPD Art. 18) |
| RNF06 | Funcionar offline | E1 | — | 🚫 Não aplicável | Requisito de mobile — não se aplica ao backend |

---

### 3.3 Features e Critérios de Aceitação (Originários de E2.pdf / Apresentação HTML)

| ID | Feature | Origem | Código | Status Código | Observação |
|----|---------|--------|--------|--------------|------------|
| F01 | Registro e autenticação de usuários | E2 | ✅ | ✅ Implementado | `extended_server.go` — `/auth/register`, `/auth/login` via Firebase Auth |
| F02 | Catálogo de medicamentos (CRUD) | E2 | ✅ | ✅ Implementado | CRUD implementado via Prescription + Medicament |
| F03 | Gestão de prescrições por médicos | E2 | ✅ | ✅ Implementado | `prescription_commands.go` — `CreatePrescription`, `Update`, etc. |
| F04 | Gestão de agendamentos | E2 | ✅ | ✅ Implementado | `RedisScheduler.Schedule` + `medicament.Times/Frequency` |
| F05 | Geração de lembretes | E2 | ✅ | ✅ Implementado | Scheduler Worker → Watermill → FCM |
| F06 | Registro de tomada de medicamento | E2 | ✅ | ✅ Implementado | `dose_record_commands.go` — `ConfirmDose`, `MissDose` |
| F07 | Relatórios de adherence | E2 | ❌ | ❌ Não Implementado | Ver RF11 |

---

### 3.4 Requisitos do Documento Anterior (`requirements.md`)

| ID | Descrição | Status Código | Status Docs | Observação |
|----|-----------|---------------|-------------|------------|
| REQ-001 | API REST para CRUD de usuários | ✅ Implementado | ✅ Documentado | |
| REQ-002 | API REST para CRUD de médicos | ✅ Implementado | ✅ Documentado | |
| REQ-003 | API REST para CRUD de prescrições | ✅ Implementado | ✅ Documentado | |
| REQ-004 | Eventos Watermill para domain events | ✅ Implementado | ✅ Documentado | Watermill pub/sub para notificações |
| REQ-005 | Agendamento Redis para notificações | ✅ Implementado | ✅ Documentado | |
| REQ-006 | Integração Firebase FCM para push | ✅ Implementado | ✅ Documentado | |
| REQ-007 | Padrão CQRS (Command/Query separation) | ✅ Implementado | ✅ Documentado | `application/commands/` e `application/queries/` |
| REQ-008 | Clean Architecture | ✅ Implementado | ✅ Documentado | `domain/`, `application/`, `infrastructure/` |
| REQ-009 | Testes unitários com cobertura > 70% | ❌ Não Implementado | ⚠️ Parcial | Cobertura atual ~0% (apenas alguns testes em domain/) |
| REQ-010 | Testes de integração com build tag `integration` | ❌ Não Implementado | ⚠️ Parcial | Existem `*_integration_test.go` mas sem build tag configurada |
| REQ-011 | Documentação PlantUML | ✅ Implementado | ✅ Documentado | |
| REQ-012 | LGPD Privacy by Design | ⚠️ Parcial | 🔲 Não Documentado | Sem código de minimização/anonimização. Ver RNF05 |
| REQ-013 | Interface mobile para confirmação de dose | 🚫 Fora de escopo | ✅ Documentado | Mobile é projeto separado (care-connect) |

---

### 3.5 Gaps do Sprint Scope (`MEZ-29_SPRINT_SCOPE.md`)

| ID | Gap | Prioridade | Código | Status Código | Observação |
|----|-----|-----------|--------|--------------|------------|
| SP-001 | Cancelamento de notificação agendada | P1 | ⚠️ | ⚠️ Parcial | `RedisScheduler.CancelByPrescriptionID` existe (`redis_scheduler.go:101`) mas **não há endpoint HTTP** para expô-lo. Não há comando handler na camada de aplicação |
| SP-002 | Mecanismo de retry com backoff exponencial | P1 | ❌ | ❌ Não Implementado | `worker.go:224` — `msg.Nack()` sem retry, sem backoff, sem limite de tentativas. Caregiver failures são silenciosamente ignorados |
| SP-003 | Documentação LGPD | P2 | ⚠️ | 🔲 Não Documentado | Ver REQ-012 e RNF05 |

---

### 3.6 Endpoints REST Documentados vs Existentes

| Método | Path | Documentado em `api.yaml` | Existe no Router | Handler |
|--------|------|--------------------------|-----------------|---------|
| GET | `/api/v1/users` | ✅ | ✅ | `ListUsers` |
| POST | `/api/v1/users` | ✅ | ✅ | `CreateUser` |
| GET | `/api/v1/users/{userId}` | ✅ | ✅ | `GetUserById` |
| PUT | `/api/v1/users/{userId}` | ✅ | ✅ | `UpdateUser` |
| DELETE | `/api/v1/users/{userId}` | ✅ | ✅ | `DeleteUser` |
| PATCH | `/api/v1/users/{userId}/firebase-token` | ✅ | ✅ | `UpdateFirebaseToken` |
| PATCH | `/api/v1/users/{userId}/notifications` | ✅ | ✅ | `ToggleNotifications` |
| GET | `/api/v1/doctors` | ✅ | ✅ | `ListDoctors` |
| POST | `/api/v1/doctors` | ✅ | ✅ | `CreateDoctor` |
| GET | `/api/v1/doctors/{doctorId}` | ✅ | ✅ | `GetDoctorById` |
| PUT | `/api/v1/doctors/{doctorId}` | ✅ | ✅ | `UpdateDoctor` |
| DELETE | `/api/v1/doctors/{doctorId}` | ✅ | ✅ | `DeleteDoctor` |
| GET | `/api/v1/prescriptions` | ✅ | ✅ | `ListPrescriptions` |
| POST | `/api/v1/prescriptions` | ✅ | ✅ | `CreatePrescription` |
| GET | `/api/v1/prescriptions/{prescriptionId}` | ✅ | ✅ | `GetPrescriptionById` |
| PUT | `/api/v1/prescriptions/{prescriptionId}` | ✅ | ✅ | `UpdatePrescription` |
| DELETE | `/api/v1/prescriptions/{prescriptionId}` | ✅ | ✅ | `DeletePrescription` |
| POST | `/api/v1/prescriptions/{prescriptionId}/activate` | ✅ | ✅ | `ActivatePrescription` |
| POST | `/api/v1/prescriptions/{prescriptionId}/deactivate` | ✅ | ✅ | `DeactivatePrescription` |
| GET | `/api/v1/health` | ✅ | ✅ | `HealthCheck` |
| POST | `/api/v1/auth/register` | ❌ | ✅ | `ExtendedServer.Register` |
| POST | `/api/v1/auth/login` | ❌ | ✅ | `ExtendedServer.Login` |
| POST | `/api/v1/auth/doctors/register` | ❌ | ✅ | `ExtendedServer.RegisterDoctor` |
| POST | `/api/v1/auth/doctors/login` | ❌ | ✅ | `ExtendedServer.LoginDoctor` |
| POST | `/api/v1/invitations` | ❌ | ✅ | `ExtendedServer.CreateInvitation` |
| GET | `/api/v1/invitations/{token}` | ❌ | ✅ | `ExtendedServer.GetInvitationByToken` |
| POST | `/api/v1/invitations/{token}/accept` | ❌ | ✅ | `ExtendedServer.AcceptInvitation` |
| POST | `/api/v1/invitations/{token}/reject` | ❌ | ✅ | `ExtendedServer.RejectInvitation` |
| GET | `/api/v1/users/{userId}/caregivers` | ❌ | ✅ | `ExtendedServer.ListCaregivers` |
| DELETE | `/api/v1/users/{userId}/caregivers/{caregiverId}` | ❌ | ✅ | `ExtendedServer.UnlinkUsers` |
| GET | `/api/v1/users/{userId}/charges` | ❌ | ✅ | `ExtendedServer.ListCharges` |
| GET | `/api/v1/users/{userId}/invitations` | ❌ | ✅ | `ExtendedServer.ListCaregiverInvitations` |
| GET | `/api/v1/users/{userId}/dose-records` | ❌ | ✅ | `ExtendedServer.ListDoseRecords` |
| POST | `/api/v1/dose-records/{doseRecordId}/confirm` | ❌ | ✅ | `ExtendedServer.ConfirmDose` |
| POST | `/api/v1/dose-records/{doseRecordId}/miss` | ❌ | ✅ | `ExtendedServer.MarkDoseMissed` |

**15 endpoints completamente implementados e roteados mas ausentes do `api.yaml`.**

---

## 4. Gaps por Categoria

### ✅ Implementado e Documentado (conforme requisitos)

- CRUD de usuários, médicos, prescrições (REQ-001, 002, 003)
- Eventos Watermill (REQ-004)
- Agendamento Redis (REQ-005)
- Integração Firebase FCM (REQ-006)
- Padrão CQRS (REQ-007)
- Clean Architecture (REQ-008)
- Sistema de convite cuidador ↔ idoso
- Notificações fan-out para cuidadores vinculados
- Dose record tracking (confirm/miss)
- PlantUML diagrams (class, sequence, architecture, ER, state)

---

### 📄 Implementado mas Não Documentado

| Item | Código | Descrição |
|------|--------|-----------|
| Endpoints estendidos | `extended_server.go` | 15 endpoints (auth, convite, dose records, linked users) completamente implementados mas não aparecem em `api.yaml` |
| NotificationScheduler port | `application/ports.go:10` | Interface `NotificationScheduler` com `Schedule` e `CancelByPrescriptionID` documentada em nenhum lugar |
| Redis cleanup store | `scheduler/cleanup_store.go` | `CleanupStore` interface e `RedisCleanupStore` implementados mas não documentados |
| DoseRecordStore interface | `scheduler/event_store.go` | Interface para criação automática de dose records pelo worker |

---

### ❌ Não Implementado

| Item | Descrição | Impacto |
|------|-----------|---------|
| **Retry mechanism** (SP-002) | Notificações com falha são nack'd sem backoff. Caregiver failures silenciados | Pacientes podem perder notificações sem retry |
| **Cancelamento de notificação** (SP-001) | `RedisScheduler.CancelByPrescriptionID` existe mas não é exposto via HTTP | Não há como cancelar uma notificação já agendada |
| **LGPD compliance** (REQ-012) | Sem minimização de dados, sem anonimização, sem endpoint de export/deletion (Art. 18 LGPD), sem política de retenção | Risco legal — dados de saúde sensíveis sem base legal documentada |
| **Cadastro de farmacêutico** (RF04) | Entidade inexistente | Requisito funcional de E1 não atende |
| **Cadastro de farmácia** (RF05) | Entidade inexistente | Requisito funcional de E1 não atende |
| **Cadastro de unidade de saúde** (RF06) | Entidade inexistente | Requisito funcional de E1 não atende |
| **Relatórios de adherence** (RF11, F07) | Sem agregação de DoseRecord → relatório/percentual | Cuidadores não têm dashboard de adherence |
| **Notificação a médicos sobre adherence** (RF12) | Doctors não recebem alertas de má adherence | Requisito funcional de E1 não atende |
| **Testes unitários** (REQ-009) | Cobertura ~0% | Meta > 70% não atingida |
| **Testes de integração** (REQ-010) | Tests existem mas sem build tag `integration` | Não é possível executar só integração |
| **SMS fallback** | FMEA de E1 menciona SMS fallback como mitigação — não implementado | Se FCM falhar, não há canal alternativo |
| **FCM rate limiting** | MEZ-29 menciona como risk, não implementado | Risco de rate limit pelo FCM |
| **Dead letter queue** | Mensagens com falha em `msg.Nack()` voltam para a fila sem limite | Loop infinito de retry sem DLQ |
| **Doctor → patient linking** | Doctors não têm equivalente de caregiver linking — não é possível listar pacientes de um doctor via API | Não há visão médica do paciente |

---

## 5. Gaps Críticos (Requerem Ação Imediata)

> Ordenados por impacto, do maior para o menor.

| # | Gap | Por que crítico | Ação recomendada |
|---|-----|----------------|-----------------|
| 1 | **Retry mechanism** | Notificações falhadas são perdidas sem qualquer retry. Caregiver failures são completamente ignorados (`log.Printf` only). Nenhuma tolerância a falhas transitórias de FCM | Implementar `NotificationRetryPolicy` com 3 tentativas e backoff exponencial (1min, 5min, 15min). Criar DLQ para notificações que excederem max retries |
| 2 | **LGPD compliance** | Dados de saúde de pacientes sem base legal documentada. Ausência de endpoint de exportação de dados do usuário (LGPD Art. 18) e direito ao apagamento. Requisito marcado como "não documentado" há dois semesters | Criar `docs/PRIVACY.md` documentando minimização de dados, implementar endpoint `DELETE /users/{id}` com soft-delete ou hard-delete completo, adicionar política de retenção |
| 3 | **Cancelamento de notificação** | `RedisScheduler.CancelByPrescriptionID` existe mas é inacessível via API. O único path de cancelamento é `DeactivatePrescription` que também muda o estado `active` da prescrição — comportamento diferente de um cancelamento puro de job | Expor endpoint `DELETE /prescriptions/{id}/notifications` ou similar, ou usar o comando já existente |
| 4 | **Relatórios de adherence** | Sistema não gera nenhum relatório/percentual de adherence. Cuidadores precisam calcular à mão a partir de dose records | Implementar endpoint `GET /users/{id}/adherence` com agregação de `DoseRecord.Status` → percentual |
| 5 | **Testes unitários** | Cobertura ~0%. Bugs podem passar despercebidos. Requisito de disciplina (REQ-009) não atendido | Implementar testes com `testing package` cobrindo domain models e command handlers |

---

## 6. Gaps Secundários (Importantes mas Não Críticos)

| # | Gap | Ação recomendada |
|---|-----|-----------------|
| A | **15 endpoints ausentes do api.yaml** | Atualizar `docs/api.yaml` com os endpoints estendidos ou gerar nova especificação OpenAPI completa |
| B | **SMS fallback** (Twilio) | Implementar `SMSSender` como fallback quando FCM falha, conforme FMEA de E1 |
| C | **FCM rate limiting** | Implementar rate limiter no `FirebaseSender` ou consumer para evitar rate limit do FCM |
| D | **NotificationEventStore** existente mas não utilizado para audit completo | `notification_events` table existe mas não é exposta via API para geração de relatórios |
| E | **Doctor → patient linking** | Doctors criam prescrições mas não têm forma de listar seus pacientes vinculados |
| F | **Notificação a médicos sobre adherence** | RF12 de E1 — quando um paciente tem baixa adherence, notificar o médico que prescreveu |
| G | **Testes de integração** | Adicionar build tag `integration` aos testes de repository e adicionar CI |
| H | **Convite com expiração** | `CaregiverInvitation` não expira — token pode ser usado indefinidamente |

---

## 7. Recomendações e Próximos Passos

### Curto prazo (este sprint)
1. Implementar retry mechanism com backoff exponencial e DLQ
2. Documentar conformidade LGPD em `docs/PRIVACY.md`
3. Expor endpoint de cancelamento de notificação (SP-001)
4. Iniciar cobertura de testes unitários (target: 50% intermediário)

### Médio prazo (próximo release)
1. Atualizar `api.yaml` com todos os 35 endpoints
2. Implementar endpoint de relatório de adherence (`GET /users/{id}/adherence`)
3. Implementar build tag `integration` para testes de integração
4. Implementar SMS fallback (Twilio) como mitigação FMEA

### Longo prazo (v2)
1. Entidade `Pharmacist` e `HealthUnit` (se RF04/RF06 forem reativados no escopo)
2. Doctor → patient linking e notificação de adherence para médicos
3. Política de retenção de dados (anonimização pós-5 anos)

---

## 8. Disclaimer

Este mapeamento reflete o estado do código no branch `docs/diagrams` em 2026-05-10. Alguns requisitos de E1/E2 (farmácia, unidade de saúde, farmacêutico) provavelmente foram intencionalmente removidos do escopo durante o redesign para Go/Clean Architecture, mas nunca foram oficialmente documentados como tal no `requirements.md` original.

---

*Documento gerado por: Lucas Mezencio + Claude (MiniMax M2.7)*
*Fontes consultadas: E1.pdf, E2.pdf, E3_Manuais.docx, careconnect_presentation.html, requirements.md, MEZ-29_SPRINT_SCOPE.md, código fonte em `/internal/`*
