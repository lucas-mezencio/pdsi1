# MEZ-29: Sprint Scope for CareConnect API Gaps

**Issue:** MEZ-29  
**Created:** 2026-05-10  
**Priority:** High  
**Status:** Draft

---

## 1. Objetivo

Definir o escopo do sprint atual para endereçamento das lacunas identificadas no mapeamento de requisitos do CareConnect.

---

## 2. Gaps Identificados (Priorizados)

### 2.1 Cancelamento de Notificação (P1 - Crítico)

**Gap:** Não há mecanismo para cancelar uma notificação agendada antes do envio.

**Impacto:** Pacientes podem receber lembretes desnecessários ou para meds que já foram tomados.

**Recomendação de Solução:**
- Adicionar endpoint `DELETE /scheduled-notifications/{id}` ou `POST /doses/{id}/cancel`
- Implementar remoção do scheduled job no Redis
- Publicar evento `NotificationCancelledEvent` via Watermill

**Tarefas:**
- [ ] Criar `CancelNotificationCommand` no domain
- [ ] Implementar `CancelNotificationHandler` no application layer
- [ ] Adicionar endpoint REST para cancelamento
- [ ] Integrar com Redis para cancelamento do job agendado
- [ ] Adicionar testes unitários

---

### 2.2 Mecanismo de Retry (P1 - Crítico)

**Gap:** Notificações com falha não são retentadas automaticamente.

**Impacto:** Pacientes podem perder lembretes importantes devido a falhas transitórias (timeout, FCM indisponível).

**Recomendação de Solução:**
- Implementar política de retry exponencial (3 tentativas: 1min, 5min, 15min)
- Usar Redis sorted set para tracking de tentativas
- Publicar `NotificationRetryScheduledEvent` após cada falha

**Tarefas:**
- [ ] Criar `RetryNotificationPolicy` no domain
- [ ] Implementar `NotificationRetryScheduler` no infrastructure layer
- [ ] Adicionar lógica de backoff exponencial
- [ ] Criar `NotificationFailedEvent` com metadata de retry
- [ ] Adicionar testes unitários e de integração

---

### 2.3 Conformidade LGPD (P2 - Importante)

**Gap:** Requisito REQ-012 não está documentado: "LGPD Privacy by Design (minimização de dados)".

**Impacto:** Não conformidade legal para dados sensíveis de saúde (prontuários).

**Recomendação de Solução:**
- Documentar minimização de dados em todos os endpoints
- Implementar soft-delete para dados de saúde
- Adicionar consentimento tracking

**Tarefas:**
- [ ] Criar documento de privacidade (PRIVACY.md)
- [ ] Mapear dados sensíveis por endpoint
- [ ] Implementar política de retenção (anonimização após 5 anos)
- [ ] Documentar base legal para cada operação de dados
- [ ] Adicionar endpoint para exportação de dados do usuário (LGPD Art. 18)

---

## 3. Gaps Secundários

### 3.1 Testes Unitários (REQ-009)

**Status:** Não implementado  
**Cobertura atual:** 0%  
**Meta:** > 70%

### 3.2 Testes de Integração (REQ-010)

**Status:** Não implementado  
**Requisito:** Build tag `integration`

---

## 4. Sprint Backlog

| ID | Item | Prioridade | Estimativa |
|----|------|------------|------------|
| SP-001 | Implementar cancelamento de notificação | P1 | 3 dias |
| SP-002 | Implementar mecanismo de retry | P1 | 3 dias |
| SP-003 | Documentar conformidade LGPD | P2 | 2 dias |
| SP-004 | Testes unitários (>70% cobertura) | P2 | 3 dias |
| SP-005 | Testes de integração | P2 | 2 dias |

---

## 5. Critérios de Aceitação do Sprint

- [ ] Endpoint de cancelamento implementado e testado
- [ ] Retry mechanism com 3 tentativas e backoff exponencial
- [ ] Documento PRIVACY.md criado com mapeamento de dados sensíveis
- [ ] Cobertura de testes unitários > 50% (meta intermediária)
- [ ] Testes de integração com build tag `integration` passando

---

## 6. Dependências

- MEZ-27: Work breakdown detalhado para cada feature (referência)
- Infraestrutura Redis e FCM configurada (já existente)

---

## 7. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Redis falha durante retry | Low | High | Fila dead-letter para retry manual |
| FCM rate limiting | Medium | Medium | Implementar rate limiter no publisher |
| LGPD audit complexity | High | Medium | Começar documentação cedo no sprint |

---

*Documento de escopo de sprint - MEZ-29*
