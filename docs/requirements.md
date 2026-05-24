# CareConnect - Especificação de Requisitos de Software (ERS)

## 1. Requisitos de Sistema

O objetivo desta ERS (Especificação de Requisitos de Software) consiste em documentar os requisitos do software CareConnect a ser produzido. Este documento visa garantir que o cliente (instituição de saúde, cuidadores e pacientes idosos) e os desenvolvedores tenham um entendimento comum e inequívoco de todas as funcionalidades, capacidades e restrições do software, servindo como base para o design, desenvolvimento, testes e validação do produto final.

Este documento é destinado a desenvolvedores, testadores, gestores de projeto e demais stakeholders do projeto CareConnect.

### 1.1. Escopo do Produto

O sistema CareConnect tem como propósito **facilitar o acompanhamento e adesão à medicação por parte de idosos**, permitindo que cuidadores e familiares monitorem remotamente o tratamento e recebam alertas em caso de não adesão. O sistema integra dados de prescrição digital (ex.: Memed) e oferece lembretes via notificações push, SMS e ligações telefônicas.

Dentre os principais objetivos destacam-se:
- Permitir o cadastro e gestão de pacientes idosos e seus cuidadores
- Integrar prescrições médicas provenientes de plataformas digitais
- Enviar lembretes de medicação nos horários programados
- Registrar a tomada ou perda de dose dos medicamentos
- Notificar cuidadores em caso de não adesão
- Gerar relatórios de acompanhamento de adesão

O software permitirá ao usuário **cadastrar pacientes**, **cadastrar médicos**, **cadastrar cuidadores**, **registrar prescrições**, **agendar lembretes de medicação automaticamente via prescrição**, **registrar tomada de medicamentos** e **visualizar relatórios de adesão**.

**Escopo Negativo:** Este sistema não realizará agendamento de consultas, registro de médicos no aplicativo (app mobile), busca por hospitais e unidades de saúde, ou processamento de folhas de pagamento.

### 1.2. Requisitos Funcionais

**RF01 - Cadastro de Medicamentos**
- **Nome do Caso de Uso:** Cadastrar Medicamento
- **Brief description:** Permitir que médicos cadastrem medicamentos associados a uma prescrição médica. O sistema deve permitir incluir nome, dosagem, frequência e horário de tomada.
- **Atores envolvidos:** Médico
- **Pré-condições:** Médico estar autenticado no sistema; paciente estar cadastrado
- **Fluxo Principal de Eventos:**
  1. Médico acessa a funcionalidade de prescrição
  2. Seleciona ou cria uma prescrição para o paciente
  3. Adiciona medicamento(s) com: nome, dosagem, frequência (vezes ao dia), horários específicos, quantidade de doses total
  4. Sistema valida os dados
  5. Sistema armazena o medicamento vinculado à prescrição
- **Pós-condições:** Medicamento vinculado à prescrição do paciente; agendamento criado automaticamente para os horários definidos
- **Fluxo Secundário de Eventos:**
  - Médico edita medicamento já cadastrado
  - Médico remove medicamento da prescrição (soft delete)
- **Observações:** Medicamento é atrelado a uma prescrição, não existe isoladamente. O agendamento das doses é criado automaticamente pelo sistema.

---

**RF02 - Cadastro de Pacientes**
- **Nome do Caso de Uso:** Cadastrar Paciente
- **Brief description:** Permitir o cadastro de pacientes idosos no sistema. Cada paciente pode ser vinculado a um ou mais cuidadores que monitoram sua medicação.
- **Atores envolvidos:** Cuidador, Administrador do Sistema
- **Pré-condições:** -
- **Fluxo Principal de Eventos:**
  1. Usuário acessa a funcionalidade de cadastro de paciente
  2. Informa dados do paciente: nome, data de nascimento, contato, histórico de saúde (opcional)
  3. Sistema valida os dados
  4. Sistema armazena o paciente
- **Pós-condições:** Paciente cadastrado e disponível para receber prescrições
- **Fluxo Secundário de Eventos:**
  - Cuidador é vinculado ao paciente
  - Dados do paciente são editados
  - Paciente é desativado
