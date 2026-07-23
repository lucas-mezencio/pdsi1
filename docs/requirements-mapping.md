# CareConnect — Mapeamento de Requisitos

**Projeto:** CareConnect / MedNotify — Backend API (Go)
**Fonte:** `requirements.md` (ERS document)

---

## Legenda

| Status | Significado |
|--------|-------------|
| Implementado | Existe no codigo e funciona |
| Parcialmente | Implementado com lacunas |
| Nao Implementado | Sem codigo correspondente |

---

## Requisitos Funcionais

| ID | Descricao | Status |
|----|-----------|--------|
| RF01 | Cadastro de Medicamentos | Implementado |
| RF02 | Cadastro de Pacientes | Implementado |
| RF03 | Cadastro de Medicos | Implementado |
| RF07 | Cadastro de Cuidadores | Implementado |
| RF08 | Agendamento Automatico de Notificacoes | Implementado |
| RF09 | Envio de Lembretes de Medicamentos | Implementado |
| RF10 | Registro de Uso de Medicamentos | Implementado |
| RF11 | Relatorios de Adesao | Nao Implementado |
| RF12 | Notificar Medicos sobre Adesao | Nao Implementado |

---

## Requisitos Nao-Funcionais

| ID | Descricao | Status |
|----|-----------|--------|
| RNF01 | Disponibilidade 24/7 | Parcialmente |
| RNF03 | Tempo de Resposta < 5s | Parcialmente |
| RNF04 | Armazenamento Seguro | Parcialmente |
| RNF05 | Conformidade LGPD | Nao Implementado |
| RNF06 | Portabilidade | Implementado |
| RNF07 | Interoperabilidade | Implementado |
| RNF08 | Cobertura de Testes > 70% | Nao Implementado |

---

## Gaps Identificados

### Nao Implementados

| Item | Descricao |
|------|-----------|
| RF11 | Relatorios de Adesao — sistema apenas registra doses, sem agregacao/relatorio |
| RF12 | Notificar Medicos sobre Adesao — medicos nao recebem alertas de baixa adesao |
| RNF05 | LGPD — sem minimizacao de dados, sem endpoint de export/deletion (Art. 18) |
| RNF08 | Cobertura de testes < 70% (atualmente ~0%) |

### Parcialmente Implementados

| Item | Descricao |
|------|-----------|
| RNF01 | Disponibilidade depende de infraestrutura/deploy |
| RNF03 | Performance nao foi medida formalmente |
| RNF04 | PostgreSQL com UUIDs, mas sem criptografia em repouso documentada |

---

*Documento gerado em: 2026-05-23*