- **Observações:** Paciente é o usuário final que toma os medicamentos. O monitoramento é feito pelos cuidadores vinculados.

---

**RF03 - Cadastro de Médicos**
- **Nome do Caso de Uso:** Cadastrar Médico
- **Brief description:** Permitir o cadastro de médicos no sistema para que possam criar e gerenciar prescrições de medicamentos para seus pacientes.
- **Atores envolvidos:** Médico, Administrador do Sistema
- **Pré-condições:** -
- **Fluxo Principal de Eventos:**
  1. Médico acessa a funcionalidade de cadastro
  2. Informa dados: nome, CRM, especialidade, contato
  3. Sistema valida os dados
  4. Sistema armazena o médico
- **Pós-condições:** Médico cadastrado e capaz de criar prescrições
- **Fluxo Secundário de Eventos:**
  - Dados do médico são editados
  - Médico é desativado
- **Observações:** Médico é o ator que prescreve medicamentos aos pacientes.

---

**RF07 - Cadastro de Cuidadores**
- **Nome do Caso de Uso:** Cadastrar Cuidador
- **Brief description:** Permitir o cadastro de familiares/cuidadores que monitoram a medicação de pacientes idosos. Cuidadores são vinculados a pacientes via convite.
- **Atores envolvidos:** Cuidador, Paciente
- **Pré-condições:** Paciente estar cadastrado
- **Fluxo Principal de Eventos:**
  1. Sistema gera código/token de convite para o paciente
  2. Cuidador recebe o convite
  3. Cuidador aceita o convite e se cadastra no sistema
  4. Sistema vincula o cuidador ao paciente
- **Pós-condições:** Cuidador vinculado ao paciente e capaz de receber notificações sobre a medicação
- **Fluxo Secundário de Eventos:**
  - Cuidador rejeita o convite
  - Cuidador é desvinculado do paciente
  - Cuidador atualiza seus dados
- **Observações:** Um paciente pode ter múltiplos cuidadores vinculados. Cuidadores recebem notificações quando o paciente não toma a medicação.

---

**RF08 - Agendamento Automático de Notificações**
- **Nome do Caso de Uso:** Agendar Notificações Automaticamente
- **Brief description:** Quando uma prescrição é criada ou ativada, o sistema cria automaticamente os agendamentos de notificação com base nos horários e frequência definidos na prescrição.
- **Atores envolvidos:** Sistema (processo automático)
- **Pré-condições:** Prescrição criada com medicamentos e horários definidos
- **Fluxo Principal de Eventos:**
  1. Prescrição é criada ou ativada
  2. Sistema extrai horários e frequência de cada medicamento
  3. Sistema cria agendamentos de notificação para cada dose
- **Pós-condições:** Todos os agendamentos de notificação criados e aguardando execução
- **Fluxo Secundário de Eventos:**
  - Prescrição é editada - agendamentos antigos são cancelados e novos são criados
  - Prescrição é desativada ou cancelada - todos os agendamentos são cancelados
- **Observações:** Não existe um cadastro separado de regras - o sistema deriva os agendamentos automaticamente a partir da prescrição.

---

**RF09 - Envio de Lembretes de Medicamentos**
- **Nome do Caso de Uso:** Enviar Lembretes de Medicamentos
- **Brief description:** Quando um agendamento atinge o horário definido, o sistema envia uma notificação ao paciente e aos cuidadores vinculados lembrando da toma do medicamento.
- **Atores envolvidos:** Sistema (processo automático)
- **Pré-condições:** Agendamento existir e estar ativo
- **Fluxo Principal de Eventos:**
  1. Agendamento atinge horário de notificação
  2. Sistema identifica paciente e cuidadores vinculados
  3. Sistema envia notificação *push* para o dispositivo do paciente
  4. Sistema envia notificação *push* para todos os cuidadores vinculados
- **Pós-condições:** Notificação enviada; registro de notificação criado
- **Fluxo Secundário de Eventos:**
  - Notificação *push* falha - sistema tenta novamente com *backoff* exponencial
  - Notificação continua falhando após máximo de tentativas - mensagem vai para *dead letter queue*
- **Observações:** Notificações são enviadas via *Firebase Cloud Messaging (FCM)*. O sistema deve garantir que falhas em cuidadores sejam tratadas com *retry*.

---

**RF10 - Registro de Uso de Medicamentos**
- **Nome do Caso de Uso:** Registrar Uso de Medicamento
- **Brief description:** Permitir que o paciente (ou cuidador) registre se a dose do medicamento foi tomada ou perdida no horário agendado.
- **Atores envolvidos:** Paciente, Cuidador
- **Pré-condições:** Agendamento existir e estar no horário de tomada
- **Fluxo Principal de Eventos:**
  1. Paciente ou cuidador recebe notificação de lembrete
  2. Usuário confirma que tomou o medicamento (confirmar dose) ou indica que perdeu (registrar falha)
  3. Sistema atualiza o registro de dose com o status correspondente
- **Pós-condições:** Registro de dose atualizado com status de tomado ou perdido
- **Fluxo Secundário de Eventos:**
  - Cuidador registra toma em nome do paciente
  - Usuário não interage - sistema marca automaticamente como perdido após janela de tolerância
- **Observações:** O registro de dose é fundamental para calcular o relatório de adesão (RF11).

---

**RF11 - Relatórios de Adesão**
- **Nome do Caso de Uso:** Gerar Relatório de Adesão
- **Brief description:** Permitir que cuidadores visualizem o percentual de adesão do paciente à medicação em um período definido, baseado nos registros de doses tomadas e perdidas.
- **Atores envolvidos:** Cuidador, Médico
- **Pré-condições:** Registros de doses existirem
- **Fluxo Principal de Eventos:**
  1. Cuidador solicita relatório de adesão para um paciente
  2. Sistema define período do relatório (ex: últimos 30 dias)
  3. Sistema calcula percentual com base nos registros: (doses tomadas / total de doses) * 100
  4. Sistema apresenta relatório com métricas de adesão
- **Pós-condições:** Relatório de adesão apresentado ao solicitante
- **Fluxo Secundário de Eventos:**
  - Relatório é exportado em formato PDF
  - Histórico de relatórios anteriores é disponível
- **Observações:** Relatório deve ser capaz de identificar padrões de não-adesão (horários específicos, dias da semana).

---

**RF12 - Notificar Médicos sobre Adesão**
- **Nome do Caso de Uso:** Notificar Médico sobre Adesão
- **Brief description:** Quando um paciente apresenta baixa adesão à medicação (abaixo de um limiar definido), o sistema deve notificar o médico que prescreveu a medicação.
- **Atores envolvidos:** Sistema (processo automático)
- **Pré-condições:** Relatório de adesão calculado; médico estar vinculado à prescrição
- **Fluxo Principal de Eventos:**
  1. Sistema identifica paciente com adesão abaixo do limiar (ex: 80%)
  2. Sistema identifica médico que criou a prescrição
  3. Sistema envia notificação ao médico sobre a baixa adesão
- **Pós-condições:** Médico notificado sobre baixa adesão do paciente
- **Fluxo Secundário de Eventos:**
  - Médico recebe notificação com link para ver relatório detalhado
  - Médico pode entrar em contato com o paciente ou cuidador
- **Observações:** Notificação ao médico é uma medida de segurança para intervenção precoce em caso de não-adesão.

### 1.3. Requisitos Não-Funcionais

**RNF01 - Disponibilidade 24/7**
- **Categoria:** Requisitos de confiabilidade
- **Descrição:** O sistema backend deve estar disponível 24 horas por dia, 7 dias por semana, para garantir que notificações sejam enviadas pontualmente. Tempo de disponibilidade mínimo: 99% do tempo.
- **Justificativa:** Pacientes dependem de lembretes de medicação em horários específicos. Falhas no sistema podem resultar em perda de doses.

---

**RNF02 - Acessibilidade Mobile**
- **Categoria:** Requisitos de sistema
- **Descrição:** O sistema deve ser acessível via dispositivos móveis (iOS e Android) através do aplicativo CareConnect, permitindo que pacientes e cuidadores acessem suas funcionalidades principais.
- **Justificativa:** O público-alvo principal (idosos e cuidadores) utiliza predominantemente dispositivos móveis para interação com o sistema.

---

**RNF03 - Tempo de Resposta**
- **Categoria:** Requisitos de eficiência
- **Descrição:** O sistema deve processar requisições da API em menos de 5 segundos em condições normais de operação, garantindo resposta ágil para ações de confirmação de dose e consulta de prescrições.
- **Justificativa:** Usuários esperam feedback imediato ao confirmar uma dose ou consultar seu cronograma de medicamentos.

---

**RNF04 - Armazenamento Seguro de Dados**
- **Categoria:** Requisitos de segurança
- **Descrição:** Dados dos usuários devem ser armazenados de forma segura no banco de dados, utilizando práticas de criptografia e controle de acesso. Dados sensíveis (saúde, medicação) devem ter proteção reforçada.
- **Justificativa:** O sistema lida com dados pessoais e de saúde sensíveis - proteção é requisito legal e ético.

---

**RNF05 - Conformidade com LGPD**
- **Categoria:** Requisitos legais
- **Descrição:** O sistema deve operar em conformidade com a Lei Geral de Proteção de Dados (LGPD). Isso inclui: minimização da coleta de dados, direito ao acesso e apagamento dos dados do usuário, e política de retenção de dados.
- **Justificativa:** Dados de saúde são considerados dados sensíveis pela LGPD e exigem base legal específica para tratamento.

---

**RNF06 - Portabilidade do Sistema**
- **Categoria:** Requisitos de portabilidade
- **Descrição:** O sistema backend deve ser desenvolvido para funcionar em diferentes ambientes (desenvolvimento, staging, produção) sem alterações de código, utilizando variáveis de ambiente para configuração.
- **Justificativa:** Facilita o deploy e a manutenção em diferentes infraestrutura.

---

**RNF07 - Interoperabilidade com Sistemas Externos**
- **Categoria:** Requisitos de interoperabilidade
- **Descrição:** O sistema deve ser capaz de integrar com sistemas externos de prescrição digital (ex.: Memed) e plataformas de notificação (ex.: Firebase Cloud Messaging).
- **Justificativa:** A integração com plataformas de prescrição digital é parte do escopo do CareConnect para importação automática de receitas.

---

**RNF08 - Cobertura de Testes**
- **Categoria:** Requisitos de implementação
- **Descrição:** O sistema deve possuir testes unitários com cobertura mínima de 70% do código. Testes de integração devem cobrir os principais fluxos de funcionamento.
- **Justificativa:** Garantia de qualidade e redução de regressões em um sistema crítico de saúde.

### 1.4. Protótipo

O CareConnect é um sistema centrado em notificações e agendamento automático de medicação. Sua proposta de valor principal é a pontualidade e confiabilidade no envio de lembretes, tornando a interação do usuário com o sistema relativamente simples e transparente.

Como o sistema não requer entrada de dados complexa pelo usuário final (paciente/cuidador) - o fluxo principal é a visualização do cronograma e a confirmação de dose - optou-se por não desenvolver protótipos mockup tradicionais. O aplicativo mobile está em desenvolvimento e pode ser consultado para referência visual.

**Arquitetura de Integração de Prescrições:**

O fluxo de prescrições do CareConnect funciona da seguinte forma:
1. Médico cria prescrição em plataforma de prescrição digital (ex.: Memed)
2. Plataforma externa envia prescrição ao backend CareConnect via API REST
3. Backend extrai medicamentos, horários e frequência da prescrição
4. Sistema cria automaticamente os agendamentos de notificação
5. Paciente e cuidadores recebem notificações nos horários programados

**Funcionalidades principais acessíveis ao usuário:**

- **Paciente/Cuidador:** Visualizar cronograma de medicamentos, confirmar dose tomada, visualizar relatórios de adesão
- **Médico:** Cadastrar e gerenciar prescrições (via integração API ou manualmente), receber alertas de baixa adesão
- **Sistema:** Enviar notificações, registrar doses, gerar relatórios de adesão (processos automáticos)

**Sobre a ausência de protótipos:** Os protótipos visuais do aplicativo mobile estão em desenvolvimento no repositório care-connect (React Native).